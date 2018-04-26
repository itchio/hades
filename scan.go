package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

func (c *Context) Scan(stmt *sqlite.Stmt, columns []string, result reflect.Value) error {
	for i, c := range columns {
		// FIXME: that's bad/slow
		fieldName := FromDBName(c)
		field := result.FieldByName(fieldName)

		switch field.Type().Kind() {
		case reflect.Int64:
			field.SetInt(stmt.ColumnInt64(i))
		case reflect.Float64:
			field.SetFloat(stmt.ColumnFloat(i))
		case reflect.Bool:
			field.SetBool(stmt.ColumnInt(i) == 1)
		case reflect.String:
			field.SetString(stmt.ColumnText(i))
		default:
			return errors.Errorf("For model %s, unknown kind %s for field %s", result.Type(), field.Type().Kind(), fieldName)
		}
	}
	return nil
}
