package hades_test

import (
	"context"
	"testing"

	"github.com/go-xorm/builder"
	"github.com/stretchr/testify/assert"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
)

type Language struct {
	ID    int64
	Words []*Word `hades:"many_to_many:language_words"`
}

type Word struct {
	ID        string
	Comment   string
	Languages []*Language `hades:"many_to_many:language_words"`
}

type LanguageWord struct {
	LanguageID int64  `hades:"primary_key"`
	WordID     string `hades:"primary_key"`
}

func Test_ManyToMany(t *testing.T) {
	models := []interface{}{&Language{}, &Word{}, &LanguageWord{}}
	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		fr := &Language{
			ID: 123,
			Words: []*Word{
				{ID: "Plume"},
				{ID: "Week-end"},
			},
		}
		t.Logf("saving just fr")
		wtest.Must(t, c.Save(conn, fr, hades.Assoc("Words")))

		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			count, err := c.Count(conn, model, builder.NewCond())
			wtest.Must(t, err)
			assert.EqualValues(t, expectedCount, count)
		}
		assertCount(&Language{}, 1)
		assertCount(&Word{}, 2)
		assertCount(&LanguageWord{}, 2)

		en := &Language{
			ID: 456,
			Words: []*Word{
				{ID: "Plume"},
				{ID: "Week-end"},
			},
		}
		t.Logf("saving fr+en")
		wtest.Must(t, c.Save(conn, []*Language{fr, en}, hades.Assoc("Words")))
		assertCount(&Language{}, 2)
		assertCount(&Word{}, 2)
		assertCount(&LanguageWord{}, 4)

		t.Logf("saving without culling ('add' words to english)")
		en.Words = []*Word{
			{ID: "Wreck"},
			{ID: "Nervous"},
		}
		wtest.Must(t, c.Save(conn, []*Language{en}, hades.Assoc("Words")))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 6)

		t.Logf("replacing all english words")
		wtest.Must(t, c.Save(conn, []*Language{en}, hades.AssocReplace("Words")))

		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 4)

		t.Logf("adding commentary")
		en.Words[0].Comment = "punk band reference"
		wtest.Must(t, c.Save(conn, []*Language{en}, hades.Assoc("Words")))
		assertCount(&Language{}, 2)
		assertCount(&Word{}, 4)
		assertCount(&LanguageWord{}, 4)

		{
			w := &Word{}
			found, err := c.SelectOne(conn, w, builder.Eq{"id": "Wreck"})
			wtest.Must(t, err)
			assert.True(t, found)
			assert.EqualValues(t, "punk band reference", w.Comment)
		}

		langs := []*Language{
			{ID: fr.ID},
			{ID: en.ID},
		}
		err := c.Preload(conn, langs, hades.Assoc("Words"))
		// many_to_many preload is not implemented
		assert.Error(t, err)
	})
}

type Profile struct {
	ID           int64
	ProfileGames []*ProfileGame
}

type Game struct {
	ID    int64
	Title string
}

type ProfileGame struct {
	ProfileID int64 `hades:"primary_key"`
	Profile   *Profile

	GameID int64 `hades:"primary_key"`
	Game   *Game

	Order int64
}

func Test_ManyToManyRevenge(t *testing.T) {
	models := []interface{}{&Profile{}, &ProfileGame{}, &Game{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		makeProfile := func() *Profile {
			return &Profile{
				ID: 389,
				ProfileGames: []*ProfileGame{
					{
						Order: 1,
						Game: &Game{
							ID:    58372,
							Title: "First offensive",
						},
					},
					{
						Order: 5,
						Game: &Game{
							ID:    235971,
							Title: "Seconds until midnight",
						},
					},
					{
						Order: 7,
						Game: &Game{
							ID:    10598,
							Title: "Three was company",
						},
					},
				},
			}
		}
		p := makeProfile()
		wtest.Must(t, c.Save(conn, p,
			hades.Assoc("ProfileGames",
				hades.Assoc("Game"),
			),
		))

		var names []struct {
			Name string
		}
		wtest.Must(t, c.ExecWithSearch(conn,
			builder.Select("games.title").
				From("games").
				LeftJoin("profile_games", builder.Expr("profile_games.game_id = games.id")),
			hades.Search().OrderBy("profile_games.\"order\" ASC"),
			c.IntoRowsScanner(&names),
		))
		assert.EqualValues(t, []struct {
			Name string
		}{
			{"First offensive"},
			{"Seconds until midnight"},
			{"Three was company"},
		}, names)
	})
}

type Piece struct {
	ID      int64
	Authors []*Author `hades:"many_to_many:piece_authors"`
}

type Author struct {
	ID     int64
	Name   string
	Pieces []*Piece `hades:"many_to_many:piece_authors"`
}

type PieceAuthor struct {
	AuthorID int64 `hades:"primary_key"`
	PieceID  int64 `hades:"primary_key"`
}

func Test_ManyToManyThorough(t *testing.T) {
	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	models := []interface{}{&Piece{}, &Author{}, &PieceAuthor{}}

	c, err := hades.NewContext(makeConsumer(t), models...)
	ordie(err)
	c.Log = true

	ordie(c.AutoMigrate(conn))

	assertCount := func(model interface{}, expected int) {
		t.Helper()
		actual, err := c.Count(conn, model, builder.NewCond())
		ordie(err)
		assert.EqualValues(t, expected, actual)
	}

	t.Logf("Creating 1 piece with 10 authors")

	p := &Piece{ID: 321}

	for i := 0; i < 10; i++ {
		p.Authors = append(p.Authors, &Author{
			ID: int64(i + 1000),
		})
	}
	originalAuthors := p.Authors

	{
		beforeSaveQueryCount := c.QueryCount
		ordie(c.Save(conn, p, hades.Assoc("Authors")))

		pieceSelect := 1
		pieceInsert := 1

		authorSelect := 1
		authorInsert := len(p.Authors)

		pieceAuthorSelect := 1
		pieceAuthorInsert := len(p.Authors)

		total := pieceSelect + pieceInsert +
			authorSelect + authorInsert +
			pieceAuthorSelect + pieceAuthorInsert

		assert.EqualValues(t, total, c.QueryCount-beforeSaveQueryCount)
	}

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(p.Authors))
	assertCount(&PieceAuthor{}, len(p.Authors))

	t.Logf("Disassociating 5 authors from piece")

	var fewerAuthors []*Author
	for i, author := range p.Authors {
		if i%2 == 0 {
			fewerAuthors = append(fewerAuthors, author)
		}
	}
	p.Authors = fewerAuthors

	{
		beforeSaveQueryCount := c.QueryCount
		ordie(c.Save(conn, p, hades.AssocReplace("Authors")))

		pieceSelect := 1

		authorSelect := 1

		pieceAuthorSelect := 1
		pieceAuthorDelete := 1

		total := pieceSelect +
			authorSelect +
			pieceAuthorSelect + pieceAuthorDelete

		assert.EqualValues(t, total, c.QueryCount-beforeSaveQueryCount)
	}

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors))
	assertCount(&PieceAuthor{}, len(p.Authors))

	t.Logf("Updating 1 author")

	p.Authors[2].Name = "Hansel"

	{
		beforeSaveQueryCount := c.QueryCount
		ordie(c.Save(conn, p, hades.AssocReplace("Authors")))

		pieceSelect := 1

		authorSelect := 1
		authorUpdate := 1

		pieceAuthorSelect := 1

		total := pieceSelect +
			authorSelect + authorUpdate +
			pieceAuthorSelect

		assert.EqualValues(t, total, c.QueryCount-beforeSaveQueryCount)
	}

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors))
	assertCount(&PieceAuthor{}, len(p.Authors))

	t.Logf("Updating 2 authors, adding 1, deleting 1")

	p.Authors[0].Name = "Grieschka"
	p.Authors[1].Name = "Peggy"
	p.Authors = append(p.Authors[0:4], &Author{
		ID:   2001,
		Name: "Joseph",
	})

	{
		beforeSaveQueryCount := c.QueryCount
		ordie(c.Save(conn, p, hades.AssocReplace("Authors")))

		pieceSelect := 1

		authorSelect := 1
		authorInsert := 1
		authorUpdate := 2

		pieceAuthorSelect := 1
		pieceAuthorInsert := 1
		pieceAuthorDelete := 1

		total := pieceSelect +
			authorSelect + authorInsert + authorUpdate +
			pieceAuthorSelect + pieceAuthorInsert + pieceAuthorDelete

		assert.EqualValues(t, total, c.QueryCount-beforeSaveQueryCount)
	}

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors)+1)
	assertCount(&PieceAuthor{}, len(p.Authors))

	// now let's try to break SQLite's max variables limit
	for i := 0; i < 1200; i++ {
		p.Authors = append(p.Authors, &Author{
			ID: int64(i + 4000),
		})
	}

	ordie(c.Save(conn, p, hades.AssocReplace("Authors")))

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors)+1+1200)
	assertCount(&PieceAuthor{}, len(p.Authors))

	p.Authors = nil
	ordie(c.Save(conn, p, hades.AssocReplace("Authors")))

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors)+1+1200)
	assertCount(&PieceAuthor{}, 0)
}
