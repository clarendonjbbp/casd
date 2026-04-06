package group

import (
	"encoding/csv"
	"fmt"
	"io"
	"maps"
	"os"
	"regexp"
	"slices"
	"sort"
	"strconv"
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

func ReadGroups(file string) ([]*Group, error) {
	var groups []*Group
	idCounts := make(map[string]int)

	reader, err := readAndParseCSV(file)
	if err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		teacher := strings.TrimSpace(record[0])
		room := strings.TrimSpace(record[1])

		grade, err := getGrade(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid Grade %s for teacher %s: %w", record[2], teacher, err)
		}

		name := record[3]
		artIDs := normalizePreferences(record[5:9])
		sciIDs := normalizePreferences(record[9:13])

		parentIDs := make(map[string]struct{})
		for _, parentID := range normalizeParentIDs(record[13]) {
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
			Students:            strings.Split(record[4], ","),
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
	return fmt.Sprintf("%s-%d-%s", strings.ReplaceAll(teacher, " ", "_"), grade, strings.ReplaceAll(name, " ", "_"))
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
	if err == io.EOF {
		return nil, fmt.Errorf("empty csv file: %v", err)
	}
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func getGrade(grade string) (int, error) {
	if strings.ToLower(grade) == "k" {
		return 0, nil
	}

	if grade == "4/5" {
		return 4, nil
	}

	return strconv.Atoi(grade)
}
