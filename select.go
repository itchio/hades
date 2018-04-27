package hades

import (
	"fmt"
	"reflect"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
)

func (c *Context) Select(conn *sqlite.Conn, result interface{}, cond builder.Cond, search *SearchParams) error {
	ms := c.NewScope(result).GetModelStruct()
	columns, fields := c.selectFields(ms)

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = search.Apply(query)

	// TODO: validate types
	resultVal := reflect.ValueOf(result).Elem()

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		el := reflect.New(ms.ModelType)
		err := c.Scan(stmt, fields, el.Elem())
		if err != nil {
			return err
		}
		resultVal.Set(reflect.Append(resultVal, el))
		return nil
	}, args...)
}

//

func (c *Context) SelectOne(conn *sqlite.Conn, result interface{}, cond builder.Cond) error {
	ms := c.NewScope(result).GetModelStruct()
	columns, fields := c.selectFields(ms)

	query, args, err := builder.Select(columns...).From(ms.TableName).Where(cond).ToSQL()
	if err != nil {
		return err
	}
	query = Search().Limit(1).Apply(query)

	// TODO: validate types
	resultVal := reflect.ValueOf(result).Elem()

	return c.ExecRaw(conn, query, func(stmt *sqlite.Stmt) error {
		return c.Scan(stmt, fields, resultVal)
	}, args...)
}

func (c *Context) selectFields(ms *ModelStruct) ([]string, []*StructField) {
	var columns []string
	var fields []*StructField
	for _, sf := range ms.StructFields {
		if sf.Relationship != nil {
			continue
		}
		columns = append(columns, fmt.Sprintf(`%s.%s`, EscapeIdentifier(ms.TableName), EscapeIdentifier(sf.DBName)))
		fields = append(fields, sf)
	}

	return columns, fields
}
