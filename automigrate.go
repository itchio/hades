package hades

import (
	"fmt"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/pkg/errors"
)

// TODO: if table already exists, just add fields
func (c *Context) AutoMigrate(conn *sqlite.Conn) error {
	for _, m := range c.ScopeMap.byDBName {
		err := c.syncTable(conn, m.GetModelStruct())
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) syncTable(conn *sqlite.Conn, ms *ModelStruct) (err error) {
	tableName := ms.TableName
	pti, err := c.PragmaTableInfo(conn, tableName)
	if err != nil {
		return err
	}
	if len(pti) == 0 {
		return c.createTable(conn, ms)
	}

	// migrate table in transaction
	defer sqliteutil.Save(conn)(&err)

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 0", nil)
	if err != nil {
		return nil
	}

	oldColumns := make(map[string]PragmaTableInfoRow)
	for _, ptir := range pti {
		oldColumns[ptir.Name] = ptir
	}

	// TODO: don't do anything if already good

	tempName := fmt.Sprintf("__hades_migrate__%s__", tableName)
	err = c.ExecRaw(conn, fmt.Sprintf("CREATE TABLE %s AS SELECT * FROM %s", tempName, tableName), nil)
	if err != nil {
		return nil
	}

	err = c.dropTable(conn, tableName)
	if err != nil {
		return nil
	}

	err = c.createTable(conn, ms)
	if err != nil {
		return err
	}

	var columns []string
	for _, sf := range ms.StructFields {
		if sf.Relationship != nil {
			continue
		}
		if _, ok := oldColumns[sf.DBName]; !ok {
			continue
		}
		columns = append(columns, EscapeIdentifier(sf.DBName))
	}
	var columnList = strings.Join(columns, ",")

	query := fmt.Sprintf("INSERT INTO %s (%s) SELECT %s FROM %s",
		tableName,
		columnList,
		columnList,
		tempName,
	)

	err = c.ExecRaw(conn, query, nil)
	if err != nil {
		return nil
	}

	err = c.dropTable(conn, tempName)
	if err != nil {
		return nil
	}

	err = c.ExecRaw(conn, "PRAGMA foreign_keys = 1", nil)
	if err != nil {
		return nil
	}

	return nil
}

func (c *Context) createTable(conn *sqlite.Conn, ms *ModelStruct) error {
	query := fmt.Sprintf("CREATE TABLE %s", EscapeIdentifier(ms.TableName))
	var columns []string
	var pks []string
	for _, sf := range ms.StructFields {
		if sf.Relationship != nil {
			continue
		}

		var sqliteType string
		switch sf.Struct.Type.Kind() {
		case reflect.Int64, reflect.Bool:
			sqliteType = "INTEGER"
		case reflect.Float64:
			sqliteType = "REAL"
		case reflect.String:
			sqliteType = "TEXT"
		default:
			return errors.Errorf("Unsupported model field type: %v (in model %v)", sf.Struct.Type, ms.ModelType)
		}
		modifier := ""
		if sf.IsPrimaryKey {
			pks = append(pks, sf.DBName)
			modifier = " NOT NULL"
		}
		column := fmt.Sprintf(`%s %s%s`, EscapeIdentifier(sf.DBName), sqliteType, modifier)
		columns = append(columns, column)
	}

	if len(pks) > 0 {
		columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ", ")))
	} else {
		return errors.Errorf("Model %v has no primary keys", ms.ModelType)
	}
	query = fmt.Sprintf("%s (%s)", query, strings.Join(columns, ", "))

	return c.ExecRaw(conn, query, nil)
}

func (c *Context) dropTable(conn *sqlite.Conn, tableName string) error {
	return c.ExecRaw(conn, fmt.Sprintf("DROP TABLE %s", EscapeIdentifier(tableName)), nil)
}
