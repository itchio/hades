package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
	"xorm.io/builder"
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

	models := []any{
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

// every field type automigrate accepts must survive a save/scan round trip
func Test_ScanTypeMatrix(t *testing.T) {
	type Label string

	type Gizmo struct {
		ID int64

		U   uint
		U8  uint8
		U32 uint32
		U64 uint64
		I32 int32
		F32 float32
		L   Label

		PI   *int
		PI32 *int32
		PI64 *int64
		PU   *uint
		PF32 *float32
		PB   *bool
		PS   *string
		PL   *Label
	}

	pi := int(-4)
	pi32 := int32(-32)
	pi64 := int64(1 << 40)
	pu := uint(7)
	pf32 := float32(2.5)
	pb := true
	ps := "hello"
	pl := Label("tag")

	withContext(t, []any{&Gizmo{}}, func(conn *sqlite.Conn, c *hades.Context) {
		in := &Gizmo{
			ID: 1,

			U:   3,
			U8:  200,
			U32: 1 << 30,
			U64: 1 << 40,
			I32: -1234,
			F32: 1.5,
			L:   "named",

			PI:   &pi,
			PI32: &pi32,
			PI64: &pi64,
			PU:   &pu,
			PF32: &pf32,
			PB:   &pb,
			PS:   &ps,
			PL:   &pl,
		}
		mtest.Must(t, c.Save(conn, in))

		var out Gizmo
		found, err := c.SelectOne(conn, &out, builder.Eq{"id": 1})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, in, &out)

		// nil pointers round-trip as NULL
		mtest.Must(t, c.Save(conn, &Gizmo{ID: 2}))
		var out2 Gizmo
		found, err = c.SelectOne(conn, &out2, builder.Eq{"id": 2})
		mtest.Must(t, err)
		assert.True(t, found)
		assert.EqualValues(t, &Gizmo{ID: 2}, &out2)
	})
}
