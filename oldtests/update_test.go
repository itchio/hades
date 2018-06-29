package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_Update(t *testing.T) {
	type Mistake struct {
		ID   int64
		Body string
	}

	models := []interface{}{&Mistake{}}

	withContext(t, models, func(q Querier, c *hades.Context) {
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

		wtest.Must(t, c.Save(q, mistakes))

		count, err = c.Count(q, &Mistake{}, builder.NewCond())
		wtest.Must(t, err)
		assert.EqualValues(t, 2, count)

		wtest.Must(t, c.Update(q, &Mistake{},
			hades.Where(builder.Eq{"id": 1}),
			builder.Eq{"body": "rewrote almost everything"},
		))

		var m Mistake
		var found bool
		found, err = c.SelectOne(q, &m, builder.Eq{"id": 1})
		wtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "rewrote almost everything", m.Body)

		wtest.Must(t, c.Update(q, &Mistake{},
			hades.Where(builder.Expr("1")),
			builder.Eq{"body": "nothing"},
		))

		found, err = c.SelectOne(q, &m, builder.Eq{"id": 1})
		wtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "nothing", m.Body)

		found, err = c.SelectOne(q, &m, builder.Eq{"id": 2})
		wtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "nothing", m.Body)
	})
}
