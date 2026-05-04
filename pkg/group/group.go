package group

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/clarendonjbbp/casd/pkg/model"
)

type Group struct {
	ParentIDs           map[string]struct{}
	ParentBookingIssues map[string]string
	Teacher             string
	Room                string
	Name                string
	Grade               int
	Students            []string
	ArtIDs              []string
	SciIDs              []string
	ID                  string
}

var workshopIDPattern = regexp.MustCompile(`(?i)^\s*([AS]\d+)\b`)

const (
	teacherColumn = iota
	roomColumn
	gradeColumn
	groupNameColumn
	studentsColumn
	artPreferencesStartColumn
	artPreferencesEndColumn       = artPreferencesStartColumn + 4
	sciencePreferencesStartColumn = artPreferencesEndColumn
	sciencePreferencesEndColumn   = sciencePreferencesStartColumn + 4
	parentWorkshopsColumn         = sciencePreferencesEndColumn
)

func ReadGroups(file string) ([]*Group, error) {
	var groups []*Group
	idCounts := make(map[string]int)

	reader, err := readAndParseCSV(file)
	if err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		teacher := strings.TrimSpace(record[teacherColumn])
		room := strings.TrimSpace(record[roomColumn])

		grade, err := getGrade(record[gradeColumn])
		if err != nil {
			return nil, fmt.Errorf("invalid Grade %s for teacher %s: %w", record[gradeColumn], teacher, err)
		}

		name := record[groupNameColumn]
		artIDs := normalizePreferences(record[artPreferencesStartColumn:artPreferencesEndColumn])
		sciIDs := normalizePreferences(record[sciencePreferencesStartColumn:sciencePreferencesEndColumn])

		parentIDs := make(map[string]struct{})
		for _, parentID := range normalizeParentIDs(record[parentWorkshopsColumn]) {
			if parentID == "0" || parentID == "" {
				continue
			}
			parentIDs[parentID] = struct{}{}
		}

		baseID := formatGroupID(teacher, grade, name)
		idCounts[baseID]++
		id := baseID
		if idCounts[baseID] > 1 {
			id = fmt.Sprintf("%s-%d", baseID, idCounts[baseID])
		}

		groups = append(groups, &Group{
			Teacher:             teacher,
			Room:                room,
			Grade:               grade,
			Name:                name,
			Students:            strings.Split(record[studentsColumn], ","),
			ArtIDs:              artIDs,
			SciIDs:              sciIDs,
			ParentIDs:           parentIDs,
			ParentBookingIssues: make(map[string]string),
			ID:                  id,
		})
	}

	return groups, nil
}

func (g *Group) NumStudents() int {
	return len(g.Students)
}

func (g *Group) PreferencesForKind(kind int) []string {
	if kind == model.ArtWorkshop {
		return g.ArtIDs
	}
	return g.SciIDs
}

func (g *Group) SortedParentIDs() []string {
	parentIDs := slices.Collect(maps.Keys(g.ParentIDs))
	sort.Strings(parentIDs)
	return parentIDs
}

func normalizePreferences(preferences []string) []string {
	normalized := make([]string, 0, len(preferences))
	seen := make(map[string]struct{}, len(preferences))

	for _, preference := range preferences {
		preference = normalizeWorkshopReference(preference)
		if preference == "" {
			continue
		}
		if _, ok := seen[preference]; ok {
			continue
		}

		seen[preference] = struct{}{}
		normalized = append(normalized, preference)
	}

	for len(normalized) < len(preferences) {
		normalized = append(normalized, "")
	}

	return normalized
}

func formatGroupID(teacher string, grade int, name string) string {
	return fmt.Sprintf("%s-%s-%s", strings.ReplaceAll(teacher, " ", "_"), gradeIDComponent(grade), strings.ReplaceAll(name, " ", "_"))
}

func gradeIDComponent(grade int) string {
	if grade == model.TKGrade {
		return "TK"
	}
	return fmt.Sprintf("%d", grade)
}

func normalizeParentIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Fields(raw)
	if len(parts) == 1 {
		normalized := normalizeWorkshopReference(parts[0])
		if normalized == "" {
			return nil
		}
		return []string{normalized}
	}

	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeWorkshopReference(part)
		if part == "" {
			continue
		}
		normalized = append(normalized, part)
	}

	return normalized
}

func normalizeWorkshopReference(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	matches := workshopIDPattern.FindStringSubmatch(value)
	if len(matches) == 2 {
		return strings.ToUpper(matches[1])
	}

	return value
}

func readAndParseCSV(file string) (*csv.Reader, error) {
	csvFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(csvFile)
	_, err = reader.Read()
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("empty csv file: %w", err)
	}
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func getGrade(grade string) (int, error) {
	return model.ParseGrade(grade)
}
