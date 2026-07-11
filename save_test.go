package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
)

func Test_Save(t *testing.T) {
	type Game struct {
		ID    int64
		Title string
	}

	type CollectionGame struct {
		ProfileID int64 `hades:"primary_key"`
		GameID    int64 `hades:"primary_key"`
	}

	type Profile struct {
		ID              int64
		CollectionGames []*CollectionGame
	}

	models := []any{
		&Game{},
		&CollectionGame{},
		&Profile{},
	}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		p := &Profile{
			ID: 1,
		}
		mtest.Must(t, c.Save(conn, p))
	})
}

// a primary key column named after an SQL keyword must survive the upsert
// path, which interpolates PK names into ON CONFLICT(...)
func Test_UpsertKeywordPrimaryKey(t *testing.T) {
	type Setting struct {
		Group string `hades:"primary_key"`
		Val   string
	}

	withContext(t, []any{&Setting{}}, func(conn *sqlite.Conn, c *hades.Context) {
		mtest.Must(t, c.Save(conn, &Setting{Group: "display", Val: "dark"}))
		// second save of the same PK exercises ON CONFLICT("group") DO UPDATE
		mtest.Must(t, c.Save(conn, &Setting{Group: "display", Val: "light"}))

		var out Setting
		found, err := c.SelectOne(conn, &out, builder.Eq{`"group"`: "display"})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "light", out.Val)
	})
}
