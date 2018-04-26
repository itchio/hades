package hades

import (
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

func (c *Context) Select(conn *sqlite.Conn, result interface{}, cond builder.Cond, search *SearchParams) error {
	var columns []string
	ms := c.NewScope(result).GetModelStruct()
	for _, sf := range ms.StructFields {
		if sf.Relationship != nil {
			continue
		}
		columns = append(columns, sf.DBName)
	}

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = search.Apply(query)

	// TODO: validate types
	resultVal := reflect.ValueOf(result).Elem()

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		el := reflect.New(ms.ModelType)
		err := c.Scan(stmt, columns, el.Elem())
		if err != nil {
			return err
		}
		resultVal.Set(reflect.Append(resultVal, el))
		return nil
	}, args...)
}

//

func (c *Context) SelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) error {
	var columns []string
	ms := c.NewScope(result).GetModelStruct()
	for _, sf := range ms.StructFields {
		columns = append(columns, sf.DBName)
	}

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = Search().Limit(1).Apply(query)

	// TODO: validate types
	resultVal := reflect.ValueOf(result).Elem()

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		return c.Scan(stmt, columns, resultVal)
	}, args...)
}
