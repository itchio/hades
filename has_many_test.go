package hades_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-xorm/builder"
	"github.com/stretchr/testify/assert"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
)

func Test_HasManyThorough(t *testing.T) {
	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

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

	ordie(c.AutoMigrate(conn))

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

	traitCount, err := c.Count(conn, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, 0, traitCount, "no traits should exist before save")

	t.Logf("...snip tons of INSERT...")
	c.Log = false
	ordie(c.Save(conn, &hades.SaveParams{
		Record: car,
		Assocs: []string{"Traits"},
	}))
	c.Log = true

	numTraits := len(car.Traits)

	traitCount, err = c.Count(conn, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, numTraits, traitCount, "all traits should exist after save")

	car.Traits = nil

	ordie(c.Save(conn, &hades.SaveParams{
		Record:   car,
		Assocs:   []string{"Traits"},
		DontCull: []interface{}{&Trait{}},
	}))

	traitCount, err = c.Count(conn, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, numTraits, traitCount, "traits should still exist after partial-join save")

	ordie(c.Save(conn, &hades.SaveParams{
		Record: car,
		Assocs: []string{"Traits"},
	}))

	traitCount, err = c.Count(conn, &Trait{}, builder.NewCond())
	ordie(err)
	assert.EqualValues(t, 0, traitCount, "no traits should exist after last save")
}
