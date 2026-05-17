package popularity

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

const PreferenceRanks = 4

type Entry struct {
	Workshop   *workshopPkg.Workshop
	ID         string
	Kind       int
	RankCounts [PreferenceRanks]int
	Score      int
	MaxScore   int
}

var preferencePoints = [PreferenceRanks]int{4, 3, 2, 1}

var gradeRangeSuffixPattern = regexp.MustCompile(`\s+\((?:TK|K|\d+)\s*-\s*(?:TK|K|\d+)\)$`)

func Calculate(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop) []*Entry {
	byID := make(map[string]*Entry, len(artWorkshops)+len(sciWorkshops))
	for _, workshop := range artWorkshops {
		byID[workshop.ID] = &Entry{
			Workshop: workshop,
			ID:       workshop.ID,
			Kind:     model.ArtWorkshop,
		}
	}
	for _, workshop := range sciWorkshops {
		byID[workshop.ID] = &Entry{
			Workshop: workshop,
			ID:       workshop.ID,
			Kind:     model.SciWorkshop,
		}
	}

	for _, group := range groups {
		scorePreferences(byID, group.ArtIDs, model.ArtWorkshop)
		scorePreferences(byID, group.SciIDs, model.SciWorkshop)
		addEligibleMaxScores(byID, group)
	}

	popularity := make([]*Entry, 0, len(byID))
	for _, workshopPopularity := range byID {
		popularity = append(popularity, workshopPopularity)
	}

	return popularity
}

func scorePreferences(byID map[string]*Entry, preferences []string, kind int) {
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
			popularity = &Entry{
				ID:   workshopID,
				Kind: kind,
			}
			byID[workshopID] = popularity
		}

		popularity.RankCounts[rank]++
		popularity.Score += points
	}
}

func addEligibleMaxScores(byID map[string]*Entry, group *groupPkg.Group) {
	for _, popularity := range byID {
		if popularity.Workshop == nil || !popularity.Workshop.WithinGradeRange(group.Grade) {
			continue
		}
		popularity.MaxScore += 4
	}
}

func Aggregate(popularity []*Entry) []*Entry {
	byName := make(map[string]*Entry, len(popularity))
	for _, item := range popularity {
		name := WorkshopName(item)
		aggregatedName := AggregateWorkshopName(name)
		key := fmt.Sprintf("%d:%s", item.Kind, strings.ToLower(aggregatedName))

		aggregated, ok := byName[key]
		if !ok {
			aggregated = &Entry{
				ID:   item.ID,
				Kind: item.Kind,
			}
			if item.Workshop != nil {
				workshopCopy := *item.Workshop
				workshopCopy.Name = aggregatedName
				aggregated.Workshop = &workshopCopy
			}
			byName[key] = aggregated
		} else {
			aggregated.ID = appendWorkshopID(aggregated.ID, item.ID)
			mergeWorkshopGradeRange(aggregated.Workshop, item.Workshop)
		}

		for rank, count := range item.RankCounts {
			aggregated.RankCounts[rank] += count
		}
		aggregated.Score += item.Score
		aggregated.MaxScore += item.MaxScore
	}

	aggregated := make([]*Entry, 0, len(byName))
	for _, item := range byName {
		aggregated = append(aggregated, item)
	}
	return aggregated
}

func AggregateWorkshopName(name string) string {
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

func FilterByKind(popularity []*Entry, kind int) []*Entry {
	filtered := make([]*Entry, 0, len(popularity))
	for _, workshopPopularity := range popularity {
		if workshopPopularity.Kind == kind {
			filtered = append(filtered, workshopPopularity)
		}
	}
	return filtered
}

func Sort(popularity []*Entry, descending bool) []*Entry {
	sorted := slices.Clone(popularity)
	slices.SortFunc(sorted, func(a, b *Entry) int {
		if descending {
			if n := cmp.Compare(b.NormalizedScoreBasisPoints(), a.NormalizedScoreBasisPoints()); n != 0 {
				return n
			}
			if n := cmp.Compare(b.Score, a.Score); n != 0 {
				return n
			}
			return cmp.Compare(a.ID, b.ID)
		}
		if n := cmp.Compare(a.NormalizedScoreBasisPoints(), b.NormalizedScoreBasisPoints()); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Score, b.Score); n != 0 {
			return n
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return sorted
}

func (p *Entry) NormalizedScoreBasisPoints() int {
	if p.MaxScore == 0 {
		return 0
	}
	return p.Score * 10000 / p.MaxScore
}

func (p *Entry) NormalizedScore() int {
	if p.MaxScore == 0 {
		return 0
	}
	return p.Score * 100 / p.MaxScore
}

func WorkshopName(popularity *Entry) string {
	if popularity.Workshop == nil {
		return "(missing from workshop CSV)"
	}
	return popularity.Workshop.Name
}

func ReportWorkshopName(popularity *Entry) string {
	if popularity.Workshop == nil {
		return WorkshopName(popularity)
	}
	return fmt.Sprintf(
		"%s (%s-%s)",
		WorkshopName(popularity),
		model.GradeLabel(popularity.Workshop.MinGrade),
		model.GradeLabel(popularity.Workshop.MaxGrade),
	)
}
