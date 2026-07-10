package hades

import (
	"context"
	"log/slog"
	"time"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"xorm.io/builder"
)

type ResultFn func(stmt *sqlite.Stmt) error

func (c *Context) Exec(conn *sqlite.Conn, b *builder.Builder, resultFn ResultFn) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return err
	}
	return c.ExecRaw(conn, query, resultFn, args...)
}

func (c *Context) ExecWithSearch(conn *sqlite.Conn, b *builder.Builder, search Search, resultFn ResultFn) error {
	query, args, err := b.ToSQL()
	if err != nil {
		return err
	}

	query = search.Apply(query)
	args = append(args, search.orderArgs...)

	return c.ExecRaw(conn, query, resultFn, args...)
}

func (c *Context) ExecRaw(conn *sqlite.Conn, query string, resultFn ResultFn, args ...any) error {
	var startTime time.Time
	debugLogging := c.Logger != nil && c.Logger.Enabled(context.Background(), slog.LevelDebug)
	if debugLogging {
		startTime = time.Now()
	}

	err := sqlitex.Exec(conn, query, resultFn, args...)

	if debugLogging {
		duration := time.Since(startTime)
		attrs := []slog.Attr{
			slog.String("query", query),
			slog.Any("args", args),
			slog.Duration("duration", duration),
		}
		if err != nil {
			attrs = append(attrs, slog.Any("error", err))
		}
		c.Logger.LogAttrs(nil, slog.LevelDebug, "hades query", attrs...)
	}

	return err
}
