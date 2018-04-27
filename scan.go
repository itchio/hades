package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

func (c *Context) Scan(stmt *sqlite.Stmt, fields []*StructField, result reflect.Value) error {
	for i, sf := range fields {
		field := result.FieldByName(sf.Name)

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
			return errors.Errorf("For model %s, unknown kind %s for field %s", result.Type(), field.Type().Kind(), sf.Name)
		}
	}
	return nil
}
