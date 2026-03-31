package group

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadGroupsDeduplicatesPreferencesInOrder(t *testing.T) {
	groupsCSV := `Teacher Name,Room Number,Grade,Group number,Names of students in this group (first and last),Art Workshops 1,Art Workshops 2,Art Workshops 3,Art Workshops 4,Science Workshop 1,Science Workshop 2,Science Workshop 3,Science Workshop 4,Presenter Kids
Elizabeth,209,K,Group 5,"Carter Schwartz, Evelyn Anderson",A20,A20,A19,A15,S14,S14,S21,S25,S13
`
	path := filepath.Join(t.TempDir(), "groups.csv")
	require.NoError(t, os.WriteFile(path, []byte(groupsCSV), 0o644))

	groups, err := ReadGroups(path)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	assert.Equal(t, []string{"A20", "A19", "A15", ""}, groups[0].ArtIDs)
	assert.Equal(t, []string{"S14", "S21", "S25", ""}, groups[0].SciIDs)
}
