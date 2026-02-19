package hades_test

import (
	"context"
	"log/slog"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
)

func testLogger(t *testing.T) *slog.Logger {
	return slog.New(slog.NewTextHandler(&testWriter{t}, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

type testWriter struct {
	t *testing.T
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

type WithContextFunc func(conn *sqlite.Conn, c *hades.Context)

func withContext(t *testing.T, models []any, f WithContextFunc) {
	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	mtest.Must(t, err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	c, err := hades.NewContext(models...)
	mtest.Must(t, err)
	c.Logger = testLogger(t)

	mtest.Must(t, c.AutoMigrate(conn))

	defer func() {
		c.ScopeMap.Each(func(scope *hades.Scope) error {
			return c.ExecRaw(conn, "DROP TABLE "+scope.TableName(), nil)
		})
	}()

	f(conn, c)
}
