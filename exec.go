package hades

import (
	"time"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades/sqliteutil2"
	"github.com/pkg/errors"
)

type ResultFn func(stmt *sqlite.Stmt) error

func (c *Context) Exec(q Querier, b *builder.Builder, resultFn ResultFn) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return errors.WithStack(err)
	}
	return c.ExecRaw(q, query, resultFn, args...)
}

func (c *Context) ExecWithSearch(q Querier, b *builder.Builder, search Search, resultFn ResultFn) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return errors.WithStack(err)
	}

	query = search.Apply(query)

	return c.ExecRaw(q, query, resultFn, args...)
}

func (c *Context) ExecRaw(q Querier, query string, resultFn ResultFn, args ...interface{}) error {
	c.QueryCount++

	var startTime time.Time
	if c.Log {
		startTime = time.Now()
	}

	err := sqliteutil2.Exec(q, query, resultFn, args...)

	if c.Log {
		c.Consumer.Debugf("[%s] %s %+v", time.Since(startTime), query, args)
		if err != nil {
			c.Consumer.Debugf("error: %+v", err)
		}
	}
	return err
}
