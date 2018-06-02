package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_BelongsTo(t *testing.T) {
	type Fate struct {
		ID   int64
		Desc string
	}

	type Human struct {
		ID     int64
		FateID int64
		Fate   *Fate
	}

	type Joke struct {
		ID      string
		HumanID int64
		Human   *Human
	}

	models := []interface{}{&Human{}, &Fate{}, &Joke{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		someFate := &Fate{
			ID:   123,
			Desc: "Consumer-grade flamethrowers",
		}
		t.Log("Saving one fate")
		wtest.Must(t, c.Save(conn, someFate))

		lea := &Human{
			ID:     3,
			FateID: someFate.ID,
		}
		t.Log("Saving one human")
		wtest.Must(t, c.Save(conn, lea))

		t.Log("Preloading lea")
		c.Preload(conn, lea, hades.Assoc("Fate"))

		assert.NotNil(t, lea.Fate)
		assert.EqualValues(t, someFate.Desc, lea.Fate.Desc)
	})

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		lea := &Human{
			ID: 3,
			Fate: &Fate{
				ID:   421,
				Desc: "Book authorship",
			},
		}
		wtest.Must(t, c.Save(conn, lea, hades.Assoc("Fate")))

		fate := &Fate{}
		found, err := c.SelectOne(conn, fate, builder.Eq{"id": 421})
		wtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, "Book authorship", fate.Desc)
	})

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		fate := &Fate{
			ID:   3,
			Desc: "Space rodeo",
		}
		wtest.Must(t, c.Save(conn, fate))

		human := &Human{
			ID:     6,
			FateID: 3,
		}
		wtest.Must(t, c.Save(conn, human))

		joke := &Joke{
			ID:      "neuf",
			HumanID: 6,
		}
		wtest.Must(t, c.Save(conn, joke))

		c.Preload(conn, &hades.PreloadParams{
			Record: joke,
			Fields: []hades.PreloadField{
				{Name: "Human"},
				{Name: "Human.Fate"},
			},
		})
		assert.NotNil(t, joke.Human)
		assert.NotNil(t, joke.Human.Fate)
		assert.EqualValues(t, "Space rodeo", joke.Human.Fate.Desc)
	})
}
