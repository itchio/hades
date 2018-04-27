package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

func (scope *Scope) ToEq(rec reflect.Value) builder.Eq {
	recEl := rec

	if recEl.Type().Kind() == reflect.Ptr {
		recEl = recEl.Elem()
	}

	if recEl.Type().Kind() != reflect.Struct {
		panic("ToEq needs its argument to be a struct")
	}

	eq := make(builder.Eq)
	for _, sf := range scope.GetModelStruct().StructFields {
		if !sf.IsNormal {
			continue
		}
		eq[sf.DBName] = recEl.FieldByName(sf.Name).Interface()
	}
	return eq
}

func (c *Context) Insert(conn *sqlite.Conn, scope *Scope, rec reflect.Value) error {
	eq := scope.ToEq(rec)
	return c.Exec(conn, builder.Insert(eq).Into(scope.TableName()), nil)
}
