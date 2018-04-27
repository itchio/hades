package hades

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Search(t *testing.T) {
	assert.EqualValues(t, "x", Search().Apply("x"))
	assert.EqualValues(t, "x LIMIT 1", Search().Limit(1).Apply("x"))
	assert.EqualValues(t, "x ORDER BY id desc", Search().OrderBy("id desc").Apply("x"))
	assert.EqualValues(t, "x ORDER BY id asc", Search().OrderBy("id asc").Apply("x"))
	assert.EqualValues(t, "x ORDER BY id asc, created_at desc", Search().OrderBy("id asc").OrderBy("created_at desc").Apply("x"))
}
