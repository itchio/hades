package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_HasOne(t *testing.T) {
	type Drawback struct {
		ID          int64
		Comment     string
		SpecialtyID string
	}

	type Specialty struct {
		ID        string
		CountryID int64
		Drawback  *Drawback
	}

	type Country struct {
		ID        int64
		Desc      string
		Specialty *Specialty
	}

	models := []interface{}{&Country{}, &Specialty{}, &Drawback{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		country := &Country{
			ID:   324,
			Desc: "Shmance",
			Specialty: &Specialty{
				ID: "complain",
				Drawback: &Drawback{
					ID:      1249,
					Comment: "bitterness",
				},
			},
		}
		assertCount := func(model interface{}, expectedCount int64) {
			t.Helper()
			var count int64
			count, err := c.Count(conn, model, builder.NewCond())
			wtest.Must(t, err)
			assert.EqualValues(t, expectedCount, count)
		}

		wtest.Must(t, c.Save(conn, &hades.SaveParams{Record: country, Assocs: []string{"Specialty"}}))
		assertCount(&Country{}, 0)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		wtest.Must(t, c.Save(conn, &hades.SaveParams{Record: country}))
		assertCount(&Country{}, 1)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		var countries []*Country
		for i := 0; i < 4; i++ {
			country := &Country{}
			wtest.Must(t, c.SelectOne(conn, country, builder.Eq{"id": 324}))
			countries = append(countries, country)
		}

		wtest.Must(t, c.Preload(conn, &hades.PreloadParams{
			Record: countries,
			Fields: []hades.PreloadField{
				{Name: "Specialty"},
				{Name: "Specialty.Drawback"},
			},
		}))
	})
}
