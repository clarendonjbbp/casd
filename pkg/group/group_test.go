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

func TestReadGroupsNormalizesVerboseWorkshopReferences(t *testing.T) {
	groupsCSV := `Teacher Name,Room Number,Grade,Group number,Names of students in this group (first and last),Art Workshops 1,Art Workshops 2,Art Workshops 3,Art Workshops 4,Science Workshop 1,Science Workshop 2,Science Workshop 3,Science Workshop 4,Presenter Kids
Korey Snyder,103,1,Group 1,"Elon Slepnyov, Shiv Devgan",A22 - Percussion with Teacher Janie Hsiao (K-2),A3 - SFPL: Mini Bookmaking (K-5),A6 - Physical Education by Push Academy (K-5),A16 - The Playspace: Theater Games (K-2),S1 - Soda Geysers (K-5),S2 - Dolphin Project: All About Dolphins (K-5),S8 - Marble Run (K-2),S25 - Potion Making (K-5),S20
`
	path := filepath.Join(t.TempDir(), "groups.csv")
	require.NoError(t, os.WriteFile(path, []byte(groupsCSV), 0o644))

	groups, err := ReadGroups(path)
	require.NoError(t, err)
	require.Len(t, groups, 1)

	assert.Equal(t, []string{"A22", "A3", "A6", "A16"}, groups[0].ArtIDs)
	assert.Equal(t, []string{"S1", "S2", "S8", "S25"}, groups[0].SciIDs)
	assert.Contains(t, groups[0].ParentIDs, "S20")
}

func TestReadGroupsAddsSuffixForDuplicateGeneratedIDs(t *testing.T) {
	groupsCSV := `Teacher Name,Room Number,Grade,Group number,Names of students in this group (first and last),Art Workshops 1,Art Workshops 2,Art Workshops 3,Art Workshops 4,Science Workshop 1,Science Workshop 2,Science Workshop 3,Science Workshop 4,Presenter Kids
Colleen Tate,204,3,Group 1,"A, B",A1,A2,A3,A4,S1,S2,S3,S4,
Colleen Tate,204,3,Group 1,"C, D",A1,A2,A3,A4,S1,S2,S3,S4,
Colleen Tate,204,3,Group 1,"E, F",A1,A2,A3,A4,S1,S2,S3,S4,
`
	path := filepath.Join(t.TempDir(), "groups.csv")
	require.NoError(t, os.WriteFile(path, []byte(groupsCSV), 0o644))

	groups, err := ReadGroups(path)
	require.NoError(t, err)
	require.Len(t, groups, 3)

	assert.Equal(t, "Colleen_Tate-3-Group_1", groups[0].ID)
	assert.Equal(t, "Colleen_Tate-3-Group_1-2", groups[1].ID)
	assert.Equal(t, "Colleen_Tate-3-Group_1-3", groups[2].ID)
}
