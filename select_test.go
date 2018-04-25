package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

type Honor struct {
	ID    int64
	Title string
}

func Test_Select(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, message string) {
			t.Logf("[%s] %s", lvl, message)
		},
	}

	c, err := hades.NewContext(consumer, &Honor{})
	if err != nil {
		panic(err)
	}
	c.Log = true

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		t.Logf("[SQLITE] %d %s", code, string(msg))
	}

	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	wtest.Must(t, c.ExecRaw(conn, "CREATE TABLE honors (id INTEGER PRIMARY KEY, title TEXT)", nil))

	baseHonors := []Honor{
		{ID: 0, Title: "Best Picture"},
		{ID: 1, Title: "Best Supporting Actor"},
		{ID: 2, Title: "Best Muffins"},
		{ID: 3, Title: "Second Best Muffins"},
	}

	for _, h := range baseHonors {
		wtest.Must(t, c.Exec(conn, builder.Insert(builder.Eq{"id": h.ID, "title": h.Title}).Into("honors"), nil))
	}

	count, err := c.Count(conn, &Honor{}, builder.NewCond())
	wtest.Must(t, err)
	assert.EqualValues(t, 4, count)

	honor := &Honor{}
	wtest.Must(t, c.SelectOne(conn, honor, builder.Eq{"id": 3}))

	var honors []*Honor
	wtest.Must(t, c.Select(conn, &honors, builder.Gte{"id": 2}, nil))
	assert.EqualValues(t, 2, len(honors))

	wtest.Must(t, c.ExecRaw(conn, "DROP TABLE honors", nil))
}
