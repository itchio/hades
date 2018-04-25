package hades

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
)

func (c *Context) Select(conn *sqlite.Conn, result interface{}, where string, args ...interface{}) error {
	s := c.NewScope(result)

	var columns []string
	ms := s.GetModelStruct()
	for _, sf := range ms.StructFields {
		columns = append(columns, sf.DBName)
	}

	query := fmt.Sprintf("select %s from %s where %s",
		strings.Join(columns, ", "),
		s.TableName(),
		where)
	log.Printf("query = %s", query)

	args = append([]interface{}{}, args...)

	resultVal := reflect.ValueOf(result).Elem()

	sqliteutil.Exec(conn, query, func(stmt *sqlite.Stmt) error {
		el := reflect.Zero(ms.ModelType)
		for _, sf := range ms.StructFields {
			el.FieldByName(sf.Name)
			columns = append(columns, sf.Name)
		}

		resultVal.Set(reflect.Append(resultVal, el))

		return nil
	}, args...)
	return nil
}
