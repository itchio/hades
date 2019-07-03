package hades_test

import (
	"reflect"
	"testing"

	"crawshaw.io/sqlite"
	"xorm.io/builder"
	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
)

func Test_SquashedToEq(t *testing.T) {
	type BoneName string

	type BoneTraits struct {
		Name     BoneName
		Goodness int64
	}

	type Bone struct {
		ID     int64
		Traits BoneTraits `hades:"squash"`
	}

	b := &Bone{
		ID: 510,
		Traits: BoneTraits{
			Name:     "cranium",
			Goodness: 3,
		},
	}

	c, err := hades.NewContext(nil, &Bone{})
	mtest.Must(t, err)

	boneScope := c.NewScope(b)
	eq := boneScope.ToEq(reflect.ValueOf(b))

	assert.EqualValues(t, 3, len(eq))
	assert.EqualValues(t, 510, eq["id"])
	assert.EqualValues(t, "cranium", eq["name"])
	assert.EqualValues(t, 3, eq["goodness"])
}

func Test_SquashedInsert(t *testing.T) {
	type BoneTraits struct {
		Name     string
		Goodness int64
	}

	type Bone struct {
		ID     int64
		Traits BoneTraits `hades:"squash"`
	}

	models := []interface{}{&Bone{}}
	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		b := &Bone{
			ID: 128,
			Traits: BoneTraits{
				Name:     "humerus",
				Goodness: 98,
			},
		}
		val := reflect.ValueOf(b)
		scope := c.ScopeMap.ByType(val.Type())
		mtest.Must(t, c.Insert(conn, scope, val))

		bb := &Bone{}
		mtest.Must(t, c.ExecRaw(conn, "SELECT * FROM bones", func(stmt *sqlite.Stmt) error {
			bb.ID = stmt.ColumnInt64(0)
			bb.Traits.Name = stmt.ColumnText(1)
			bb.Traits.Goodness = stmt.ColumnInt64(2)
			return nil
		}))

		assert.EqualValues(t, b, bb)
	})
}

func Test_SquashedFull(t *testing.T) {
	type FakeGameTraits struct {
		Storied    bool
		Ubiquitous bool
	}

	type FakeGame struct {
		ID         int64
		FakeUserID int64
		Traits     FakeGameTraits `hades:"squash"`
	}

	type FakeUser struct {
		ID    int64
		Games []*FakeGame
	}

	models := []interface{}{&FakeGame{}, &FakeUser{}}

	withContext(t, models, func(conn *sqlite.Conn, c *hades.Context) {
		fu := &FakeUser{
			ID: 15,
			Games: []*FakeGame{
				&FakeGame{
					ID: 2,
					Traits: FakeGameTraits{
						Storied: true,
					},
				},
				&FakeGame{
					ID: 4,
					Traits: FakeGameTraits{
						Ubiquitous: true,
					},
				},
				&FakeGame{
					ID: 6,
					Traits: FakeGameTraits{
						Storied:    true,
						Ubiquitous: true,
					},
				},
			},
		}
		mtest.Must(t, c.Save(conn, fu, hades.Assoc("Games")))

		u := &FakeUser{}

		found, err := c.SelectOne(conn, u, builder.NewCond())
		mtest.Must(t, err)
		assert.True(t, found)

		assert.EqualValues(t, 15, u.ID)

		mtest.Must(t, c.Preload(conn, u,
			hades.AssocWithSearch("Games", hades.Search{}.OrderBy("id ASC").Offset(1)),
		))
		assert.EqualValues(t, 2, len(u.Games))
		assert.EqualValues(t, FakeGameTraits{Ubiquitous: true}, u.Games[0].Traits)
		assert.EqualValues(t, FakeGameTraits{Storied: true, Ubiquitous: true}, u.Games[1].Traits)

		fu.Games[2].Traits.Storied = false
		fu.Games[2].Traits.Ubiquitous = false
		mtest.Must(t, c.Save(conn, fu, hades.Assoc("Games")))

		mtest.Must(t, c.Preload(conn, u,
			hades.AssocWithSearch("Games", hades.Search{}.OrderBy("id ASC").Offset(2).Limit(1)),
		))
		assert.EqualValues(t, 1, len(u.Games))
		assert.EqualValues(t, 6, u.Games[0].ID)
		assert.EqualValues(t, FakeGameTraits{Ubiquitous: false, Storied: false}, u.Games[0].Traits)
	})
}
