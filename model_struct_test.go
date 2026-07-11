package hades_test

import (
	"sync"
	"testing"

	"github.com/itchio/hades"
	"github.com/itchio/hades/mtest"
	"github.com/stretchr/testify/assert"
)

// relationship-free types (like ScanIntoRows row structs) can be
// first-built from concurrent goroutines, so the cached ModelStruct must
// be complete when published. run with -race
func Test_ModelStructConcurrentFirstUse(t *testing.T) {
	type Inner struct {
		ID    int64
		Title string
	}
	// metadata is cached globally by type: only the first use in the
	// process exercises construction, so Row must be unique to this test
	type Row struct {
		Inner `hades:"squash"`
	}

	c, err := hades.NewContext(&Inner{})
	mtest.Must(t, err)

	var start, done sync.WaitGroup
	start.Add(1)
	for i := 0; i < 16; i++ {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			ms := c.NewScope(Row{}).GetModelStruct()
			assert.NotNil(t, ms.StructFieldsByName["Inner"])
			assert.Len(t, ms.StructFields, 1)
		}()
	}
	start.Done()
	done.Wait()
}
