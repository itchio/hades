package hades

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
)

type AutoMigrateStats struct {
	NumCreated  int64
	NumMigrated int64
	NumCurrent  int64

	NumIndexesCreated int64
	NumIndexesDropped int64
}

func (c *Context) AutoMigrate(conn *sqlite.Conn) error {
	return c.AutoMigrateEx(conn, &AutoMigrateStats{})
}

func (c *Context) AutoMigrateEx(conn *sqlite.Conn, stats *AutoMigrateStats) (err error) {
	// the whole run is one transaction (per-table savepoints nest inside):
	// table rebuilds drop secondary indexes together with the table, and
	// declared indexes are only re-created by ensureIndexes after all
	// tables are synced — without an enclosing transaction, a failed or
	// interrupted run would strand already-committed tables without their
	// indexes (and commit a half-applied multi-table schema change)
	defer sqlitex.Save(conn)(&err)

	for _, m := range c.ScopeMap.byDBName {
		err = c.syncTable(conn, stats, m.GetModelStruct())
		if err != nil {
			return err
		}
	}

	return c.ensureIndexes(conn, stats)
}

func (c *Context) syncTable(conn *sqlite.Conn, stats *AutoMigrateStats, ms *ModelStruct) (err error) {
	tableName := ms.TableName
	pti, err := c.PragmaTableInfo(conn, tableName)
	if err != nil {
		return err
	}
	if len(pti) == 0 {
		stats.NumCreated++
		return c.createTable(conn, ms)
	}

	// migrate table in transaction
	defer sqlitex.Save(conn)(&err)

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 0", nil)
	if err != nil {
		return err
	}

	oldColumns := make(map[string]PragmaTableInfoRow)
	for _, ptir := range pti {
		oldColumns[ptir.Name] = ptir
	}

	numOldCols := len(oldColumns)
	numNewCols := 0
	isMissingCols := false

	ms.EachNormalField(func(sf *StructField) error {
		numNewCols++
		if _, ok := oldColumns[sf.DBName]; !ok {
			isMissingCols = true
		}
		return nil
	})

	if !isMissingCols && numOldCols == numNewCols {
		// all done
		stats.NumCurrent++
		return nil
	}

	stats.NumMigrated++
	tempName := fmt.Sprintf("__hades_migrate__%s__%d__", tableName, time.Now().UnixNano())
	err = c.ExecRaw(conn, fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", tempName, tableName), nil)
	if err != nil {
		return err
	}

	err = c.dropTable(conn, tableName)
	if err != nil {
		return err
	}

	err = c.createTable(conn, ms)
	if err != nil {
		return err
	}

	var columns []string
	ms.EachNormalField(func(sf *StructField) error {
		if _, ok := oldColumns[sf.DBName]; ok {
			columns = append(columns, EscapeIdentifier(sf.DBName))
		}
		return nil
	})
	var columnList = strings.Join(columns, ",")

	query := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s",
		tableName,
		columnList,
		columnList,
		tempName,
	)

	err = c.ExecRaw(conn, query, nil)
	if err != nil {
		return err
	}

	err = c.dropTable(conn, tempName)
	if err != nil {
		return err
	}

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 1", nil)
	if err != nil {
		return err
	}

	return nil
}

func (c *Context) createTable(conn *sqlite.Conn, ms *ModelStruct) error {
	query := fmt.Sprintf("CREATE TABLE %s", EscapeIdentifier(ms.TableName))
	var columns []string
	var pks []string

	err := ms.EachNormalField(func(sf *StructField) error {
		var sqliteType string
		typ := sf.Struct.Type
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
		}

		switch typ.Kind() {
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
			reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
			sqliteType = "INTEGER"
		case reflect.Bool:
			sqliteType = "BOOLEAN"
		case reflect.Float64, reflect.Float32:
			sqliteType = "REAL"
		case reflect.String:
			sqliteType = "TEXT"
		case reflect.Struct:
			if typ == reflect.TypeOf(time.Time{}) {
				sqliteType = "DATETIME"
				break
			}
			fallthrough
		default:
			return fmt.Errorf("Unsupported model field type: %v (in model %v)", sf.Struct.Type, ms.ModelType)
		}
		modifier := ""
		if sf.IsPrimaryKey {
			pks = append(pks, sf.DBName)
			modifier = " NOT NULL"
		}
		column := fmt.Sprintf(`%s %s%s`, EscapeIdentifier(sf.DBName), sqliteType, modifier)
		columns = append(columns, column)
		return nil
	})
	if err != nil {
		return err
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ", ")))
	} else {
		return fmt.Errorf("Model %v has no primary keys", ms.ModelType)
	}
	query = fmt.Sprintf("%s (%s)", query, strings.Join(columns, ", "))

	return c.ExecRaw(conn, query, nil)
}

func (c *Context) dropTable(conn *sqlite.Conn, tableName string) error {
	return c.ExecRaw(conn, fmt.Sprintf("DROP TABLE %s", EscapeIdentifier(tableName)), nil)
}
