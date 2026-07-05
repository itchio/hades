package hades

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

// managedIndexPrefix marks indexes whose lifecycle hades owns. AutoMigrate
// creates declared indexes that are missing, re-creates them after a table
// rebuild (SQLite drops a table's indexes along with the table), and drops
// indexes carrying this prefix that are no longer declared.
//
// Ownership is scoped to the tables registered with the context: a context
// never prunes managed indexes on tables it doesn't know about. Contexts
// that register the same table must agree on that table's declared indexes,
// or their AutoMigrate runs will drop and re-create each other's indexes.
const managedIndexPrefix = "hades_idx_"

// An IndexSpec describes a non-unique secondary index maintained by
// AutoMigrate. Declare specs with Context.DeclareIndex.
type IndexSpec struct {
	TableName string
	Columns   []string
}

// IndexName returns the name of the index in SQLite. The name encodes the
// table and column list, so changing a declaration normally retires the old
// index and creates a new one on the next AutoMigrate. The encoding is not
// injective (identifiers may themselves contain "__"), which is why
// DeclareIndex rejects same-name declarations with different definitions,
// and ensureIndexes verifies an existing index's actual columns instead of
// trusting its name.
func (spec IndexSpec) IndexName() string {
	return managedIndexPrefix + spec.TableName + "__" + strings.Join(spec.Columns, "__")
}

// DeclareIndex registers a secondary index on the given model's table, to
// be created (or re-created) by the next AutoMigrate call. The model must
// be one of the models the context was built with (matched by type, not by
// table name), and columns must be database column names of that model.
func (c *Context) DeclareIndex(model any, columns ...string) error {
	scope := c.ScopeMap.ByModel(model)
	if scope == nil {
		return fmt.Errorf("DeclareIndex: model %T is not registered with this context", model)
	}
	tableName := scope.TableName()
	if len(columns) == 0 {
		return fmt.Errorf("DeclareIndex: no columns given for table %q", tableName)
	}

	valid := make(map[string]bool)
	scope.GetModelStruct().EachNormalField(func(sf *StructField) error {
		valid[sf.DBName] = true
		return nil
	})

	seen := make(map[string]bool)
	for _, col := range columns {
		if !valid[col] {
			return fmt.Errorf("DeclareIndex: table %q has no column %q", tableName, col)
		}
		if seen[col] {
			return fmt.Errorf("DeclareIndex: duplicate column %q for table %q", tableName, col)
		}
		seen[col] = true
	}

	// clone: the caller may alias and later mutate a variadic slice, which
	// would bypass the validation above
	spec := IndexSpec{TableName: tableName, Columns: slices.Clone(columns)}
	for _, other := range c.indexes {
		if other.IndexName() == spec.IndexName() {
			if other.TableName == tableName && slices.Equal(other.Columns, columns) {
				// identical duplicate declaration, harmless
				return nil
			}
			return fmt.Errorf(
				"DeclareIndex: %q on table %q collides with declaration %v on table %q — identifiers containing \"__\" produce ambiguous index names",
				spec.IndexName(), tableName, other.Columns, other.TableName)
		}
	}

	c.indexes = append(c.indexes, spec)
	return nil
}

// indexColumns returns the columns of an existing index, in index order.
func (c *Context) indexColumns(conn *sqlite.Conn, name string) ([]string, error) {
	type row struct {
		seqno int64
		col   string
	}
	var rows []row
	err := c.ExecRaw(conn,
		fmt.Sprintf("PRAGMA index_info(%s)", EscapeIdentifier(name)),
		func(stmt *sqlite.Stmt) error {
			rows = append(rows, row{seqno: stmt.ColumnInt64(0), col: stmt.ColumnText(2)})
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].seqno < rows[j].seqno })
	cols := make([]string, len(rows))
	for i, r := range rows {
		cols[i] = r.col
	}
	return cols, nil
}

// ensureIndexes converges the hades-managed indexes on this context's
// tables to the declared set. AutoMigrate calls this after all tables have
// been synced, so indexes dropped by a table rebuild are re-created in the
// same run. Managed indexes on tables outside the context's scope map are
// left alone.
func (c *Context) ensureIndexes(conn *sqlite.Conn, stats *AutoMigrateStats) (err error) {
	// converge indexes in transaction, so a failure mid-run can't leave an
	// index dropped without its replacement
	defer sqlitex.Save(conn)(&err)

	// managed index name -> table it actually lives on
	existing := make(map[string]string)
	err = c.ExecRaw(conn,
		"SELECT name, tbl_name FROM sqlite_master WHERE type = 'index'",
		func(stmt *sqlite.Stmt) error {
			name := stmt.ColumnText(0)
			tblName := stmt.ColumnText(1)
			if strings.HasPrefix(name, managedIndexPrefix) && c.ScopeMap.ByDBName(tblName) != nil {
				existing[name] = tblName
			}
			return nil
		},
	)
	if err != nil {
		return err
	}

	declared := make(map[string]bool)
	for _, spec := range c.indexes {
		name := spec.IndexName()
		declared[name] = true
		if tblName, ok := existing[name]; ok {
			// names are not injective encodings of the definition (see
			// IndexName), and the database may have drifted; trust the
			// actual table and indexed columns, not the name
			if tblName == spec.TableName {
				actualCols, err := c.indexColumns(conn, name)
				if err != nil {
					return err
				}
				if slices.Equal(actualCols, spec.Columns) {
					continue
				}
			}
			err = c.ExecRaw(conn, fmt.Sprintf("DROP INDEX %s", EscapeIdentifier(name)), nil)
			if err != nil {
				return err
			}
			stats.NumIndexesDropped++
		}

		cols := make([]string, len(spec.Columns))
		for i, col := range spec.Columns {
			cols[i] = EscapeIdentifier(col)
		}
		query := fmt.Sprintf("CREATE INDEX %s ON %s (%s)",
			EscapeIdentifier(name),
			EscapeIdentifier(spec.TableName),
			strings.Join(cols, ", "),
		)
		err = c.ExecRaw(conn, query, nil)
		if err != nil {
			return err
		}
		stats.NumIndexesCreated++
	}

	for name := range existing {
		if !declared[name] {
			err = c.ExecRaw(conn, fmt.Sprintf("DROP INDEX %s", EscapeIdentifier(name)), nil)
			if err != nil {
				return err
			}
			stats.NumIndexesDropped++
		}
	}

	return nil
}
