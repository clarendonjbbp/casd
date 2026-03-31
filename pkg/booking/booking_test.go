package booking

import (
	"testing"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBookWorkshopIfAvailableBooksSessionWithMostRemainingCapacity(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{5, 10, -1, -1})
	filler := testGroup("filler", 3, 5, nil, nil)
	state := testScheduleState([]*groupPkg.Group{group, filler}, workshop)
	state.Book(filler, workshop, 0)

	booked := BookWorkshopIfAvailable(state, workshop, group)

	assert.True(t, booked)
	assert.Same(t, workshop, state.WorkshopForGroupSession(group, 1))
	assert.Equal(t, 6, state.SpotsAvailable(workshop, 1))
	require.Len(t, state.GroupsForWorkshopSession(workshop, 1), 1)
	assert.Same(t, group, state.GroupsForWorkshopSession(workshop, 1)[0])
}

func TestBookWorkshopIfAvailableRejectsOutOfGradeRange(t *testing.T) {
	group := testGroup("group-1", 1, 4, nil, nil)
	workshop := testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1})
	state := testScheduleState([]*groupPkg.Group{group}, workshop)

	booked := BookWorkshopIfAvailable(state, workshop, group)

	assert.False(t, booked)
}

func TestBookWorkshopIfAvailableRejectsDuplicateWorkshop(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{10, 10, -1, -1})
	state := testScheduleState([]*groupPkg.Group{group}, workshop)
	state.Book(group, workshop, 0)

	booked := BookWorkshopIfAvailable(state, workshop, group)

	assert.False(t, booked)
}

func TestBookWorkshopIfAvailableRejectsWhenCapacityIsTooSmall(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", model.ArtWorkshop, 3, 5, 3, []int{3, -1, -1, -1})
	state := testScheduleState([]*groupPkg.Group{group}, workshop)

	booked := BookWorkshopIfAvailable(state, workshop, group)

	assert.False(t, booked)
}

func TestBookWorkshopIfAvailableUsesDeterministicFirstSessionWhenRandomDisabled(t *testing.T) {
	group := testGroup("group-1", 3, 4, nil, nil)
	workshop := testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{10, 10, -1, -1})
	state := testScheduleState([]*groupPkg.Group{group}, workshop)
	state.SetRandomSelection(false)

	booked := BookWorkshopIfAvailable(state, workshop, group)

	assert.True(t, booked)
	assert.Same(t, workshop, state.WorkshopForGroupSession(group, 0))
	assert.Nil(t, state.WorkshopForGroupSession(group, 1))
}

func TestGroupSatisfactionCalculatesPointsAndPercent(t *testing.T) {
	group := testGroup("group-1", 3, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	group.ParentIDs["S99"] = struct{}{}
	state := NewScheduleState([]*groupPkg.Group{group})

	state.Book(group, testWorkshop("A2", model.ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}), 0)
	state.Book(group, testWorkshop("A9", model.ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}), 1)
	state.Book(group, testWorkshop("S1", model.SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}), 2)
	state.Book(group, testWorkshop("S99", model.SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}), 3)

	assert.Equal(t, 12, GetSatisfaction(group, state))
	assert.Equal(t, 16, MaxSatisfaction(group))
	assert.Equal(t, 75, SatisfactionPercent(group, state))
}

func TestGroupMaxSatisfactionIgnoresBlankPreferences(t *testing.T) {
	group := testGroup("group-1", 3, 4, []string{"A1", "", "", ""}, []string{"", "", "", ""})
	state := NewScheduleState([]*groupPkg.Group{group})
	state.Book(group, testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}), 0)
	state.Book(group, testWorkshop("A9", model.ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}), 1)
	state.Book(group, testWorkshop("S9", model.SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}), 2)
	state.Book(group, testWorkshop("S8", model.SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}), 3)

	assert.Equal(t, 4, MaxSatisfaction(group))
	assert.Equal(t, 100, SatisfactionPercent(group, state))
}

func TestGroupSatisfactionDoesNotDoubleCountDuplicatePreferences(t *testing.T) {
	group := testGroup("group-1", 0, 4, []string{"A20", "A19", "A15", "A14"}, []string{"S14", "S21", "S25", ""})
	group.ParentIDs["S13"] = struct{}{}
	state := NewScheduleState([]*groupPkg.Group{group})

	state.Book(group, testWorkshop("A20", model.ArtWorkshop, 0, 0, 10, []int{10, -1, -1, -1}), 0)
	state.Book(group, testWorkshop("A19", model.ArtWorkshop, 0, 0, 10, []int{-1, 10, -1, -1}), 1)
	state.Book(group, testWorkshop("S14", model.SciWorkshop, 0, 0, 10, []int{-1, -1, 10, -1}), 2)
	state.Book(group, testWorkshop("S13", model.SciWorkshop, 0, 0, 10, []int{-1, -1, -1, 10}), 3)

	assert.Equal(t, 16, GetSatisfaction(group, state))
	assert.Equal(t, 16, MaxSatisfaction(group))
	assert.Equal(t, 100, SatisfactionPercent(group, state))
}

func TestCalculateScheduleSummary(t *testing.T) {
	first := testGroup("group-1", 3, 4, []string{"A1", "A2", "A3", "A4"}, []string{"S1", "S2", "S3", "S4"})
	second := testGroup("group-2", 3, 4, []string{"A5", "A6", "A7", "A8"}, []string{"S5", "S6", "S7", "S8"})
	state := NewScheduleState([]*groupPkg.Group{first, second})

	state.Book(first, testWorkshop("A1", model.ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}), 0)
	state.Book(first, testWorkshop("A9", model.ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}), 1)
	state.Book(first, testWorkshop("S2", model.SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}), 2)
	state.Book(first, testWorkshop("S9", model.SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}), 3)

	state.Book(second, testWorkshop("A9", model.ArtWorkshop, 3, 5, 10, []int{10, -1, -1, -1}), 0)
	state.Book(second, testWorkshop("A6", model.ArtWorkshop, 3, 5, 10, []int{-1, 10, -1, -1}), 1)
	state.Book(second, testWorkshop("S5", model.SciWorkshop, 3, 5, 10, []int{-1, -1, 10, -1}), 2)
	state.Book(second, testWorkshop("S8", model.SciWorkshop, 3, 5, 10, []int{-1, -1, -1, 10}), 3)

	summary := CalculateScheduleSummary([]*groupPkg.Group{first, second}, state)

	assert.Equal(t, 15, summary.OverallSatisfactionPoints)
	assert.Equal(t, 53, summary.AverageSatisfactionPercent)
	assert.Equal(t, 2, summary.GroupsWithPreferredArt)
	assert.Equal(t, 2, summary.GroupsWithPreferredScience)
	assert.Equal(t, 2, summary.TotalGroups)
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

func testWorkshop(id string, kind, minGrade, maxGrade, capacity int, spotsAvailable []int) *workshopPkg.Workshop {
	offeredSessions := make([]bool, len(spotsAvailable))
	for i, spots := range spotsAvailable {
		offeredSessions[i] = spots != -1
	}

	return &workshopPkg.Workshop{
		Kind:            kind,
		ID:              id,
		Name:            id,
		MinGrade:        minGrade,
		MaxGrade:        maxGrade,
		Capacity:        capacity,
		OfferedSessions: offeredSessions,
	}
}

func testScheduleState(groups []*groupPkg.Group, workshops ...*workshopPkg.Workshop) *ScheduleState {
	workshopMap := make(map[string]*workshopPkg.Workshop, len(workshops))
	for _, workshop := range workshops {
		workshopMap[workshop.ID] = workshop
	}
	return NewScheduleState(groups, workshopMap)
}
