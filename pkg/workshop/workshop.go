package workshop

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

type Workshop struct {
	Kind            int
	ID              string
	Name            string
	MinGrade        int
	MaxGrade        int
	Capacity        int
	OfferedSessions []bool
	SessionsOffered int
	Room            string
}

func ReadWorkshops(file string, kind int) (map[string]*Workshop, error) {
	workshops := make(map[string]*Workshop)

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

		id, name, found := strings.Cut(record[0], "-")
		if !found {
			return nil, fmt.Errorf("invalid workshop name %q", record[0])
		}
		id = strings.TrimSpace(id)
		name = strings.TrimSpace(name)

		grades := strings.Split(record[1], "-")
		minGrade, err := getGrade(grades[0])
		if err != nil {
			return nil, err
		}
		maxGrade, err := getGrade(grades[1])
		if err != nil {
			return nil, err
		}

		capacity, err := strconv.Atoi(record[6])
		if err != nil {
			return nil, err
		}

		offeredSessions := make([]bool, 4)
		var sessionsOffered int
		for i := 2; i < 6; i++ {
			if strings.ToLower(record[i]) == "y" {
				offeredSessions[i-2] = true
				sessionsOffered++
			}
		}

		workshops[id] = &Workshop{
			Kind:            kind,
			ID:              id,
			Name:            name,
			MinGrade:        minGrade,
			MaxGrade:        maxGrade,
			Capacity:        capacity,
			OfferedSessions: offeredSessions,
			SessionsOffered: sessionsOffered,
			Room:            record[7],
		}
	}

	return workshops, nil
}

func (w *Workshop) WithinGradeRange(grade int) bool {
	return w.MinGrade <= grade && grade <= w.MaxGrade
}

func (w *Workshop) IsSessionOffered(session int) bool {
	return session >= 0 && session < len(w.OfferedSessions) && w.OfferedSessions[session]
}

func (w *Workshop) GetID() string {
	return w.ID
}

func SortWorkshopsByOverallUtilization(workshops map[string]*Workshop, overallUtilization func(*Workshop) int) []*Workshop {
	sortedWorkshops := slices.Collect(maps.Values(workshops))

	slices.SortFunc(sortedWorkshops, func(a, b *Workshop) int {
		if n := cmp.Compare(overallUtilization(a), overallUtilization(b)); n != 0 {
			return n
		}
		return cmp.Compare(a.SessionsOffered, b.SessionsOffered)
	})

	return sortedWorkshops
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
