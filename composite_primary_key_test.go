package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
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

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		mtest.Must(t, c.Save(conn, &Profile{ID: 14}))
		mtest.Must(t, c.Save(conn, &ProfileData{
			ProfileID: 14,
			Key:       "foo",
			Value:     "bar",
		}))

		dataCount, err := c.Count(conn, &ProfileData{}, builder.NewCond())
		mtest.Must(t, err)
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

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		hh := []*Helicopter{
			&Helicopter{A: 1, B: "hey"},
			&Helicopter{A: 1, B: "hey"},
		}
		mtest.Must(t, c.Save(conn, hh))
	})
}