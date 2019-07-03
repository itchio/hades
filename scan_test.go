package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
)

func Test_Scan(t *testing.T) {
	type GameEmbedData struct {
		GameID int64 `hades:"primary_key"`
		Width  int64
		Height int64
	}

	type Game struct {
		ID        int64
		Title     string
		EmbedData *GameEmbedData
	}

	models := []interface{}{
		&Game{},
		&GameEmbedData{},
	}
	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		mtest.Must(t, c.Save(conn, []*Game{
			&Game{
				ID:    24,
				Title: "Jazz Jackrabbit",
				EmbedData: &GameEmbedData{
					Width:  640,
					Height: 480,
				},
			},
			&Game{
				ID:    46,
				Title: "Duke Nukem 2",
				EmbedData: &GameEmbedData{
					Width:  320,
					Height: 240,
				},
			},
		}, hades.Assoc("EmbedData")))

		var rows []struct {
			Game          `hades:"squash"`
			GameEmbedData `hades:"squash"`
		}
		mtest.Must(t, c.ExecWithSearch(conn,
			builder.Select("games.*", "game_embed_data.*").
				From("games").
				LeftJoin("game_embed_data", builder.Expr("game_embed_data.game_id = games.id")),
			hades.Search{}.OrderBy("games.id ASC"),
			c.IntoRowsScanner(&rows),
		))

		assert.EqualValues(t, 2, len(rows))
		assert.EqualValues(t, "Jazz Jackrabbit", rows[0].Game.Title)
		assert.EqualValues(t, 640, rows[0].GameEmbedData.Width)
		assert.EqualValues(t, 480, rows[0].GameEmbedData.Height)
		assert.EqualValues(t, "Duke Nukem 2", rows[1].Game.Title)
		assert.EqualValues(t, 320, rows[1].GameEmbedData.Width)
		assert.EqualValues(t, 240, rows[1].GameEmbedData.Height)
	})
}
