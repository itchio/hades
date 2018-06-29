package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_CompositePrimaryKey(t *testing.T) {
	type Profile struct {
		ID int64
	}

	type ProfileData struct {
		ProfileID int64  `hades:"primary_key"`
		Key       string `hades:"primary_key"`
		Value     string
	}

	models := []interface{}{
		&Profile{},
		&ProfileData{},
	}

	withContext(t, models, func(q Querier, c *hades.Context) {
		wtest.Must(t, c.Save(q, &Profile{ID: 14}))
		wtest.Must(t, c.Save(q, &ProfileData{
			ProfileID: 14,
			Key:       "foo",
			Value:     "bar",
		}))

		dataCount, err := c.Count(q, &ProfileData{}, builder.NewCond())
		wtest.Must(t, err)
		assert.EqualValues(t, dataCount, 1)
	})
}

func Test_SaveDuplicateCompositePrimaryKeys(t *testing.T) {
	type Helicopter struct {
		A int64  `hades:"primary_key"`
		B string `hades:"primary_key"`
	}

	models := []interface{}{
		&Helicopter{},
	}

	withContext(t, models, func(q Querier, c *hades.Context) {
		hh := []*Helicopter{
			&Helicopter{A: 1, B: "hey"},
			&Helicopter{A: 1, B: "hey"},
		}
		wtest.Must(t, c.Save(q, hh))
	})
}