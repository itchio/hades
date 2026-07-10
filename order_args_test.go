package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"crawshaw.io/sqlite/sqlitex"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

func Test_SelectOrderByArgs(t *testing.T) {
	type Prize struct {
		ID    int64
		Title string
	}

	c, err := hades.NewContext(&Prize{})
	if err != nil {
		panic(err)
	}
	c.Logger = testLogger(t)

	dbpool, err := sqlitex.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}
	defer dbpool.Close()

	conn := dbpool.Get(context.Background())
	defer dbpool.Put(conn)

	mtest.Must(t, c.ExecRaw(conn, "CREATE TABLE prizes (id INTEGER PRIMARY KEY, title TEXT)", nil))

	prizes := []Prize{
		{ID: 1, Title: "Gold"},
		{ID: 2, Title: "Silver"},
		{ID: 3, Title: "Bronze"},
		{ID: 4, Title: "Gold Star"},
	}
	for _, p := range prizes {
		mtest.Must(t, c.Exec(conn, builder.Insert(builder.Eq{"id": p.ID, "title": p.Title}).Into("prizes"), nil))
	}

	ids := func(prizes []*Prize) []int64 {
		var out []int64
		for _, p := range prizes {
			out = append(out, p.ID)
		}
		return out
	}

	// ORDER BY with a bound parameter: exact match ranks first
	var results []*Prize
	mtest.Must(t, c.Select(conn, &results, builder.NewCond(),
		hades.Search{}.OrderBy("(title = ?) desc, id asc", "Bronze")))
	assert.EqualValues(t, []int64{3, 1, 2, 4}, ids(results))

	// WHERE args and ORDER BY args bind positionally: WHERE args first
	results = nil
	mtest.Must(t, c.Select(conn, &results, builder.Expr("id >= ?", 2),
		hades.Search{}.OrderBy("(title = ?) desc, id asc", "Gold Star")))
	assert.EqualValues(t, []int64{4, 2, 3}, ids(results))

	// same plumbing through ExecWithSearch
	var titles []string
	mtest.Must(t, c.ExecWithSearch(conn,
		builder.Select("title").From("prizes").Where(builder.Expr("id >= ?", 2)),
		hades.Search{}.OrderBy("(title = ?) desc, id asc", "Bronze"),
		func(stmt *sqlite.Stmt) error {
			titles = append(titles, stmt.ColumnText(0))
			return nil
		}))
	assert.EqualValues(t, []string{"Bronze", "Silver", "Gold Star"}, titles)
}
