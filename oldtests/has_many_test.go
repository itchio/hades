package hades_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-xorm/builder"
	"github.com/stretchr/testify/assert"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
)

func Test_HasMany(t *testing.T) {
	type Quality struct {
		ID           int64
		ProgrammerID int64
		Label        string
	}

	type Programmer struct {
		ID        int64
		Qualities []*Quality
	}

	models := []interface{}{&Quality{}, &Programmer{}}
	withContext(t, models, func(q Querier, c *hades.Context) {
		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			count, err := c.Count(q, model, builder.NewCond())
			wtest.Must(t, err)
			assert.EqualValues(t, expectedCount, count)
		}

		p1 := &Programmer{
			ID: 3,
			Qualities: []*Quality{
				{ID: 9, Label: "Inspiration"},
				{ID: 10, Label: "Creativity"},
				{ID: 11, Label: "Ability to not repeat oneself"},
			},
		}
		wtest.Must(t, c.Save(q, p1, hades.Assoc("Qualities")))
		assertCount(&Programmer{}, 1)
		assertCount(&Quality{}, 3)

		p1.Qualities[2].Label = "Inspiration again"
		wtest.Must(t, c.Save(q, p1, hades.Assoc("Qualities")))
		assertCount(&Programmer{}, 1)
		assertCount(&Quality{}, 3)
		{
			q := &Quality{}
			found, err := c.SelectOne(q, q, builder.Eq{"id": 11})
			wtest.Must(t, err)
			assert.True(t, found)
			assert.EqualValues(t, "Inspiration again", q.Label)
		}

		p2 := &Programmer{
			ID: 8,
			Qualities: []*Quality{
				{ID: 40, Label: "Peace"},
				{ID: 41, Label: "Serenity"},
			},
		}
		programmers := []*Programmer{p1, p2}
		wtest.Must(t, c.Save(q, programmers, hades.Assoc("Qualities")))
		assertCount(&Programmer{}, 2)
		assertCount(&Quality{}, 5)

		p1bis := &Programmer{ID: 3}
		wtest.Must(t, c.Preload(q, p1bis, hades.Assoc("Qualities")))
		assert.EqualValues(t, 3, len(p1bis.Qualities), "preload has_many")

		wtest.Must(t, c.Preload(q, p1bis, hades.Assoc("Qualities")))
		assert.EqualValues(t, 3, len(p1bis.Qualities), "preload replaces, doesn't append")

		wtest.Must(t, c.Preload(q, p1bis,
			hades.AssocWithSearch("Qualities", hades.Search{}.OrderBy("id ASC"))),
		)
		assert.EqualValues(t, "Inspiration", p1bis.Qualities[0].Label, "orders by (asc)")

		wtest.Must(t, c.Preload(q, p1bis,
			hades.AssocWithSearch("Qualities", hades.Search{}.OrderBy("id DESC"))),
		)
		assert.EqualValues(t, "Inspiration again", p1bis.Qualities[0].Label, "orders by (desc)")

		// no fields
		assert.Error(t, c.Preload(q, p1bis))

		// not a model
		assert.Error(t, c.Preload(q, 42, hades.Assoc("Qualities")))

		// non-existent relation
		assert.Error(t, c.Preload(q, 42, hades.Assoc("Woops")))
	})
}

func Test_HasManyThorough(t *testing.T) {
	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	q := dbpool.Get(context.Background().Done())
	defer dbpool.Put(q)

	type Trait struct {
		ID    int64
		CarID int64
		Label string
	}

	type Car struct {
		ID     int64
		Traits []*Trait
	}

	models := []interface{}{&Car{}, &Trait{}}

	c, err := hades.NewContext(makeConsumer(t), models...)
	ordie(err)
	c.Log = true

	ordie(c.AutoMigrate(q))

	// let's be terrible
	car := &Car{ID: 123}

	// the goal here is to go above SQLite's 999 variables limit
	for i := 0; i < 1300; i++ {
		car.Traits = append(car.Traits, &Trait{
			ID:    int64(i),
			CarID: car.ID,
			Label: fmt.Sprintf("car-trait-#%d", i),
		})
	}

	traitCount, err := c.Count(q, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, 0, traitCount, "no traits should exist before save")

	t.Logf("...snip tons of INSERT...")
	c.Log = false
	ordie(c.Save(q, car, hades.Assoc("Traits")))
	c.Log = true

	numTraits := len(car.Traits)

	traitCount, err = c.Count(q, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, numTraits, traitCount, "all traits should exist after save")

	car.Traits = nil

	ordie(c.Save(q, car, hades.Assoc("Traits")))

	traitCount, err = c.Count(q, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, numTraits, traitCount, "traits should still exist after partial-join save")

	ordie(c.Save(q, car, hades.AssocReplace("Traits")))

	traitCount, err = c.Count(q, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, 0, traitCount, "no traits should exist after last save")
}
