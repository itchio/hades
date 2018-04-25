package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqliteutil"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
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

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		t.Logf("[SQLITE] %d %s", code, string(msg))
	}

	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	err = sqliteutil.Exec(conn, "CREATE TABLE honors (id INTEGER PRIMARY KEY, title TEXT)", nil)

	insHonor := "INSERT INTO honors (id, title) VALUES (?, ?);"

	err = sqliteutil.Exec(conn, insHonor, nil, 0, "Best Picture")
	if err != nil {
		panic(err)
	}

	err = sqliteutil.Exec(conn, insHonor, nil, 1, "Best Supporting Actor")
	if err != nil {
		panic(err)
	}

	err = sqliteutil.Exec(conn, insHonor, nil, 2, "Best Muffins")
	if err != nil {
		panic(err)
	}

	err = sqliteutil.Exec(conn, insHonor, nil, 3, "Second Best Muffins")
	if err != nil {
		panic(err)
	}

	var honors []Honor
	err = c.Select(conn, &honors, "id >= ?", 2)
	if err != nil {
		panic(err)
	}

	assert.EqualValues(t, 2, len(honors))
}
