package hades_test

import (
	"testing"

	"github.com/itchio/hades"

	"github.com/stretchr/testify/assert"
)

func Test_FromDBName(t *testing.T) {
	assert.EqualValues(t, "OwnerID", hades.FromDBName("owner_id"))
	assert.EqualValues(t, "ID", hades.FromDBName("id"))
	assert.EqualValues(t, "ProfileGames", hades.FromDBName("profile_games"))
}

func Test_ToDBName(t *testing.T) {
	assert.EqualValues(t, "owner_id", hades.ToDBName("OwnerId"))
	assert.EqualValues(t, "id", hades.ToDBName("ID"))
	assert.EqualValues(t, "profile_games", hades.ToDBName("ProfileGames"))
}
