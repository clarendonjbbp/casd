package scheduler

import (
	"path/filepath"
	"testing"

	"github.com/clarendonjbbp/casd/pkg/booking"
	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCSVFilesLoadsRepositoryFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "2024")

	groups, artWorkshops, sciWorkshops, err := ReadCSVFiles(
		filepath.Join(root, "groups_randomized.csv"),
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
	assert.Equal(t, []string{"A5", "A7", "A12", "A18"}, groups[0].ArtIDs)
	assert.Contains(t, artWorkshops, "A5")
	assert.Contains(t, sciWorkshops, "S23")
}

func TestRebalanceWorkshopsMovesGroupIntoUnderutilizedSession(t *testing.T) {
	group := testGroup("group-1", 3, 2, []string{"A9"}, nil)
	filler := testGroup("filler", 3, 5, nil, nil)
	oldWorkshop := testWorkshop("A1", model.ArtWorkshop, []int{0, -1, -1, -1})
	targetWorkshop := testWorkshop("A2", model.ArtWorkshop, []int{8, -1, -1, -1})
	state := booking.NewScheduleState([]*groupPkg.Group{group, filler}, map[string]*workshopPkg.Workshop{
		oldWorkshop.ID:    oldWorkshop,
		targetWorkshop.ID: targetWorkshop,
	})
	state.Book(filler, oldWorkshop, 0)
	state.Book(group, oldWorkshop, 0)

	workshops := map[string]*workshopPkg.Workshop{
		oldWorkshop.ID:    oldWorkshop,
		targetWorkshop.ID: targetWorkshop,
	}

	err := RebalanceWorkshops(state, 30, workshops, []*groupPkg.Group{group})
	require.Error(t, err)
	assert.Same(t, targetWorkshop, state.WorkshopForGroupSession(group, 0))
	assert.Equal(t, 5, state.SpotsAvailable(oldWorkshop, 0))
	assert.Equal(t, 8, state.SpotsAvailable(targetWorkshop, 0))
	require.Len(t, state.GroupsForWorkshopSession(oldWorkshop, 0), 1)
	assert.Same(t, filler, state.GroupsForWorkshopSession(oldWorkshop, 0)[0])
	require.Len(t, state.GroupsForWorkshopSession(targetWorkshop, 0), 1)
	assert.Same(t, group, state.GroupsForWorkshopSession(targetWorkshop, 0)[0])
}

func TestRebalanceWorkshopsDoesNotBreakOnlyPreferredWorkshopGuarantee(t *testing.T) {
	group := testGroup("group-1", 3, 2, []string{"A1", "", "", ""}, nil)
	group2 := testGroup("group-2", 3, 2, nil, nil)
	group3 := testGroup("group-3", 3, 2, nil, nil)
	oldWorkshop := testWorkshop("A1", model.ArtWorkshop, []int{4, -1, -1, -1})
	targetWorkshop := testWorkshop("A2", model.ArtWorkshop, []int{8, -1, -1, -1})
	state := booking.NewScheduleState([]*groupPkg.Group{group, group2, group3}, map[string]*workshopPkg.Workshop{
		oldWorkshop.ID:    oldWorkshop,
		targetWorkshop.ID: targetWorkshop,
	})
	state.Book(group, oldWorkshop, 0)
	state.Book(group2, oldWorkshop, 0)
	state.Book(group3, oldWorkshop, 0)

	workshops := map[string]*workshopPkg.Workshop{
		oldWorkshop.ID:    oldWorkshop,
		targetWorkshop.ID: targetWorkshop,
	}

	err := RebalanceWorkshops(state, 30, workshops, []*groupPkg.Group{group})
	require.Error(t, err)
	assert.Same(t, oldWorkshop, state.WorkshopForGroupSession(group, 0))
	assert.Equal(t, 4, state.SpotsAvailable(oldWorkshop, 0))
	assert.Equal(t, 10, state.SpotsAvailable(targetWorkshop, 0))
	assert.Empty(t, state.GroupsForWorkshopSession(targetWorkshop, 0))
}

func TestBookParentClassesRecordsGradeMismatchIssue(t *testing.T) {
	group := testGroup("group-1", 2, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	group.ParentIDs["S24"] = struct{}{}
	state := booking.NewScheduleState([]*groupPkg.Group{group})

	artWorkshops := map[string]*workshopPkg.Workshop{}
	sciWorkshops := map[string]*workshopPkg.Workshop{
		"S24": testWorkshop("S24", model.SciWorkshop, []int{10, 10, 10, 10}),
	}

	err := bookParentClasses(state, []*groupPkg.Group{group}, artWorkshops, sciWorkshops, true)
	require.NoError(t, err)
	assert.Equal(t, "grade mismatch", group.ParentBookingIssues["S24"])
	assert.Zero(t, state.SessionsBooked(group, model.SciWorkshop))
}

func TestBookGuaranteedPreferredWorkshopBooksOnePreferredWorkshopPerKind(t *testing.T) {
	group := testGroup("group-1", 3, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	state := booking.NewScheduleState([]*groupPkg.Group{group})

	artWorkshops := map[string]*workshopPkg.Workshop{
		"A1": testWorkshop("A1", model.ArtWorkshop, []int{10, 10, -1, -1}),
		"A2": testWorkshop("A2", model.ArtWorkshop, []int{10, 10, -1, -1}),
	}
	sciWorkshops := map[string]*workshopPkg.Workshop{
		"S1": testWorkshop("S1", model.SciWorkshop, []int{-1, -1, 10, 10}),
		"S2": testWorkshop("S2", model.SciWorkshop, []int{-1, -1, 10, 10}),
	}

	bookGuaranteedPreferredWorkshop(state, []*groupPkg.Group{group}, artWorkshops, model.ArtWorkshop)
	bookGuaranteedPreferredWorkshop(state, []*groupPkg.Group{group}, sciWorkshops, model.SciWorkshop)

	assert.Equal(t, 1, state.SessionsBooked(group, model.ArtWorkshop))
	assert.Equal(t, 1, state.SessionsBooked(group, model.SciWorkshop))
	assert.True(t, booking.HasPreferredWorkshopOfKind(group, state, model.ArtWorkshop))
	assert.True(t, booking.HasPreferredWorkshopOfKind(group, state, model.SciWorkshop))
}

func TestScheduleRandomRunsUsesReproducibleSeed(t *testing.T) {
	groups, artWorkshops, sciWorkshops := testOptimizationInputs()
	state, err := Schedule(groups, artWorkshops, sciWorkshops, ScheduleOptions{
		MinUtilization:            0,
		ContinueOnParentLookupErr: true,
		RandomRuns:                5,
		RandomSeed:                42,
		RandomSeedSet:             true,
	})
	require.NoError(t, err)

	groupsAgain, artWorkshopsAgain, sciWorkshopsAgain := testOptimizationInputs()
	stateAgain, err := Schedule(groupsAgain, artWorkshopsAgain, sciWorkshopsAgain, ScheduleOptions{
		MinUtilization:            0,
		ContinueOnParentLookupErr: true,
		RandomRuns:                5,
		RandomSeed:                42,
		RandomSeedSet:             true,
	})
	require.NoError(t, err)

	assert.Equal(t, scheduleSignature(groups, state), scheduleSignature(groupsAgain, stateAgain))
	assert.Equal(t, 5, state.RandomRuns)
	assert.GreaterOrEqual(t, state.SelectedRun, 1)
	assert.LessOrEqual(t, state.SelectedRun, 5)
	assert.Equal(t, int64(42+state.SelectedRun-1), state.SelectedSeed)
}

func testGroup(id string, grade, students int, artIDs, sciIDs []string) *groupPkg.Group {
	studentNames := make([]string, students)
	for i := range studentNames {
		studentNames[i] = "student"
	}

	return &groupPkg.Group{
		Teacher:             "Teacher",
		Name:                "Group",
		Grade:               grade,
		Students:            studentNames,
		ArtIDs:              artIDs,
		SciIDs:              sciIDs,
		ParentIDs:           map[string]struct{}{},
		ParentBookingIssues: map[string]string{},
		ID:                  id,
	}
}

func testOptimizationInputs() ([]*groupPkg.Group, map[string]*workshopPkg.Workshop, map[string]*workshopPkg.Workshop) {
	groups := []*groupPkg.Group{
		testGroup("group-1", 3, 4, []string{"A1", "A2", "", ""}, []string{"S1", "S2", "", ""}),
		testGroup("group-2", 3, 4, []string{"A1", "A2", "", ""}, []string{"S1", "S2", "", ""}),
	}
	artWorkshops := map[string]*workshopPkg.Workshop{
		"A1": testWorkshop("A1", model.ArtWorkshop, []int{10, 10, 10, 10}),
		"A2": testWorkshop("A2", model.ArtWorkshop, []int{10, 10, 10, 10}),
	}
	sciWorkshops := map[string]*workshopPkg.Workshop{
		"S1": testWorkshop("S1", model.SciWorkshop, []int{10, 10, 10, 10}),
		"S2": testWorkshop("S2", model.SciWorkshop, []int{10, 10, 10, 10}),
	}
	return groups, artWorkshops, sciWorkshops
}

func scheduleSignature(groups []*groupPkg.Group, state *booking.ScheduleState) []string {
	signature := make([]string, 0, len(groups)*model.NumSessions)
	for _, group := range groups {
		for session := range model.NumSessions {
			workshop := state.WorkshopForGroupSession(group, session)
			if workshop == nil {
				signature = append(signature, group.ID+":")
				continue
			}
			signature = append(signature, group.ID+":"+workshop.ID)
		}
	}
	return signature
}

func testWorkshop(id string, kind int, spotsAvailable []int) *workshopPkg.Workshop {
	offeredSessions := make([]bool, len(spotsAvailable))
	for i, spots := range spotsAvailable {
		offeredSessions[i] = spots != -1
	}

	return &workshopPkg.Workshop{
		Kind:            kind,
		ID:              id,
		Name:            id,
		MinGrade:        3,
		MaxGrade:        5,
		Capacity:        10,
		OfferedSessions: offeredSessions,
	}
}
