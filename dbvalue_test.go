package hades_test

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/itchio/hades"
	"github.com/stretchr/testify/assert"
)

func Test_DBValue(t *testing.T) {
	must := func(v any, err error) any {
		assert.NoError(t, err)
		return v
	}

	var s *string = nil
	assert.Nil(t, must(hades.DBValue(s)))

	tim := time.Now()
	assert.EqualValues(t, tim.Format(time.RFC3339Nano), must(hades.DBValue(tim)))

	assert.EqualValues(t, 42, must(hades.DBValue(42)))
	assert.EqualValues(t, 3.14, must(hades.DBValue(3.14)))

	var ns []string = nil
	assert.Nil(t, must(hades.DBValue(ns)))
	assert.EqualValues(t, `[]`, must(hades.DBValue([]string{})))
	assert.EqualValues(t, `["a","b"]`, must(hades.DBValue([]string{"a", "b"})))
	assert.EqualValues(t, `{"x":1}`, must(hades.DBValue(json.RawMessage(`{ "x": 1 }`))))
	assert.EqualValues(t, `[1,2]`, must(hades.DBValue(&[]int{1, 2})))

	_, err := hades.DBValue(json.RawMessage(`{not json`))
	assert.Error(t, err)
	_, err = hades.DBValue(json.RawMessage{})
	assert.Error(t, err)
	_, err = hades.DBValue([]float64{math.NaN()})
	assert.Error(t, err)
}
