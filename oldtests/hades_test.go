package hades_test

import (
	"database/sql"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
)

func makeConsumer(t *testing.T) *state.Consumer {
	return &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			t.Logf("[%s] %s", lvl, msg)
		},
	}
}

type WithContextFunc func(q Querier, c *hades.Context)

func withContext(t *testing.T, models []interface{}, f WithContextFunc) {
	db, err := sql.Open("sqlite3", "file:memory:?mode=memory")
	wtest.Must(t, err)
	defer db.Close()

	c, err := hades.NewContext(makeConsumer(t), models...)
	wtest.Must(t, err)
	c.Log = true

	wtest.Must(t, c.AutoMigrate(q))

	defer func() {
		c.ScopeMap.Each(func(scope *hades.Scope) error {
			return c.ExecRaw(q, "DROP TABLE "+scope.TableName(), nil)
		})
	}()

	f(q, c)
}
