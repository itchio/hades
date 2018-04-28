package hades_test

import (
	"testing"
	"time"

	"github.com/itchio/hades"
	"github.com/stretchr/testify/assert"
)

func Test_DBValue(t *testing.T) {
	var s *string = nil
	assert.Nil(t, hades.DBValue(s))

	tim := time.Now()
	assert.EqualValues(t, tim.Format(time.RFC3339Nano), hades.DBValue(tim))

	assert.EqualValues(t, 42, hades.DBValue(42))
	assert.EqualValues(t, 3.14, hades.DBValue(3.14))
}
