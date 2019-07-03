package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/headway/state"
	"github.com/itchio/hades/mtest"
)

func makeConsumer(t *testing.T) *state.Consumer {
	return &state.Consumer{
		OnMessage: func(lvl string, msg string) {
			t.Logf("[%s] %s", lvl, msg)
		},
	}
}

type WithContextFunc func(conn *sqlite.Conn, c *hades.Context)

func withContext(t *testing.T, models []interface{}, f WithContextFunc) {
	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	mtest.Must(t, err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	c, err := hades.NewContext(makeConsumer(t), models...)
	mtest.Must(t, err)
	c.Log = true

	mtest.Must(t, c.AutoMigrate(conn))

	defer func() {
		c.ScopeMap.Each(func(scope *hades.Scope) error {
			return c.ExecRaw(conn, "DROP TABLE "+scope.TableName(), nil)
		})
	}()

	f(conn, c)
}
