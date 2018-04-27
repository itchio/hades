package hades_test

import (
	"context"
	"testing"

	"crawshaw.io/sqlite"
	"github.com/go-xorm/builder"
	"github.com/itchio/hades"
	"github.com/stretchr/testify/assert"
)

func Test_AutoMigrate(t *testing.T) {
	dbpool, err := sqlite.Open("file:memory:?mode=memory", 0, 10)
	ordie(err)
	defer dbpool.Close()

	conn := dbpool.Get(context.Background().Done())
	defer dbpool.Put(conn)

	{
		type User struct {
			ID        int64
			FirstName string
		}

		models := []interface{}{&User{}}

		c, err := hades.NewContext(makeConsumer(t), models...)
		ordie(err)
		c.Log = true

		t.Logf("first migration")
		ordie(c.AutoMigrate(conn))

		pti, err := c.PragmaTableInfo(conn, "users")
		ordie(err)
		assert.EqualValues(t, "id", pti[0].Name)
		assert.EqualValues(t, "INTEGER", pti[0].Type)
		assert.True(t, pti[0].PrimaryKey)
		assert.True(t, pti[0].NotNull)

		assert.EqualValues(t, "first_name", pti[1].Name)
		assert.EqualValues(t, "TEXT", pti[1].Type)
		assert.False(t, pti[1].PrimaryKey)
		assert.False(t, pti[1].NotNull)

		ordie(c.SaveOne(conn, &User{ID: 123, FirstName: "Joanna"}))
		u := &User{}
		ordie(c.SelectOne(conn, u, builder.Eq{"id": 123}))
		assert.EqualValues(t, &User{ID: 123, FirstName: "Joanna"}, u)

		t.Logf("first migration (bis)")
		ordie(c.AutoMigrate(conn))
	}

	{
		type User struct {
			ID        int64
			FirstName string
			LastName  string
		}

		models := []interface{}{&User{}}

		c, err := hades.NewContext(makeConsumer(t), models...)
		ordie(err)
		c.Log = true

		t.Logf("second migration")
		ordie(c.AutoMigrate(conn))

		pti, err := c.PragmaTableInfo(conn, "users")
		ordie(err)
		assert.EqualValues(t, "id", pti[0].Name)
		assert.EqualValues(t, "INTEGER", pti[0].Type)
		assert.True(t, pti[0].PrimaryKey)
		assert.True(t, pti[0].NotNull)

		assert.EqualValues(t, "first_name", pti[1].Name)
		assert.EqualValues(t, "TEXT", pti[1].Type)
		assert.False(t, pti[1].PrimaryKey)
		assert.False(t, pti[1].NotNull)

		assert.EqualValues(t, "last_name", pti[2].Name)
		assert.EqualValues(t, "TEXT", pti[2].Type)
		assert.False(t, pti[2].PrimaryKey)
		assert.False(t, pti[2].NotNull)

		u := &User{}
		ordie(c.SelectOne(conn, u, builder.Eq{"id": 123}))
		assert.EqualValues(t, &User{ID: 123, FirstName: "Joanna", LastName: ""}, u)

		t.Logf("second migration (bis)")
		ordie(c.AutoMigrate(conn))
	}
}

func ordie(err error) {
	if err != nil {
		panic(err)
	}
}
