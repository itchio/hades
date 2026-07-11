package hades

import (
	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

func (c *Context) Count(conn *sqlite.Conn, model any, cond builder.Cond) (int64, error) {
	ms := c.NewScope(model).GetModelStruct()

	query, args, err := builder.Select("count(*)").From(EscapeIdentifier(ms.TableName)).Where(cond).ToSQL()
	if err != nil {
		return 0, err
	}

	var result int64

	err = c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		result = stmt.ColumnInt64(0)
		return nil
	}, args...)

	if err != nil {
		return 0, err
	}
	return result, nil
}
