package hades_test

import (
	"reflect"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
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
		wtest.Must(t, c.SaveOne(conn, []*Game{
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
		}))

		type Row struct {
			Game          `hades:"squash"`
			GameEmbedData `hades:"squash"`
		}
		var rows []Row
		rowScope := c.NewScope(&Row{})
		gameTable := c.TableName(&Game{})
		embedTable := c.TableName(&GameEmbedData{})
		wtest.Must(t, c.ExecWithSearch(conn,
			builder.Select(gameTable+".*", embedTable+".*").
				From(gameTable).
				LeftJoin(embedTable, builder.Expr(embedTable+".game_id = "+gameTable+".id")),
			hades.Search().OrderBy(gameTable+".id ASC"),
			func(stmt *sqlite.Stmt) error {
				var row Row
				err := c.Scan(stmt, rowScope.GetStructFields(), reflect.ValueOf(&row).Elem())
				if err != nil {
					return err
				}
				rows = append(rows, row)
				return nil
			},
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
