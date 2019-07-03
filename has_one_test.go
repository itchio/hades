package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
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
			mtest.Must(t, err)
			assert.EqualValues(t, expectedCount, count)
		}

		mtest.Must(t, c.Save(conn, country, hades.OmitRoot(), hades.Assoc("Specialty", hades.Assoc("Drawback"))))
		assertCount(&Country{}, 0)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		mtest.Must(t, c.Save(conn, country, hades.Assoc("Specialty", hades.Assoc("Drawback"))))
		assertCount(&Country{}, 1)
		assertCount(&Specialty{}, 1)
		assertCount(&Drawback{}, 1)

		var countries []*Country
		for i := 0; i < 4; i++ {
			country := &Country{}
			found, err := c.SelectOne(conn, country, builder.Eq{"id": 324})
			mtest.Must(t, err)
			assert.True(t, found)
			countries = append(countries, country)
		}

		mtest.Must(t, c.Preload(conn, countries,
			hades.Assoc("Specialty",
				hades.Assoc("Drawback"))))
	})
}
