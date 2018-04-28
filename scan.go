package hades

import (
	"reflect"
	"time"

	"crawshaw.io/sqlite"
	"github.com/pkg/errors"
)

func (c *Context) Scan(stmt *sqlite.Stmt, structFields []*StructField, result reflect.Value) error {
	for i, sf := range structFields {
		field := result.FieldByName(sf.Name)

		fieldEl := field
		typ := field.Type()
		wasPtr := false

		colTyp := stmt.ColumnType(i)

		if typ.Kind() == reflect.Ptr {
			wasPtr = true
			if colTyp == sqlite.SQLITE_NULL {
				field.Set(reflect.Zero(field.Type()))
				continue
			}

			fieldEl = field.Elem()
			typ = typ.Elem()
		}

		switch typ.Kind() {
		case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int,
			reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:

			val := stmt.ColumnInt64(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetInt(val)
			}
		case reflect.Float64, reflect.Float32:
			val := stmt.ColumnFloat(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetFloat(val)
			}
		case reflect.Bool:
			val := stmt.ColumnInt(i) == 1
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetBool(val)
			}
		case reflect.String:
			val := stmt.ColumnText(i)
			if wasPtr {
				field.Set(reflect.ValueOf(&val))
			} else {
				fieldEl.SetString(val)
			}
		case reflect.Struct:
			if typ == reflect.TypeOf(time.Time{}) {
				text := stmt.ColumnText(i)
				tim, err := time.Parse(time.RFC3339Nano, text)
				if err == nil {
					if wasPtr {
						field.Set(reflect.ValueOf(&tim))
					} else {
						fieldEl.Set(reflect.ValueOf(tim))
					}
				}
				break
			}
			fallthrough
		default:
			return errors.Errorf("For model %s, unknown kind %s for field %s", result.Type(), field.Type().Kind(), sf.Name)
		}
	}
	return nil
}
