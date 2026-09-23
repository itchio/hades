package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

func (scope *Scope) ToEq(rec reflect.Value) (builder.Eq, error) {
	recEl := rec

	if recEl.Type().Kind() == reflect.Pointer {
		recEl = recEl.Elem()
	}

	if recEl.Type().Kind() != reflect.Struct {
		panic("ToEq needs its argument to be a struct")
	}

	eq := make(builder.Eq)

	var processField func(sf *StructField, val reflect.Value) error
	processField = func(sf *StructField, val reflect.Value) error {
		field := val.FieldByName(sf.Name)
		if sf.IsSquashed {
			for _, nsf := range sf.SquashedFields {
				if err := processField(nsf, field); err != nil {
					return err
				}
			}
		}

		if !sf.IsNormal {
			return nil
		}
		v, err := DBValue(field.Interface())
		if err != nil {
			return fmt.Errorf("field %s of %s: %w", sf.Name, recEl.Type(), err)
		}
		eq[EscapeIdentifier(sf.DBName)] = v
		return nil
	}

	for _, sf := range scope.GetModelStruct().StructFields {
		if err := processField(sf, recEl); err != nil {
			return nil, err
		}
	}
	return eq, nil
}

func (c *Context) Insert(conn *sqlite.Conn, scope *Scope, rec reflect.Value) error {
	eq, err := scope.ToEq(rec)
	if err != nil {
		return err
	}
	query, args, err := builder.Insert(eq).Into(EscapeIdentifier(scope.TableName())).ToSQL()
	if err != nil {
		return err
	}
	return c.execWrite(conn, scope.TableName(), query, args...)
}
