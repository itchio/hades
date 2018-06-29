package hades_test

import (
	"testing"

	"crawshaw.io/sqlite"
	"github.com/itchio/hades"
	"github.com/itchio/wharf/wtest"
)

func Test_PreloadEdgeCases(t *testing.T) {
	type Bar struct {
		ID int64
	}

	type Foo struct {
		ID    int64
		BarID int64
		Bar   *Bar
	}

	models := []interface{}{&Foo{}, &Bar{}}

	withContext(t, models, func(q Querier, c *hades.Context) {
		// non-existent Bar
		f := &Foo{ID: 1, BarID: 999}
		wtest.Must(t, c.Preload(q, f, hades.Assoc("Bar")))

		// empty slice
		var foos []*Foo
		wtest.Must(t, c.Preload(q, foos, hades.Assoc("Bar")))
	})
}
