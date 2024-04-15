package sorter

import (
	"fmt"
	"io"
	"log"
	"slices"
	"sort"
	"strings"
)

type Group struct {
	ParentIDs map[string]struct{}
	Teacher   string
	Name      string

	Grade    int
	students []string
	ArtIDs   []string
	SciIDs   []string

	workshops []*Workshop
	id        string
}

func ReadGroups(file string) ([]*Group, error) {
	var groups []*Group

	reader, err := readAndParseCSV(file)
	if err != nil {
		return nil, err
	}

	for {
		// Read each record from csv
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		teacher := strings.TrimSpace(record[0])

		grade, err := getGrade(record[2])
		if err != nil {
			return nil, fmt.Errorf("invalid Grade %s for teacher %s: %w", record[2], teacher, err)
		}

		name := record[3]
		artIDs := record[5:9]
		sciIDs := record[9:13]

		parentIDs := make(map[string]struct{})
		parentIDsRaw := strings.Split(record[13], " ")
		for _, parentID := range parentIDsRaw {
			if parentID == "0" || parentID == "" {
				continue
			}
			parentIDs[parentID] = struct{}{}
		}

		groups = append(groups, &Group{
			Teacher:   teacher,
			Grade:     grade,
			Name:      name,
			students:  strings.Split(record[4], ","),
			ArtIDs:    artIDs,
			SciIDs:    sciIDs,
			workshops: make([]*Workshop, 4),
			ParentIDs: parentIDs,
			id:        fmt.Sprintf("%s-%d-%s", strings.ReplaceAll(teacher, " ", "_"), grade, strings.ReplaceAll(name, " ", "_")),
		})
	}

	return groups, nil
}

func (g Group) IsEnrolledInWorkshop(id string) bool {
	for _, workshop := range g.workshops {
		if workshop == nil {
			continue
		}
		if workshop.id == id {
			return true
		}
	}

	return false
}

func (g Group) SessionsBooked(kind int) int {
	booked := 0
	for _, workshop := range g.workshops {
		if workshop != nil {
			workshopKind := idToKind(workshop.id)
			if workshopKind == kind {
				booked++
			}
		}
	}

	return booked
}

func (g *Group) BookWorkshop(session int, workshop *Workshop) {
	g.workshops[session] = workshop
}

func (g *Group) HowPreferredIsBookedArtWorkshop(sessionNumber int) int {
	workshop := g.workshops[sessionNumber]

	if idToKind(workshop.id) != ArtWorkshop {
		return -1
	}
	if _, ok := g.ParentIDs[workshop.id]; ok {
		return 5
	}

	for i := range g.ArtIDs {
		if g.ArtIDs[i] == workshop.id {
			return i + 1
		}
	}

	return 0

}

func (g *Group) HowPreferredIsBookedSciWorkshop(sessionNumber int) int {
	workshop := g.workshops[sessionNumber]

	if idToKind(workshop.id) != SciWorkshop {
		return -1
	}
	if _, ok := g.ParentIDs[workshop.id]; ok {
		return 5
	}

	for i := range g.SciIDs {
		if g.SciIDs[i] == workshop.id {
			return i + 1
		}
	}

	return 0

}

func (g *Group) GetSatisfaction() int {
	satisfaction := 0

	for _, workshop := range g.workshops {
		if workshop == nil {
			return 0
		}
		if _, ok := g.ParentIDs[workshop.id]; ok {
			satisfaction += 5
			continue
		}

		var preferences []string
		if workshop.Kind == ArtWorkshop {
			preferences = g.ArtIDs
		} else {
			preferences = g.SciIDs
		}

		for i := range preferences {
			if workshop.id == preferences[i] {
				satisfaction += 4 - i
			}
		}
	}

	return satisfaction
}

func (g *Group) GetWorkshop(session int) *Workshop {
	return g.workshops[session]
}

func (g *Group) HowPreferredIsBookedWorkshop(session int) int {
	workshop := g.workshops[session]

	if _, ok := g.ParentIDs[workshop.id]; ok {
		return 5
	}

	var preferences []string
	if idToKind(workshop.id) != ArtWorkshop {
		preferences = g.ArtIDs
	} else {
		preferences = g.SciIDs
	}
	for i := range preferences {
		if preferences[i] == workshop.id {
			return i + 1
		}
	}

	return 0

}

func (g Group) NumStudents() int {
	return len(g.students)
}

func (g *Group) Print(w io.Writer) {
	fmt.Fprintf(w, "Teacher = %s  \n", g.Teacher)
	if g.Grade == 0 {
		fmt.Fprintf(w, "Grade = K  \n")
	} else {
		fmt.Fprintf(w, "Grade = %d  \n", g.Grade)
	}
	fmt.Fprintf(w, "ID = %s  \n", g.id)
	//fmt.Fprintf(w, "Satisfaction =  %d\n", g.GetSatisfaction())
	fmt.Fprintf(w, "Students =  %v  \n", strings.Join(g.students, ","))
	//fmt.Fprintf(w, "Art Preferences = %v  \n", g.ArtIDs)
	if len(g.ParentIDs) > 0 {
		parentIDs := []string{}
		for parentID := range g.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		fmt.Fprintf(w, "Group contains child of presenter or assistant of workshop = %v  \n", strings.Join(parentIDs, ","))
	}
	fmt.Fprintln(w, "Schedule")
	fmt.Fprintln(w, "| ID | Class | Room |")
	fmt.Fprintln(w, "| -- | ----- | ---- |")
	for _, workshop := range g.workshops {
		if workshop != nil {
			fmt.Fprintf(w, "| %s | %s | %s |\n", workshop.id, workshop.Name, workshop.room)
		} else {
			fmt.Fprintf(w, "| - | - | - |\n")
			log.Printf("====UNFILLED SLOT====\n")
		}
	}
	fmt.Fprintf(w, "\n---\n\n")
}

func SortGroupsByPrefferedArtSession(sessionNumber int, groups []*Group) []*Group {
	sortedGroups := groups

	slices.SortFunc(sortedGroups, func(a, b *Group) int {
		if a.HowPreferredIsBookedArtWorkshop(sessionNumber) < b.HowPreferredIsBookedArtWorkshop(sessionNumber) {
			return -1
		}
		if a.HowPreferredIsBookedArtWorkshop(sessionNumber) > b.HowPreferredIsBookedArtWorkshop(sessionNumber) {
			return 1
		}

		return 0
	})

	return sortedGroups

}

func PrintGroups(w io.Writer, groups []*Group) {
	sort.Slice(groups, func(i, j int) bool {
		idi := fmt.Sprintf("%s-%d-%s  \n", strings.ReplaceAll(groups[i].Teacher, " ", "_"), groups[i].Grade, strings.ReplaceAll(groups[i].Name, " ", "_"))
		idj := fmt.Sprintf("%s-%d-%s  \n", strings.ReplaceAll(groups[j].Teacher, " ", "_"), groups[j].Grade, strings.ReplaceAll(groups[j].Name, " ", "_"))
		return idi < idj
	})
	for _, group := range groups {
		group.Print(w)
	}
}
