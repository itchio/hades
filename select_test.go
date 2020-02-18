package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/itchio/headway/state"
	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

func Test_Select(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, message string) {
			t.Logf("[%s] %s", lvl, message)
		},
	}

	type Honor struct {
		ID    int64
		Title string
	}

	c, err := hades.NewContext(consumer, &Honor{})
	if err != nil {
		panic(err)
	}
	c.Log = true

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		t.Logf("[SQLITE] %d %s", code, string(msg))
	}

	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	mtest.Must(t, c.ExecRaw(conn, "CREATE TABLE honors (id INTEGER PRIMARY KEY, title TEXT)", nil))

	baseHonors := []Honor{
		{ID: 0, Title: "Best Picture"},
		{ID: 1, Title: "Best Supporting Actor"},
		{ID: 2, Title: "Best Muffins"},
		{ID: 3, Title: "Second Best Muffins"},
	}

	for _, h := range baseHonors {
		mtest.Must(t, c.Exec(conn, builder.Insert(builder.Eq{"id": h.ID, "title": h.Title}).Into("honors"), nil))
	}

	count, err := c.Count(conn, &Honor{}, builder.NewCond())
	mtest.Must(t, err)
	assert.EqualValues(t, 4, count)

	honor := &Honor{}
	found, err := c.SelectOne(conn, honor, builder.Eq{"id": 3})
	mtest.Must(t, err)
	assert.True(t, found)

	var honors []*Honor
	mtest.Must(t, c.Select(conn, &honors, builder.Gte{"id": 2}, hades.Search{}))
	assert.EqualValues(t, 2, len(honors))

	mtest.Must(t, c.ExecRaw(conn, "DROP TABLE honors", nil))

	// ---------

	err = c.Select(conn, []Honor{}, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject non-pointer slice")

	err = c.Select(conn, &Honor{}, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject pointer to non-slice")

	type NotAModel struct {
		ID int64
	}

	var namSlice []*NotAModel
	err = c.Select(conn, &namSlice, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject pointer to slice of non-models")

	// ---------

	hhh := &Honor{}
	_, err = c.SelectOne(conn, &hhh, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must pointer to pointer")

	_, err = c.SelectOne(conn, []Honor{}, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must reject slice")

	answer := 42
	_, err = c.SelectOne(conn, &answer, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must reject pointer to non-struct")

	nam := &NotAModel{}
	_, err = c.SelectOne(conn, nam, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must reject pointer to non-struct")
}

func Test_SelectSquashed(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, message string) {
			t.Logf("[%s] %s", lvl, message)
		},
	}

	type AndroidTraits struct {
		Wise  bool
		Funny bool
	}

	type Android struct {
		ID     int64
		Traits AndroidTraits `hades:"squash"`
	}

	c, err := hades.NewContext(consumer, &Android{})
	if err != nil {
		panic(err)
	}
	c.Log = true

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		t.Logf("[SQLITE] %d %s", code, string(msg))
	}

	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	mtest.Must(t, c.ExecRaw(conn, "CREATE TABLE androids (id INTEGER PRIMARY KEY, wise BOOLEAN, funny BOOLEAN)", nil))
	defer c.ExecRaw(conn, "DROP TABLE androids", nil)

	baseAndroids := []Android{
		Android{ID: 1, Traits: AndroidTraits{Wise: true}},
		Android{ID: 2, Traits: AndroidTraits{Funny: true}},
		Android{ID: 3},
		Android{ID: 4, Traits: AndroidTraits{Wise: true, Funny: true}},
	}

	for _, a := range baseAndroids {
		mtest.Must(t, c.Exec(conn, builder.Insert(builder.Eq{"id": a.ID, "wise": a.Traits.Wise, "funny": a.Traits.Funny}).Into("androids"), nil))
	}

	count, err := c.Count(conn, &Android{}, builder.NewCond())
	mtest.Must(t, err)
	assert.EqualValues(t, 4, count)

	a := &Android{}
	found, err := c.SelectOne(conn, a, builder.Eq{"id": 1})
	mtest.Must(t, err)
	assert.True(t, found)
	assert.EqualValues(t, baseAndroids[0], *a)
}
