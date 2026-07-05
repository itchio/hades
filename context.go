package hades

import (
	"log/slog"
)

type Context struct {
	ScopeMap *ScopeMap
	Logger   *slog.Logger
	Error    error

	// secondary indexes registered via DeclareIndex, maintained by AutoMigrate
	indexes []IndexSpec
}

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
