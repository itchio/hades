package hades

import (
	"log/slog"
	"sync"

	"crawshaw.io/sqlite"
)

type Context struct {
	ScopeMap *ScopeMap
	Logger   *slog.Logger
	Error    error

	// AfterWrite, if set, is called with the tables whose rows were changed
	// by a Save, Update, Delete, Insert or Upsert, once per call and only
	// when it succeeded. A Save reports every table it touched, including
	// associations and join tables, sorted, after releasing its savepoint.
	// Statements run through Exec and ExecRaw are not reported.
	//
	// It runs on the goroutine that did the write. If the caller has its
	// own transaction open on the connection, that is before the commit,
	// when other connections can't see the change yet. Set it before
	// sharing the context across goroutines.
	AfterWrite func(tables []string)

	batchMu sync.Mutex
	batches map[*sqlite.Conn]*writeBatch

	// secondary indexes registered via DeclareIndex, maintained by AutoMigrate
	indexes []IndexSpec
}

// NewContext builds a context over the given models. Register all models
// before sharing the context across goroutines: metadata for a type with
// relationship fields is built lazily on first use and that construction
// is not goroutine-safe. Squash-only row structs are safe to first-use
// concurrently.
func NewContext(models ...any) (*Context, error) {
	c := &Context{
		ScopeMap: NewScopeMap(),
	}

	for _, m := range models {
		err := c.ScopeMap.Add(c, m)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (c *Context) TableName(model any) string {
	return c.NewScope(model).TableName()
}

func (c *Context) NewScope(value any) *Scope {
	return &Scope{
		Value: value,
		ctx:   c,
	}
}

func (c *Context) AddError(err error) {
	c.Error = err
}
