package hades_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

func Test_AfterWrite(t *testing.T) {
	type Trait struct {
		ID     int64
		HeroID int64
		Name   string
	}
	type Hero struct {
		ID     int64
		Name   string
		Traits []*Trait
	}

	models := []any{&Hero{}, &Trait{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		var calls [][]string
		c.AfterWrite = func(tables []string) {
			calls = append(calls, tables)
		}
		take := func() [][]string {
			res := calls
			calls = nil
			return res
		}

		hero := &Hero{ID: 1, Name: "Gwen", Traits: []*Trait{
			{ID: 10, Name: "brave"},
			{ID: 11, Name: "loud"},
		}}

		mtest.Must(t, c.Save(conn, hero, hades.Assoc("Traits")))
		assert.Equal(t, [][]string{{"heros", "traits"}}, take())

		mtest.Must(t, c.Save(conn, hero))
		assert.Equal(t, [][]string{{"heros"}}, take())

		// replacing a has_many deletes from the child table
		hero.Traits = hero.Traits[:1]
		mtest.Must(t, c.Save(conn, hero, hades.AssocReplace("Traits")))
		assert.Equal(t, [][]string{{"heros", "traits"}}, take())

		mtest.Must(t, c.Update(conn, &Trait{},
			hades.Where(builder.Eq{"id": 10}),
			builder.Eq{"name": "bold"},
		))
		assert.Equal(t, [][]string{{"traits"}}, take())

		mtest.Must(t, c.Update(conn, &Trait{},
			hades.Where(builder.Eq{"id": 999}),
			builder.Eq{"name": "absent"},
		))
		mtest.Must(t, c.Delete(conn, &Trait{}, builder.Eq{"id": 999}))
		assert.Empty(t, take())

		mtest.Must(t, c.Delete(conn, &Trait{}, builder.Eq{"id": 10}))
		assert.Equal(t, [][]string{{"traits"}}, take())

		_, err := c.Count(conn, &Hero{}, builder.NewCond())
		mtest.Must(t, err)
		mtest.Must(t, c.ExecRaw(conn, "UPDATE heros SET name = 'raw'", nil))
		assert.Empty(t, take())
	})
}

func Test_AfterWriteFailedSave(t *testing.T) {
	type Pet struct {
		ID   int64
		Name string
	}

	withContext(t, []any{&Pet{}}, func(conn *sqlite.Conn, c *hades.Context) {
		called := false
		c.AfterWrite = func(tables []string) {
			called = true
		}

		assert.Error(t, c.Save(conn, Pet{ID: 1}))
		assert.False(t, called)

		// inside the caller's transaction the call comes before the commit
		func() {
			var err error
			defer sqlitex.Save(conn)(&err)
			mtest.Must(t, c.Save(conn, &Pet{ID: 1, Name: "Rex"}))
			assert.True(t, called)
			assert.False(t, conn.GetAutocommit())
		}()
	})
}

// panicHandler panics the nth time it logs a query containing trigger.
type panicHandler struct {
	trigger string
	nth     int
	seen    *int
}

func (h panicHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h panicHandler) WithAttrs([]slog.Attr) slog.Handler       { return h }
func (h panicHandler) WithGroup(string) slog.Handler            { return h }
func (h panicHandler) Handle(_ context.Context, r slog.Record) error {
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "query" && strings.Contains(a.Value.String(), h.trigger) {
			*h.seen++
			if *h.seen == h.nth {
				panic("logger panicked")
			}
		}
		return true
	})
	return nil
}

func Test_AfterWritePanickedSave(t *testing.T) {
	type Owner struct {
		ID int64
	}

	withContext(t, []any{&Owner{}}, func(conn *sqlite.Conn, c *hades.Context) {
		var calls [][]string
		c.AfterWrite = func(tables []string) {
			calls = append(calls, tables)
		}

		// the first insert queues a notification, the second panics
		logger := c.Logger
		c.Logger = slog.New(panicHandler{trigger: `INTO "owners"`, nth: 2, seen: new(int)})
		assert.Panics(t, func() {
			_ = c.Save(conn, []*Owner{{ID: 1}, {ID: 2}})
		})
		c.Logger = logger

		count, err := c.Count(conn, &Owner{}, builder.NewCond())
		mtest.Must(t, err)
		assert.EqualValues(t, 0, count)
		assert.Empty(t, calls)

		mtest.Must(t, c.Save(conn, &Owner{ID: 3}))
		assert.Equal(t, [][]string{{"owners"}}, calls)
	})
}
