package hades

import (
	"reflect"

	"fmt"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

func (c *Context) Delete(conn *sqlite.Conn, model any, cond builder.Cond) error {
	modelType := reflect.TypeOf(model)

	scope := c.ScopeMap.ByType(modelType)
	if scope == nil {
		return fmt.Errorf("%v is not a model known to this hades context", modelType)
	}

	if cond == builder.NewCond() {
		return fmt.Errorf("refusing to blindly delete all %v without an explicit builder.Expr(\"1\") clause", modelType)
	}

	b := builder.Delete(cond).From(EscapeIdentifier(scope.TableName()))
	query, args, err := b.ToSQL()
	if err != nil {
		return err
	}
	return c.execWrite(conn, scope.TableName(), query, args...)
}
