package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	"github.com/clarendonjbbp/casd/pkg/workshop"
	"github.com/stretchr/testify/assert"
)

func TestCalculatePopularityScoresAllPreferenceRanks(t *testing.T) {
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

	popularityByID := make(map[string]*workshopPopularity)
	for _, popularity := range calculatePopularity(groups, artWorkshops, sciWorkshops) {
		popularityByID[popularity.id] = popularity
	}

	assert.Equal(t, 7, popularityByID["A1"].score)
	assert.Equal(t, 7, popularityByID["A2"].score)
	assert.Equal(t, 2, popularityByID["A3"].score)
	assert.Equal(t, 1, popularityByID["A4"].score)
	assert.Equal(t, 7, popularityByID["S1"].score)
	assert.Equal(t, 7, popularityByID["S2"].score)
	assert.Equal(t, 2, popularityByID["S3"].score)
	assert.Equal(t, 1, popularityByID["S4"].score)
	assert.Equal(t, 8, popularityByID["A1"].maxScore)
	assert.Equal(t, 8750, popularityByID["A1"].normalizedScoreBasisPoints())
	assert.Equal(t, 87, popularityByID["A1"].normalizedScore())
	assert.Equal(t, [preferenceRanks]int{1, 1, 0, 0}, popularityByID["A1"].rankCounts)
	assert.Equal(t, [preferenceRanks]int{1, 1, 0, 0}, popularityByID["S1"].rankCounts)
}

func TestPrintPopularityReportSeparatesArtAndScience(t *testing.T) {
	popularity := []*workshopPopularity{
		{id: "A1", kind: model.ArtWorkshop, score: 10, maxScore: 20},
		{id: "A2", kind: model.ArtWorkshop, score: 1, maxScore: 20},
		{id: "S1", kind: model.SciWorkshop, score: 12, maxScore: 20},
		{id: "S2", kind: model.SciWorkshop, score: 2, maxScore: 20},
	}

	var output bytes.Buffer
	printPopularityReport(&output, popularity, 1, textFormat)

	report := output.String()
	assert.Contains(t, report, "Most Popular Art Workshops")
	assert.Contains(t, report, "Least Popular Art Workshops")
	assert.Contains(t, report, "Most Popular Science Workshops")
	assert.Contains(t, report, "Least Popular Science Workshops")
	assert.Less(t, strings.Index(report, "A1"), strings.Index(report, "S1"))
}

func TestPrintPopularityReportCanPrintMarkdown(t *testing.T) {
	popularity := []*workshopPopularity{
		{id: "A1", kind: model.ArtWorkshop, workshop: testPopularityWorkshopWithName("A1", model.ArtWorkshop, "Pipe | Workshop"), score: 10, maxScore: 20},
	}

	var output bytes.Buffer
	printPopularityReport(&output, popularity, 1, markdownFormat)

	report := output.String()
	assert.Contains(t, report, "### Most Popular Art Workshops")
	assert.Contains(t, report, "| Rank | ID | Workshop | Score | 1st | 2nd | 3rd | 4th |")
	assert.Contains(t, report, "Pipe \\| Workshop")
}

func TestPrintPopularityReportCanPrintHTML(t *testing.T) {
	popularity := []*workshopPopularity{
		{id: "A1", kind: model.ArtWorkshop, workshop: testPopularityWorkshopWithName("A1", model.ArtWorkshop, "Workshop <One>"), score: 10, maxScore: 20},
	}

	var output bytes.Buffer
	printPopularityReport(&output, popularity, 1, htmlFormat)

	report := output.String()
	assert.Contains(t, report, "<table>")
	assert.Contains(t, report, "Workshop &lt;One&gt;")
	assert.Contains(t, report, "Score is between 0 and 100")
}

func TestPrintPopularityReportSortsByNormalizedScore(t *testing.T) {
	popularity := []*workshopPopularity{
		{id: "A1", kind: model.ArtWorkshop, score: 20, maxScore: 100},
		{id: "A2", kind: model.ArtWorkshop, score: 10, maxScore: 20},
	}

	sorted := sortPopularity(popularity, true)

	assert.Equal(t, "A2", sorted[0].id)
}

func TestAggregatePopularityCombinesWorkshopGradeVariants(t *testing.T) {
	popularity := []*workshopPopularity{
		{
			id:         "S20",
			kind:       model.SciWorkshop,
			workshop:   testPopularityWorkshopWithName("S20", model.SciWorkshop, "Goatlandia! (TK-2)"),
			rankCounts: [preferenceRanks]int{1, 2, 3, 4},
			score:      20,
			maxScore:   100,
		},
		{
			id:         "S21",
			kind:       model.SciWorkshop,
			workshop:   testPopularityWorkshopWithName("S21", model.SciWorkshop, "Goatlandia! (3-5)"),
			rankCounts: [preferenceRanks]int{5, 6, 7, 8},
			score:      40,
			maxScore:   100,
		},
	}

	aggregated := aggregatePopularity(popularity)

	assert.Len(t, aggregated, 1)
	assert.Equal(t, "S20, S21", aggregated[0].id)
	assert.Equal(t, "Goatlandia!", workshopName(aggregated[0]))
	assert.Equal(t, "Goatlandia! (TK-5)", reportWorkshopName(aggregated[0]))
	assert.Equal(t, [preferenceRanks]int{6, 8, 10, 12}, aggregated[0].rankCounts)
	assert.Equal(t, 60, aggregated[0].score)
	assert.Equal(t, 200, aggregated[0].maxScore)
}

func TestAggregateWorkshopNameOnlyRemovesGradeRangeSuffix(t *testing.T) {
	assert.Equal(t, "Goatlandia!", aggregateWorkshopName("Goatlandia! (TK-2)"))
	assert.Equal(t, "The Playspace", aggregateWorkshopName("The Playspace (3-5)"))
	assert.Equal(t, "Ooblek! (messy fun)", aggregateWorkshopName("Ooblek! (messy fun)"))
}

func TestWrapTextBreaksWorkshopNameIntoReadableLines(t *testing.T) {
	assert.Equal(
		t,
		[]string{"What Food Does for Your Body with Push", "Academy"},
		wrapText("What Food Does for Your Body with Push Academy", workshopNameWidth),
	)
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
