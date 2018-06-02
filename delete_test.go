package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
	"github.com/stretchr/testify/assert"
)

func Test_Delete(t *testing.T) {
	type Story struct {
		ID   int64
		Body string
	}

	models := []interface{}{&Story{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		stories := []*Story{
			&Story{
				ID:   1,
				Body: "jesus wept",
			},
			&Story{
				ID:   2,
				Body: "exit status 1",
			},
			&Story{
				ID:   8,
				Body: "ice cold",
			},
		}

		var count int64
		var err error

		wtest.Must(t, c.Save(conn, stories))

		count, err = c.Count(conn, &Story{}, builder.NewCond())
		wtest.Must(t, err)
		assert.EqualValues(t, 3, count)

		err = c.Delete(conn, &Story{}, builder.NewCond())
		assert.Error(t, err, "must refuse to delete with empty cond")
		assert.Contains(t, err.Error(), "refusing to blindly")

		err = c.Delete(conn, &Story{}, builder.Eq{"id": 1})
		wtest.Must(t, err)
		count, err = c.Count(conn, &Story{}, builder.NewCond())
		wtest.Must(t, err)
		assert.EqualValues(t, 2, count)

		err = c.Delete(conn, &Story{}, builder.Expr("1"))
		assert.NoError(t, err, "must delete all with expr")
		count, err = c.Count(conn, &Story{}, builder.NewCond())
		wtest.Must(t, err)
		assert.EqualValues(t, 0, count)
	})
}
