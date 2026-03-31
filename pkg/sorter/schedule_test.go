package sorter

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookWorkshopIfAvailableBooksSessionWithMostRemainingCapacity(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{5, 10, -1, -1})

	booked := BookWorkshopIfAvailable(workshop, group)

	assert.True(t, booked)
	assert.Same(t, workshop, group.GetWorkshop(1))
	assert.Equal(t, 6, workshop.SpotsAvailable[1])
	require.Len(t, workshop.GetGroupsForSession(1), 1)
	assert.Same(t, group, workshop.GetGroupsForSession(1)[0])
}

func TestBookWorkshopIfAvailableRejectsOutOfGradeRange(t *testing.T) {
	group := testGroup("group-1", 1, 4, nil, nil)
	workshop := testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1})

	booked := BookWorkshopIfAvailable(workshop, group)

	assert.False(t, booked)
}

func TestBookWorkshopIfAvailableRejectsDuplicateWorkshop(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{10, 10, -1, -1})
	group.BookWorkshop(0, workshop)

	booked := BookWorkshopIfAvailable(workshop, group)

	assert.False(t, booked)
}

func TestBookWorkshopIfAvailableRejectsWhenCapacityIsTooSmall(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{3, -1, -1, -1})

	booked := BookWorkshopIfAvailable(workshop, group)

	assert.False(t, booked)
}

func TestRebalanceWorkshopsMovesGroupIntoUnderutilizedSession(t *testing.T) {
	group := testGroup("group-1", 3, 2, []string{"A9"}, nil)
	oldWorkshop := testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{0, -1, -1, -1})
	targetWorkshop := testWorkshop("A2", ArtWorkshop, 3, 5, 10, []int{8, -1, -1, -1})

	oldWorkshop.sessionGroups[0] = []*Group{group}
	group.BookWorkshop(0, oldWorkshop)

	workshops := map[string]*Workshop{
		oldWorkshop.id:    oldWorkshop,
		targetWorkshop.id: targetWorkshop,
	}

	err := RebalanceWorkshops(30, workshops, []*Group{group})
	require.NoError(t, err)
	assert.Same(t, targetWorkshop, group.GetWorkshop(0))
	assert.Equal(t, 2, oldWorkshop.SpotsAvailable[0])
	assert.Equal(t, 6, targetWorkshop.SpotsAvailable[0])
	assert.Empty(t, oldWorkshop.GetGroupsForSession(0))
	require.Len(t, targetWorkshop.GetGroupsForSession(0), 1)
	assert.Same(t, group, targetWorkshop.GetGroupsForSession(0)[0])
}

func TestReadCSVFilesLoadsRepositoryFixtures(t *testing.T) {
	root := filepath.Join("..", "..")

	groups, artWorkshops, sciWorkshops, err := ReadCSVFiles(
		filepath.Join(root, "groups.csv"),
		filepath.Join(root, "artworkshops.csv"),
		filepath.Join(root, "scienceworkshops.csv"),
	)
	require.NoError(t, err)
	require.NotEmpty(t, groups)
	require.NotEmpty(t, artWorkshops)
	require.NotEmpty(t, sciWorkshops)
	assert.Equal(t, "Michael", groups[0].Teacher)
	assert.Equal(t, 4, groups[0].Grade)
	assert.Equal(t, "Group 1", groups[0].Name)
	assert.True(t, slices.Equal(groups[0].ArtIDs, []string{"A5", "A7", "A12", "A18"}))
	_, ok := artWorkshops["A5"]
	assert.True(t, ok)
	_, ok = sciWorkshops["S23"]
	assert.True(t, ok)
}

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

func TestGroupSatisfactionCalculatesPointsAndPercent(t *testing.T) {
	group := testGroup("group-1", 3, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	group.ParentIDs["S99"] = struct{}{}

	group.BookWorkshop(0, testWorkshop("A2", ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}))
	group.BookWorkshop(1, testWorkshop("A9", ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}))
	group.BookWorkshop(2, testWorkshop("S1", SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}))
	group.BookWorkshop(3, testWorkshop("S99", SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}))

	assert.Equal(t, 12, group.GetSatisfaction())
	assert.Equal(t, 16, group.MaxSatisfaction())
	assert.Equal(t, 75, group.SatisfactionPercent())
}

func TestGroupMaxSatisfactionIgnoresBlankPreferences(t *testing.T) {
	group := testGroup("group-1", 3, 4, []string{"A1", "", "", ""}, []string{"", "", "", ""})
	group.BookWorkshop(0, testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}))
	group.BookWorkshop(1, testWorkshop("A9", ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}))
	group.BookWorkshop(2, testWorkshop("S9", SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}))
	group.BookWorkshop(3, testWorkshop("S8", SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}))

	assert.Equal(t, 4, group.MaxSatisfaction())
	assert.Equal(t, 100, group.SatisfactionPercent())
}

func TestGroupSatisfactionDoesNotDoubleCountDuplicatePreferences(t *testing.T) {
	group := testGroup("group-1", 0, 4, []string{"A20", "A19", "A15", "A14"}, []string{"S14", "S21", "S25", ""})
	group.ParentIDs["S13"] = struct{}{}

	group.BookWorkshop(0, testWorkshop("A20", ArtWorkshop, 0, 0, 10, []int{10, -1, -1, -1}))
	group.BookWorkshop(1, testWorkshop("A19", ArtWorkshop, 0, 0, 10, []int{-1, 10, -1, -1}))
	group.BookWorkshop(2, testWorkshop("S14", SciWorkshop, 0, 0, 10, []int{-1, -1, 10, -1}))
	group.BookWorkshop(3, testWorkshop("S13", SciWorkshop, 0, 0, 10, []int{-1, -1, -1, 10}))

	assert.Equal(t, 16, group.GetSatisfaction())
	assert.Equal(t, 16, group.MaxSatisfaction())
	assert.Equal(t, 100, group.SatisfactionPercent())
}

func TestCalculateScheduleSummary(t *testing.T) {
	first := testGroup("group-1", 3, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	first.BookWorkshop(0, testWorkshop("A1", ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}))
	first.BookWorkshop(1, testWorkshop("A9", ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}))
	first.BookWorkshop(2, testWorkshop("S2", SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}))
	first.BookWorkshop(3, testWorkshop("S9", SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}))

	second := testGroup("group-2", 3, 4, []string{"A5", "A6", "A7", "A8"}, []string{"S5", "S6", "S7", "S8"})
	second.BookWorkshop(0, testWorkshop("A9", ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}))
	second.BookWorkshop(1, testWorkshop("A6", ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}))
	second.BookWorkshop(2, testWorkshop("S5", SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}))
	second.BookWorkshop(3, testWorkshop("S8", SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}))

	summary := CalculateScheduleSummary([]*Group{first, second})

	assert.Equal(t, 15, summary.OverallSatisfactionPoints)
	assert.Equal(t, 53, summary.AverageSatisfactionPercent)
	assert.Equal(t, 2, summary.GroupsWithPreferredArt)
	assert.Equal(t, 2, summary.GroupsWithPreferredScience)
	assert.Equal(t, 2, summary.TotalGroups)
}

func TestBookParentClassesRecordsGradeMismatchIssue(t *testing.T) {
	group := testGroup("group-1", 2, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	group.ParentIDs["S24"] = struct{}{}

	artWorkshops := map[string]*Workshop{}
	sciWorkshops := map[string]*Workshop{
		"S24": testWorkshop("S24", SciWorkshop, 3, 5, 10, []int{10, 10, 10, 10}),
	}

	err := bookParentClasses([]*Group{group}, artWorkshops, sciWorkshops, true)
	require.NoError(t, err)
	assert.Equal(t, "grade mismatch", group.ParentBookingIssues["S24"])
	assert.Zero(t, group.SessionsBooked(SciWorkshop))
}

func testGroup(id string, grade, students int, artIDs, sciIDs []string) *Group {
	studentNames := make([]string, students)
	for i := range studentNames {
		studentNames[i] = "student"
	}

	return &Group{
		Teacher:             "Teacher",
		Name:                "Group",
		Grade:               grade,
		students:            studentNames,
		ArtIDs:              artIDs,
		SciIDs:              sciIDs,
		workshops:           make([]*Workshop, 4),
		ParentIDs:           map[string]struct{}{},
		ParentBookingIssues: map[string]string{},
		id:                  id,
	}
}

func testWorkshop(id string, kind, minGrade, maxGrade, capacity int, spotsAvailable []int) *Workshop {
	spots := make([]int, len(spotsAvailable))
	copy(spots, spotsAvailable)

	return &Workshop{
		Kind:           kind,
		id:             id,
		Name:           id,
		MinGrade:       minGrade,
		MaxGrade:       maxGrade,
		Capacity:       capacity,
		SpotsAvailable: spots,
		sessionGroups:  make(map[int][]*Group),
	}
}
