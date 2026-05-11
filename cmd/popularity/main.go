package main

import (
	"cmp"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"strings"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	"github.com/clarendonjbbp/casd/pkg/scheduler"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

type workshopPopularity struct {
	workshop   *workshopPkg.Workshop
	id         string
	kind       int
	rankCounts [preferenceRanks]int
	score      int
	maxScore   int
}

const (
	preferenceRanks = 4
)

var preferencePoints = [preferenceRanks]int{4, 3, 2, 1}

var gradeRangeSuffixPattern = regexp.MustCompile(`\s+\((?:TK|K|\d+)\s*-\s*(?:TK|K|\d+)\)$`)

func main() {
	groupsFile := flag.String("groups", "groups.csv", "CSV file with groups")
	artWorkshopsFile := flag.String("art-workshops", "artworkshops.csv", "CSV file with Art workshops")
	sciWorkshopsFile := flag.String("science-workshops", "scienceworkshops.csv", "CSV file with Science workshops")
	limit := flag.Int("limit", 5, "Number of most and least popular workshops to print")
	format := flag.String("format", string(textFormat), "Output format: text, markdown, or html")
	flag.Parse()

	groups, artWorkshops, sciWorkshops, err := scheduler.ReadCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	popularity := calculatePopularity(groups, artWorkshops, sciWorkshops)
	reportFormat, err := parseReportFormat(*format)
	if err != nil {
		log.Fatal(err)
	}
	printPopularityReport(os.Stdout, popularity, *limit, reportFormat)
}

func calculatePopularity(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop) []*workshopPopularity {
	byID := make(map[string]*workshopPopularity, len(artWorkshops)+len(sciWorkshops))
	for _, workshop := range artWorkshops {
		byID[workshop.ID] = &workshopPopularity{
			workshop: workshop,
			id:       workshop.ID,
			kind:     model.ArtWorkshop,
		}
	}
	for _, workshop := range sciWorkshops {
		byID[workshop.ID] = &workshopPopularity{
			workshop: workshop,
			id:       workshop.ID,
			kind:     model.SciWorkshop,
		}
	}

	for _, group := range groups {
		scorePreferences(byID, group.ArtIDs, model.ArtWorkshop)
		scorePreferences(byID, group.SciIDs, model.SciWorkshop)
		addEligibleMaxScores(byID, group)
	}

	popularity := make([]*workshopPopularity, 0, len(byID))
	for _, workshopPopularity := range byID {
		popularity = append(popularity, workshopPopularity)
	}

	return popularity
}

func scorePreferences(byID map[string]*workshopPopularity, preferences []string, kind int) {
	for rank, points := range preferencePoints {
		if rank >= len(preferences) {
			return
		}

		workshopID := strings.TrimSpace(preferences[rank])
		if workshopID == "" {
			continue
		}

		popularity, ok := byID[workshopID]
		if !ok {
			popularity = &workshopPopularity{
				id:   workshopID,
				kind: kind,
			}
			byID[workshopID] = popularity
		}

		popularity.rankCounts[rank]++
		popularity.score += points
	}
}

func addEligibleMaxScores(byID map[string]*workshopPopularity, group *groupPkg.Group) {
	for _, popularity := range byID {
		if popularity.workshop == nil || !popularity.workshop.WithinGradeRange(group.Grade) {
			continue
		}
		popularity.maxScore += 4
	}
}

func aggregatePopularity(popularity []*workshopPopularity) []*workshopPopularity {
	byName := make(map[string]*workshopPopularity, len(popularity))
	for _, item := range popularity {
		name := workshopName(item)
		aggregatedName := aggregateWorkshopName(name)
		key := fmt.Sprintf("%d:%s", item.kind, strings.ToLower(aggregatedName))

		aggregated, ok := byName[key]
		if !ok {
			aggregated = &workshopPopularity{
				id:   item.id,
				kind: item.kind,
			}
			if item.workshop != nil {
				workshopCopy := *item.workshop
				workshopCopy.Name = aggregatedName
				aggregated.workshop = &workshopCopy
			}
			byName[key] = aggregated
		} else {
			aggregated.id = appendWorkshopID(aggregated.id, item.id)
			mergeWorkshopGradeRange(aggregated.workshop, item.workshop)
		}

		for rank, count := range item.rankCounts {
			aggregated.rankCounts[rank] += count
		}
		aggregated.score += item.score
		aggregated.maxScore += item.maxScore
	}

	aggregated := make([]*workshopPopularity, 0, len(byName))
	for _, item := range byName {
		aggregated = append(aggregated, item)
	}
	return aggregated
}

func aggregateWorkshopName(name string) string {
	return gradeRangeSuffixPattern.ReplaceAllString(name, "")
}

func appendWorkshopID(existing string, id string) string {
	if existing == "" {
		return id
	}
	ids := strings.Split(existing, ", ")
	if slices.Contains(ids, id) {
		return existing
	}
	ids = append(ids, id)
	slices.Sort(ids)
	return strings.Join(ids, ", ")
}

func mergeWorkshopGradeRange(target *workshopPkg.Workshop, source *workshopPkg.Workshop) {
	if target == nil || source == nil {
		return
	}
	target.MinGrade = min(target.MinGrade, source.MinGrade)
	target.MaxGrade = max(target.MaxGrade, source.MaxGrade)
}

func filterPopularityByKind(popularity []*workshopPopularity, kind int) []*workshopPopularity {
	filtered := make([]*workshopPopularity, 0, len(popularity))
	for _, workshopPopularity := range popularity {
		if workshopPopularity.kind == kind {
			filtered = append(filtered, workshopPopularity)
		}
	}
	return filtered
}

func sortPopularity(popularity []*workshopPopularity, descending bool) []*workshopPopularity {
	sorted := slices.Clone(popularity)
	slices.SortFunc(sorted, func(a, b *workshopPopularity) int {
		if descending {
			if n := cmp.Compare(b.normalizedScoreBasisPoints(), a.normalizedScoreBasisPoints()); n != 0 {
				return n
			}
			if n := cmp.Compare(b.score, a.score); n != 0 {
				return n
			}
			return cmp.Compare(a.id, b.id)
		}
		if n := cmp.Compare(a.normalizedScoreBasisPoints(), b.normalizedScoreBasisPoints()); n != 0 {
			return n
		}
		if n := cmp.Compare(a.score, b.score); n != 0 {
			return n
		}
		return cmp.Compare(a.id, b.id)
	})
	return sorted
}

func (p *workshopPopularity) normalizedScoreBasisPoints() int {
	if p.maxScore == 0 {
		return 0
	}
	return p.score * 10000 / p.maxScore
}

func (p *workshopPopularity) normalizedScore() int {
	if p.maxScore == 0 {
		return 0
	}
	return p.score * 100 / p.maxScore
}

func workshopName(popularity *workshopPopularity) string {
	if popularity.workshop == nil {
		return "(missing from workshop CSV)"
	}
	return popularity.workshop.Name
}
