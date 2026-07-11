package hades

import (
	"reflect"

	"fmt"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
)

const maxSqlVars = 900

type QueryFn func(query string) string

// retrieve cached items in a []*SomeModel
// for some reason, reflect.New returns a &[]*SomeModel instead,
// I'm guessing slices can't be interfaces, but pointers to slices can?
func (c *Context) fetchPagedByPK(conn *sqlite.Conn, PKDBName string, keys []any, sliceType reflect.Type, search Search) (reflect.Value, error) {
	// actually defaults to 999, but let's get some breathing room
	result := reflect.New(sliceType)
	resultVal := result.Elem()

	remainingItems := keys

	for len(remainingItems) > 0 {
		var pageSize int
		pageSize = min(len(remainingItems), maxSqlVars)

		pageAddr := reflect.New(sliceType)
		cond := builder.In(EscapeIdentifier(PKDBName), remainingItems[:pageSize]...)

		err := c.Select(conn, pageAddr.Interface(), cond, search)
		if err != nil {
			return result, fmt.Errorf("performing page fetch: %w", err)
		}

		appended := reflect.AppendSlice(resultVal, pageAddr.Elem())
		resultVal.Set(appended)
		remainingItems = remainingItems[pageSize:]
	}

	return result, nil
}
