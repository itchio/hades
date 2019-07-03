package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
)

func Test_Update(t *testing.T) {
	type Mistake struct {
		ID   int64
		Body string
	}

	models := []interface{}{&Mistake{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		mistakes := []*Mistake{
			&Mistake{
				ID:   1,
				Body: "rewrote everything",
			},
			&Mistake{
				ID:   2,
				Body: "mostly alone",
			},
		}

		var count int64
		var err error

		mtest.Must(t, c.Save(conn, mistakes))

		count, err = c.Count(conn, &Mistake{}, builder.NewCond())
		mtest.Must(t, err)
		assert.EqualValues(t, 2, count)

		mtest.Must(t, c.Update(conn, &Mistake{},
			hades.Where(builder.Eq{"id": 1}),
			builder.Eq{"body": "rewrote almost everything"},
		))

		var m Mistake
		var found bool
		found, err = c.SelectOne(conn, &m, builder.Eq{"id": 1})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "rewrote almost everything", m.Body)

		mtest.Must(t, c.Update(conn, &Mistake{},
			hades.Where(builder.Expr("1")),
			builder.Eq{"body": "nothing"},
		))

		found, err = c.SelectOne(conn, &m, builder.Eq{"id": 1})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "nothing", m.Body)

		found, err = c.SelectOne(conn, &m, builder.Eq{"id": 2})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "nothing", m.Body)
	})
}
