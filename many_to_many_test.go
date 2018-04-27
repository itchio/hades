package hades_test

import (
	"context"
	"testing"

	"github.com/go-xorm/builder"
	"github.com/stretchr/testify/assert"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
)

type Piece struct {
	ID      int64
	Authors []*Author `hades:"many2many:piece_authors"`
}

type Author struct {
	ID     int64
	Name   string
	Pieces []*Piece `hades:"many2many:piece_authors"`
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
		ordie(c.SaveOne(conn, p))

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
		ordie(c.SaveOne(conn, p))

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
		ordie(c.SaveOne(conn, p))

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
		ordie(c.SaveOne(conn, p))

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

	ordie(c.SaveOne(conn, p))

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors)+1+1200)
	assertCount(&PieceAuthor{}, len(p.Authors))

	p.Authors = nil
	ordie(c.SaveOne(conn, p))

	assertCount(&Piece{}, 1)
	assertCount(&Author{}, len(originalAuthors)+1+1200)
	assertCount(&PieceAuthor{}, 0)
}
