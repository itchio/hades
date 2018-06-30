package hades_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/state"
)

func Test_Null(t *testing.T) {
	consumer := &state.Consumer{
		OnMessage: func(lvl string, message string) {
			t.Logf("[%s] %s", lvl, message)
		},
	}

	type Download struct {
		ID           int64
		FinishedAt   *time.Time
		ErrorCode    *int64
		ErrorMessage *string
	}

	c, err := hades.NewContext(consumer, &Download{})
	if err != nil {
		panic(err)
	}
	c.Log = true

	sqlite.Logger = func(code sqlite.ErrorCode, msg []byte) {
		t.Logf("[SQLITE] %d %s", code, string(msg))
	}

	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	if err != nil {
		panic(err)
	}
	defer dbpool.Close()

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	ordie(c.AutoMigrate(conn))

	{
		d := &Download{
			ID: 123,
		}

		ordie(c.Save(conn, d))
		{
			dd := &Download{}
			found, err := c.SelectOne(conn, dd, builder.Eq{"id": 123})
			ordie(err)
			assert.True(t, found)

			assert.EqualValues(t, 123, dd.ID)
			assert.Nil(t, dd.FinishedAt)
			assert.Nil(t, dd.ErrorCode)
			assert.Nil(t, dd.ErrorMessage)
		}

		errMsg := "No rest for the wicked"
		d.ErrorMessage = &errMsg

		errCode := int64(9000)
		d.ErrorCode = &errCode

		finishedAt := time.Now()
		d.FinishedAt = &finishedAt

		ordie(c.Save(conn, d))

		{
			dd := &Download{}
			found, err := c.SelectOne(conn, dd, builder.Eq{"id": 123})
			ordie(err)
			assert.True(t, found)

			assert.EqualValues(t, 123, dd.ID)
			assert.EqualValues(t, *d.ErrorMessage, *dd.ErrorMessage)
			assert.EqualValues(t, *d.ErrorCode, *dd.ErrorCode)
			assert.EqualValues(t, (*d.FinishedAt).Format(time.RFC3339Nano), (*dd.FinishedAt).Format(time.RFC3339Nano))
		}

		d.ErrorMessage = nil
		ordie(c.Save(conn, d))

		{
			dd := &Download{}
			found, err := c.SelectOne(conn, dd, builder.Eq{"id": 123})
			ordie(err)
			assert.True(t, found)

			assert.EqualValues(t, 123, dd.ID)
			assert.Nil(t, dd.ErrorMessage)
			assert.EqualValues(t, *d.ErrorCode, *dd.ErrorCode)
			assert.EqualValues(t, (*d.FinishedAt).Format(time.RFC3339Nano), (*dd.FinishedAt).Format(time.RFC3339Nano))
		}
	}
}
