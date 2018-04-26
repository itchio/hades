package hades

import (
	"fmt"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

// TODO: if table already exists, just add fields
func (c *Context) AutoMigrate(conn *sqlite.Conn) error {
	for tableName, m := range c.ScopeMap.byDBName {
		ms := m.GetModelStruct()
		query := fmt.Sprintf("CREATE TABLE %s", tableName)
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
			column := fmt.Sprintf("%s %s", sf.DBName, sqliteType)
			columns = append(columns, column)
			if sf.IsPrimaryKey {
				pks = append(pks, sf.DBName)
			}
		}

		if len(pks) > 0 {
			columns = append(columns, fmt.Sprintf("PRIMARY KEY (%s)", strings.Join(pks, ", ")))
		} else {
			return errors.Errorf("Model %v has no primary keys", ms.ModelType)
		}
		query = fmt.Sprintf("%s (%s)", query, strings.Join(columns, ", "))

		err := c.ExecRaw(conn, query, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
