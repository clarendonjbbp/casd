package main

import (
	"fmt"
	"html"
	"io"
	"strings"

	"github.com/clarendonjbbp/casd/pkg/model"
)

const workshopNameWidth = 44

type reportWidths struct {
	id int
}

type reportFormat string

const (
	textFormat     reportFormat = "text"
	markdownFormat reportFormat = "markdown"
	htmlFormat     reportFormat = "html"
)

func parseReportFormat(format string) (reportFormat, error) {
	switch reportFormat(strings.ToLower(strings.TrimSpace(format))) {
	case textFormat:
		return textFormat, nil
	case markdownFormat:
		return markdownFormat, nil
	case htmlFormat:
		return htmlFormat, nil
	default:
		return "", fmt.Errorf("invalid format %q: expected text, markdown, or html", format)
	}
}

func printPopularityReport(output io.Writer, popularity []*workshopPopularity, limit int, format reportFormat) {
	if limit < 1 {
		limit = 1
	}

	aggregatedPopularity := aggregatePopularity(popularity)
	widths := calculateReportWidths(aggregatedPopularity)

	if format == htmlFormat {
		printHTMLPopularityReport(output, aggregatedPopularity, limit)
		return
	}

	printPopularityTable(output, "Most Popular Art Workshops", sortPopularity(filterPopularityByKind(aggregatedPopularity, model.ArtWorkshop), true), limit, widths, format)
	fmt.Fprintln(output)
	printPopularityTable(output, "Least Popular Art Workshops", sortPopularity(filterPopularityByKind(aggregatedPopularity, model.ArtWorkshop), false), limit, widths, format)
	fmt.Fprintln(output)
	printPopularityTable(output, "Most Popular Science Workshops", sortPopularity(filterPopularityByKind(aggregatedPopularity, model.SciWorkshop), true), limit, widths, format)
	fmt.Fprintln(output)
	printPopularityTable(output, "Least Popular Science Workshops", sortPopularity(filterPopularityByKind(aggregatedPopularity, model.SciWorkshop), false), limit, widths, format)
	fmt.Fprintln(output)
	fmt.Fprintln(output, "* Score is between 0 and 100, where 100 means every eligible group listed the workshop as its top choice.")
}

func calculateReportWidths(popularity []*workshopPopularity) reportWidths {
	widths := reportWidths{id: len("ID")}
	for _, item := range popularity {
		widths.id = max(widths.id, len(item.id))
	}
	return widths
}

func printPopularityTable(output io.Writer, title string, popularity []*workshopPopularity, limit int, widths reportWidths, format reportFormat) {
	if format == markdownFormat {
		printMarkdownPopularityTable(output, title, popularity, limit)
		return
	}
	printTextPopularityTable(output, title, popularity, limit, widths)
}

func printTextPopularityTable(output io.Writer, title string, popularity []*workshopPopularity, limit int, widths reportWidths) {
	fmt.Fprintln(output, title)
	fmt.Fprintln(output, strings.Repeat("-", len(title)))

	fmt.Fprintf(
		output,
		"%-4s  %-*s  %-*s  %5s  %3s  %3s  %3s  %3s\n",
		"Rank",
		widths.id,
		"ID",
		workshopNameWidth,
		"Workshop",
		"Score",
		"1st",
		"2nd",
		"3rd",
		"4th",
	)

	for i, workshopPopularity := range popularity[:min(limit, len(popularity))] {
		nameLines := wrapText(reportWorkshopName(workshopPopularity), workshopNameWidth)
		fmt.Fprintf(
			output,
			"%-4d  %-*s  %-*s  %5d  %3d  %3d  %3d  %3d\n",
			i+1,
			widths.id,
			workshopPopularity.id,
			workshopNameWidth,
			nameLines[0],
			workshopPopularity.normalizedScore(),
			workshopPopularity.rankCounts[0],
			workshopPopularity.rankCounts[1],
			workshopPopularity.rankCounts[2],
			workshopPopularity.rankCounts[3],
		)
		for _, line := range nameLines[1:] {
			fmt.Fprintf(output, "%-4s  %-*s  %-*s\n", "", widths.id, "", workshopNameWidth, line)
		}
	}
}

func printMarkdownPopularityTable(output io.Writer, title string, popularity []*workshopPopularity, limit int) {
	fmt.Fprintf(output, "### %s\n\n", title)
	fmt.Fprintln(output, "| Rank | ID | Workshop | Score | 1st | 2nd | 3rd | 4th |")
	fmt.Fprintln(output, "| ---: | --- | --- | ---: | ---: | ---: | ---: | ---: |")

	for i, workshopPopularity := range popularity[:min(limit, len(popularity))] {
		fmt.Fprintf(
			output,
			"| %d | %s | %s | %d | %d | %d | %d | %d |\n",
			i+1,
			escapeMarkdownTableCell(workshopPopularity.id),
			escapeMarkdownTableCell(reportWorkshopName(workshopPopularity)),
			workshopPopularity.normalizedScore(),
			workshopPopularity.rankCounts[0],
			workshopPopularity.rankCounts[1],
			workshopPopularity.rankCounts[2],
			workshopPopularity.rankCounts[3],
		)
	}
}

func printHTMLPopularityReport(output io.Writer, popularity []*workshopPopularity, limit int) {
	fmt.Fprintln(output, `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<title>Workshop Popularity Report</title>
<style>
body { font-family: Arial, sans-serif; color: #1f2933; line-height: 1.35; }
h2 { font-size: 18px; margin: 24px 0 8px; border-bottom: 1px solid #9aa5b1; padding-bottom: 4px; }
table { border-collapse: collapse; margin-bottom: 18px; width: 100%; max-width: 980px; }
th, td { border: 1px solid #d9e2ec; padding: 6px 8px; vertical-align: top; }
th { background: #f0f4f8; text-align: left; }
td.number, th.number { text-align: right; white-space: nowrap; }
td.id { white-space: nowrap; }
.note { margin-top: 20px; font-size: 13px; color: #52606d; }
</style>
</head>
<body>`)
	printHTMLPopularityTable(output, "Most Popular Art Workshops", sortPopularity(filterPopularityByKind(popularity, model.ArtWorkshop), true), limit)
	printHTMLPopularityTable(output, "Least Popular Art Workshops", sortPopularity(filterPopularityByKind(popularity, model.ArtWorkshop), false), limit)
	printHTMLPopularityTable(output, "Most Popular Science Workshops", sortPopularity(filterPopularityByKind(popularity, model.SciWorkshop), true), limit)
	printHTMLPopularityTable(output, "Least Popular Science Workshops", sortPopularity(filterPopularityByKind(popularity, model.SciWorkshop), false), limit)
	fmt.Fprintln(output, `<p class="note">* Score is between 0 and 100, where 100 means every eligible group listed the workshop as its top choice.</p>`)
	fmt.Fprintln(output, `</body>
</html>`)
}

func printHTMLPopularityTable(output io.Writer, title string, popularity []*workshopPopularity, limit int) {
	fmt.Fprintf(output, "<h2>%s</h2>\n", html.EscapeString(title))
	fmt.Fprintln(output, `<table>
<thead>
<tr><th>Rank</th><th>ID</th><th>Workshop</th><th class="number">Score</th><th class="number">1st</th><th class="number">2nd</th><th class="number">3rd</th><th class="number">4th</th></tr>
</thead>
<tbody>`)
	for i, workshopPopularity := range popularity[:min(limit, len(popularity))] {
		fmt.Fprint(output, "<tr>")
		fmt.Fprintf(output, `<td class="number">%d</td>`, i+1)
		fmt.Fprintf(output, `<td class="id">%s</td>`, html.EscapeString(workshopPopularity.id))
		fmt.Fprintf(output, "<td>%s</td>", html.EscapeString(reportWorkshopName(workshopPopularity)))
		printHTMLNumberCell(output, workshopPopularity.normalizedScore())
		for _, count := range workshopPopularity.rankCounts {
			printHTMLNumberCell(output, count)
		}
		fmt.Fprintln(output, "</tr>")
	}
	fmt.Fprintln(output, `</tbody>
</table>`)
}

func printHTMLNumberCell(output io.Writer, value int) {
	fmt.Fprintf(output, `<td class="number">%d</td>`, value)
}

func escapeMarkdownTableCell(value string) string {
	return strings.ReplaceAll(value, "|", `\|`)
}

func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		lastLine := lines[len(lines)-1]
		if len(lastLine)+1+len(word) <= width {
			lines[len(lines)-1] = lastLine + " " + word
			continue
		}
		lines = append(lines, word)
	}

	return lines
}

func reportWorkshopName(popularity *workshopPopularity) string {
	if popularity.workshop == nil {
		return workshopName(popularity)
	}
	return fmt.Sprintf(
		"%s (%s-%s)",
		workshopName(popularity),
		model.GradeLabel(popularity.workshop.MinGrade),
		model.GradeLabel(popularity.workshop.MaxGrade),
	)
}
