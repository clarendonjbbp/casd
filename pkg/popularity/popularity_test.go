package popularity

import (
	"testing"

	"github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	"github.com/clarendonjbbp/casd/pkg/workshop"
	"github.com/stretchr/testify/assert"
)

func TestCalculateScoresAllPreferenceRanks(t *testing.T) {
	groups := []*group.Group{
		{
			Grade:  1,
			ArtIDs: []string{"A1", "A2", "A3", "A4"},
			SciIDs: []string{"S1", "S2", "S3", "S4"},
		},
		{
			Grade:  1,
			ArtIDs: []string{"A2", "A1", "", ""},
			SciIDs: []string{"S2", "S1", "", ""},
		},
	}
	artWorkshops := map[string]*workshop.Workshop{
		"A1": testPopularityWorkshop("A1", model.ArtWorkshop),
		"A2": testPopularityWorkshop("A2", model.ArtWorkshop),
		"A3": testPopularityWorkshop("A3", model.ArtWorkshop),
		"A4": testPopularityWorkshop("A4", model.ArtWorkshop),
	}
	sciWorkshops := map[string]*workshop.Workshop{
		"S1": testPopularityWorkshop("S1", model.SciWorkshop),
		"S2": testPopularityWorkshop("S2", model.SciWorkshop),
		"S3": testPopularityWorkshop("S3", model.SciWorkshop),
		"S4": testPopularityWorkshop("S4", model.SciWorkshop),
	}

	popularityByID := make(map[string]*Entry)
	for _, popularity := range Calculate(groups, artWorkshops, sciWorkshops) {
		popularityByID[popularity.ID] = popularity
	}

	assert.Equal(t, 7, popularityByID["A1"].Score)
	assert.Equal(t, 7, popularityByID["A2"].Score)
	assert.Equal(t, 2, popularityByID["A3"].Score)
	assert.Equal(t, 1, popularityByID["A4"].Score)
	assert.Equal(t, 7, popularityByID["S1"].Score)
	assert.Equal(t, 7, popularityByID["S2"].Score)
	assert.Equal(t, 2, popularityByID["S3"].Score)
	assert.Equal(t, 1, popularityByID["S4"].Score)
	assert.Equal(t, 8, popularityByID["A1"].MaxScore)
	assert.Equal(t, 8750, popularityByID["A1"].NormalizedScoreBasisPoints())
	assert.Equal(t, 87, popularityByID["A1"].NormalizedScore())
	assert.Equal(t, [PreferenceRanks]int{1, 1, 0, 0}, popularityByID["A1"].RankCounts)
	assert.Equal(t, [PreferenceRanks]int{1, 1, 0, 0}, popularityByID["S1"].RankCounts)
}

func TestSortUsesNormalizedScore(t *testing.T) {
	popularity := []*Entry{
		{ID: "A1", Kind: model.ArtWorkshop, Score: 20, MaxScore: 100},
		{ID: "A2", Kind: model.ArtWorkshop, Score: 10, MaxScore: 20},
	}

	sorted := Sort(popularity, true)

	assert.Equal(t, "A2", sorted[0].ID)
}

func TestAggregateCombinesWorkshopGradeVariants(t *testing.T) {
	popularity := []*Entry{
		{
			ID:         "S20",
			Kind:       model.SciWorkshop,
			Workshop:   testPopularityWorkshopWithName("S20", model.SciWorkshop, "Goatlandia! (TK-2)"),
			RankCounts: [PreferenceRanks]int{1, 2, 3, 4},
			Score:      20,
			MaxScore:   100,
		},
		{
			ID:         "S21",
			Kind:       model.SciWorkshop,
			Workshop:   testPopularityWorkshopWithName("S21", model.SciWorkshop, "Goatlandia! (3-5)"),
			RankCounts: [PreferenceRanks]int{5, 6, 7, 8},
			Score:      40,
			MaxScore:   100,
		},
	}

	aggregated := Aggregate(popularity)

	assert.Len(t, aggregated, 1)
	assert.Equal(t, "S20, S21", aggregated[0].ID)
	assert.Equal(t, "Goatlandia!", WorkshopName(aggregated[0]))
	assert.Equal(t, "Goatlandia! (TK-5)", ReportWorkshopName(aggregated[0]))
	assert.Equal(t, [PreferenceRanks]int{6, 8, 10, 12}, aggregated[0].RankCounts)
	assert.Equal(t, 60, aggregated[0].Score)
	assert.Equal(t, 200, aggregated[0].MaxScore)
}

func TestAggregateWorkshopNameOnlyRemovesGradeRangeSuffix(t *testing.T) {
	assert.Equal(t, "Goatlandia!", AggregateWorkshopName("Goatlandia! (TK-2)"))
	assert.Equal(t, "The Playspace", AggregateWorkshopName("The Playspace (3-5)"))
	assert.Equal(t, "Ooblek! (messy fun)", AggregateWorkshopName("Ooblek! (messy fun)"))
}

func testPopularityWorkshop(id string, kind int) *workshop.Workshop {
	return testPopularityWorkshopWithName(id, kind, id+" Workshop")
}

func testPopularityWorkshopWithName(id string, kind int, name string) *workshop.Workshop {
	return &workshop.Workshop{
		ID:       id,
		Kind:     kind,
		Name:     name,
		MinGrade: minGradeForTestWorkshop(id),
		MaxGrade: maxGradeForTestWorkshop(id),
	}
}

func minGradeForTestWorkshop(id string) int {
	if id == "S21" {
		return 3
	}
	return model.TKGrade
}

func maxGradeForTestWorkshop(id string) int {
	if id == "S20" {
		return 2
	}
	return 5
}
