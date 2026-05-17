package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/clarendonjbbp/casd/pkg/model"
	popularityPkg "github.com/clarendonjbbp/casd/pkg/popularity"
	"github.com/clarendonjbbp/casd/pkg/workshop"
	"github.com/stretchr/testify/assert"
)

func TestPrintPopularityReportSeparatesArtAndScience(t *testing.T) {
	popularity := []*popularityPkg.Entry{
		{ID: "A1", Kind: model.ArtWorkshop, Score: 10, MaxScore: 20},
		{ID: "A2", Kind: model.ArtWorkshop, Score: 1, MaxScore: 20},
		{ID: "S1", Kind: model.SciWorkshop, Score: 12, MaxScore: 20},
		{ID: "S2", Kind: model.SciWorkshop, Score: 2, MaxScore: 20},
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
	popularity := []*popularityPkg.Entry{
		{ID: "A1", Kind: model.ArtWorkshop, Workshop: testRenderWorkshop("A1", model.ArtWorkshop, "Pipe | Workshop"), Score: 10, MaxScore: 20},
	}

	var output bytes.Buffer
	printPopularityReport(&output, popularity, 1, markdownFormat)

	report := output.String()
	assert.Contains(t, report, "### Most Popular Art Workshops")
	assert.Contains(t, report, "| Rank | ID | Workshop | Score | 1st | 2nd | 3rd | 4th |")
	assert.Contains(t, report, "Pipe \\| Workshop")
}

func TestPrintPopularityReportCanPrintHTML(t *testing.T) {
	popularity := []*popularityPkg.Entry{
		{ID: "A1", Kind: model.ArtWorkshop, Workshop: testRenderWorkshop("A1", model.ArtWorkshop, "Workshop <One>"), Score: 10, MaxScore: 20},
	}

	var output bytes.Buffer
	printPopularityReport(&output, popularity, 1, htmlFormat)

	report := output.String()
	assert.Contains(t, report, "<table>")
	assert.Contains(t, report, "Workshop &lt;One&gt;")
	assert.Contains(t, report, "Score is between 0 and 100")
}

func TestWrapTextBreaksWorkshopNameIntoReadableLines(t *testing.T) {
	assert.Equal(
		t,
		[]string{"What Food Does for Your Body with Push", "Academy"},
		wrapText("What Food Does for Your Body with Push Academy", workshopNameWidth),
	)
}

func testRenderWorkshop(id string, kind int, name string) *workshop.Workshop {
	return &workshop.Workshop{
		ID:       id,
		Kind:     kind,
		Name:     name,
		MinGrade: model.TKGrade,
		MaxGrade: 5,
	}
}
