package hades

import (
	"slices"

	"crawshaw.io/sqlite"
)

// writeBatch collects the tables one Save writes, so AfterWrite hears
// about them once, after the savepoint is released.
type writeBatch struct {
	tables []string
}

// Call right after the statement, while conn.Changes() still describes it.
func (c *Context) wrote(conn *sqlite.Conn, table string) {
	if c.AfterWrite == nil || conn.Changes() == 0 {
		return
	}

	c.batchMu.Lock()
	batch := c.batches[conn]
	if batch != nil && !slices.Contains(batch.tables, table) {
		batch.tables = append(batch.tables, table)
	}
	c.batchMu.Unlock()

	if batch == nil {
		c.AfterWrite([]string{table})
	}
}

func (c *Context) execWrite(conn *sqlite.Conn, table string, query string, args ...any) error {
	err := c.ExecRaw(conn, query, nil, args...)
	if err == nil {
		c.wrote(conn, table)
	}
	return err
}

// Used like sqlitex.Save: defer c.batchWrites(conn)(&err)
func (c *Context) batchWrites(conn *sqlite.Conn) func(err *error) {
	if c.AfterWrite == nil {
		return func(*error) {}
	}

	batch := &writeBatch{}
	c.batchMu.Lock()
	if c.batches == nil {
		c.batches = make(map[*sqlite.Conn]*writeBatch)
	}
	c.batches[conn] = batch
	c.batchMu.Unlock()

	return func(err *error) {
		// sqlitex.Save rolls back on a panic and panics again, leaving
		// *err nil
		p := recover()

		c.batchMu.Lock()
		delete(c.batches, conn)
		c.batchMu.Unlock()

		if p != nil {
			panic(p)
		}

		if *err == nil && len(batch.tables) > 0 {
			// Save walks a map, so the order would otherwise vary
			slices.Sort(batch.tables)
			c.AfterWrite(batch.tables)
		}
	}
}
