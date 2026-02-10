package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

type WhereCond interface {
	Cond() builder.Cond
}

type whereImpl struct {
	cond builder.Cond
}

func (wt whereImpl) Cond() builder.Cond {
	return wt.cond
}

func Where(cond builder.Cond) WhereCond {
	return whereImpl{cond: cond}
}

func (c *Context) Update(conn *sqlite.Conn, model any, where WhereCond, updates ...builder.Cond) error {
	modelType := reflect.TypeOf(model)
	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return fmt.Errorf("%v is not a know model type", modelType)
	}

	tableName := scope.TableName()
	b := builder.Update(updates...).Where(where.Cond()).From(tableName)
	return c.Exec(conn, b, nil)
}
