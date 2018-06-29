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

	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}

	q := dbpool.Get(context.Background().Done())
	defer dbpool.Put(q)

	wtest.Must(t, c.ExecRaw(q, "CREATE TABLE honors (id INTEGER PRIMARY KEY, title TEXT)", nil))

	baseHonors := []Honor{
		{ID: 0, Title: "Best Picture"},
		{ID: 1, Title: "Best Supporting Actor"},
		{ID: 2, Title: "Best Muffins"},
		{ID: 3, Title: "Second Best Muffins"},
	}

	for _, h := range baseHonors {
		wtest.Must(t, c.Exec(q, builder.Insert(builder.Eq{"id": h.ID, "title": h.Title}).Into("honors"), nil))
	}

	count, err := c.Count(q, &Honor{}, builder.NewCond())
	wtest.Must(t, err)
	assert.EqualValues(t, 4, count)

	honor := &Honor{}
	found, err := c.SelectOne(q, honor, builder.Eq{"id": 3})
	wtest.Must(t, err)
	assert.True(t, found)

	var honors []*Honor
	wtest.Must(t, c.Select(q, &honors, builder.Gte{"id": 2}, hades.Search{}))
	assert.EqualValues(t, 2, len(honors))

	wtest.Must(t, c.ExecRaw(q, "DROP TABLE honors", nil))

	// ---------

	err = c.Select(q, []Honor{}, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject non-pointer slice")

	err = c.Select(q, &Honor{}, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject pointer to non-slice")

	type NotAModel struct {
		ID int64
	}

	var namSlice []*NotAModel
	err = c.Select(q, &namSlice, builder.Eq{"id": 3}, hades.Search{})
	assert.Error(t, err, "Select must reject pointer to slice of non-models")

	// ---------

	hhh := &Honor{}
	_, err = c.SelectOne(q, &hhh, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must pointer to pointer")

	_, err = c.SelectOne(q, []Honor{}, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must reject slice")

	answer := 42
	_, err = c.SelectOne(q, &answer, builder.Eq{"id": 3})
	assert.Error(t, err, "SelectOne must reject pointer to non-struct")

	nam := &NotAModel{}
	_, err = c.SelectOne(q, nam, builder.Eq{"id": 3})
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

	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}

	q := dbpool.Get(context.Background().Done())
	defer dbpool.Put(q)

	wtest.Must(t, c.ExecRaw(q, "CREATE TABLE androids (id INTEGER PRIMARY KEY, wise BOOLEAN, funny BOOLEAN)", nil))
	defer c.ExecRaw(q, "DROP TABLE androids", nil)

	baseAndroids := []Android{
		Android{ID: 1, Traits: AndroidTraits{Wise: true}},
		Android{ID: 2, Traits: AndroidTraits{Funny: true}},
		Android{ID: 3},
		Android{ID: 4, Traits: AndroidTraits{Wise: true, Funny: true}},
	}

	for _, a := range baseAndroids {
		wtest.Must(t, c.Exec(q, builder.Insert(builder.Eq{"id": a.ID, "wise": a.Traits.Wise, "funny": a.Traits.Funny}).Into("androids"), nil))
	}

	count, err := c.Count(q, &Android{}, builder.NewCond())
	wtest.Must(t, err)
	assert.EqualValues(t, 4, count)

	a := &Android{}
	found, err := c.SelectOne(q, a, builder.Eq{"id": 1})
	wtest.Must(t, err)
	assert.True(t, found)
	assert.EqualValues(t, baseAndroids[0], *a)
}
