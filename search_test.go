package hades

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Search(t *testing.T) {
	assert.EqualValues(t, "x", Search().Apply("x"))
	assert.EqualValues(t, "x LIMIT 1", Search().Limit(1).Apply("x"))
}
