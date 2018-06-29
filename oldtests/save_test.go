package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
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

	models := []interface{}{
		&Game{},
		&CollectionGame{},
		&Profile{},
	}

	withContext(t, models, func(q Querier, c *hades.Context) {
		p := &Profile{
			ID: 1,
		}
		wtest.Must(t, c.Save(q, p))
	})
}
