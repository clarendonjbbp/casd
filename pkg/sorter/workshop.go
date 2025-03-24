package sorter

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/exp/maps"

	"github.com/dariubs/percent"
)

const (
	numSessions = 4

	ArtWorkshop = iota
	SciWorkshop
)

type Workshop struct {
	Kind            int
	id              string
	Name            string
	MinGrade        int
	MaxGrade        int
	Capacity        int
	SpotsAvailable  []int
	SessionsOffered int
	room            string

	sessionGroups map[int][]*Group
}

func ReadWorkshops(file string, kind int) (map[string]*Workshop, error) {
	workshops := make(map[string]*Workshop)

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

		// Parse fields as needed
		id, name, found := strings.Cut(record[0], "-")
		if !found {
			return nil, fmt.Errorf("Invalid workshop name \"%s\"", record[0])
		}
		id = strings.Trim(id, " ")
		name = strings.Trim(name, " ")

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

		sessionCapacities := make([]int, numSessions)
		var sessionsOffered int
		for i := 2; i < 6; i++ {
			if strings.ToLower(record[i]) == "y" {
				sessionCapacities[i-2] = capacity
				sessionsOffered++
			} else {
				sessionCapacities[i-2] = -1
			}
		}

		// Append to array
		workshops[id] = &Workshop{
			Kind:            kind,
			id:              id,
			Name:            name,
			MinGrade:        minGrade,
			MaxGrade:        maxGrade,
			Capacity:        capacity,
			SpotsAvailable:  sessionCapacities,
			SessionsOffered: sessionsOffered,
			room:            record[7],
			sessionGroups:   make(map[int][]*Group),
		}
	}

	return workshops, nil
}

func (w Workshop) GetAvailableSessions(group *Group) []int {
	var availableSessions []int
	maxRemainingSlots := 0
	for i, sessionCapacity := range w.SpotsAvailable {
		if group.workshops[i] == nil && sessionCapacity != -1 {
			remainingSlots := sessionCapacity - group.NumStudents()
			if remainingSlots > maxRemainingSlots {
				maxRemainingSlots = remainingSlots
				availableSessions = []int{i}
			} else if remainingSlots == maxRemainingSlots {
				availableSessions = append(availableSessions, i)
			}
		}
	}

	return availableSessions
}

func (w Workshop) WithinGradeRange(grade int) bool {
	return w.MinGrade <= grade && grade <= w.MaxGrade
}

func (w *Workshop) TakeSession(session int, group *Group) {
	w.SpotsAvailable[session] -= len(group.students)

	groups := w.sessionGroups[session]
	groups = append(groups, group)
	w.sessionGroups[session] = groups
}

func (w *Workshop) UnbookSession(session int, group *Group) {
	w.SpotsAvailable[session] += len(group.students)

	groups := w.sessionGroups[session]
	for i := range groups {
		if groups[i].id == group.id {
			log.Println("Found group to remove")
			groups = remove(groups, i)
			break
		}
	}

	w.sessionGroups[session] = groups
}

func (w Workshop) GetGroupsForSession(session int) []*Group {
	return w.sessionGroups[session]
}

func (w Workshop) Utilization(sessionNumber int) int {
	if w.SpotsAvailable[sessionNumber] == -1 {
		return -1
	}

	return int(percent.PercentOf(w.Capacity-w.SpotsAvailable[sessionNumber], w.Capacity))
}

func (w Workshop) UtilizationWithoutGroup(sessionNumber int, group *Group) int {
	if w.SpotsAvailable[sessionNumber] == -1 {
		return -1
	}

	return int(percent.PercentOf(w.Capacity-(w.SpotsAvailable[sessionNumber]+group.NumStudents()), w.Capacity))
}

func (w Workshop) OverallUtilization() int {
	overallSpotsTaken := 0
	overallCapacity := 0
	for _, spotsAvailable := range w.SpotsAvailable {
		if spotsAvailable != -1 {
			overallCapacity += w.Capacity
			overallSpotsTaken += w.Capacity - spotsAvailable
		}
	}

	return int(percent.PercentOf(overallSpotsTaken, overallCapacity))
}

func (w Workshop) GetID() string {
	return w.id
}

func (w Workshop) Print(wr io.Writer) {
	fmt.Fprintf(wr, "ID: %s  \n", w.id)
	fmt.Fprintf(wr, "Name: %s  \n", w.Name)
	fmt.Fprintf(wr, "Capacity per session: %d  \n", w.Capacity)
	fmt.Fprintf(wr, "Overall utilization: %d%%  \n", w.OverallUtilization())
	fmt.Fprintf(wr, "Schedule  \n")
	fmt.Fprintln(wr, "| Utilization | Students |")
	fmt.Fprintln(wr, "| --------- | -------- |")
	for i := 0; i < numSessions; i++ {
		if w.SpotsAvailable[i] == -1 {
			fmt.Fprintln(wr, "| - | - |")

		} else {
			fmt.Fprintf(wr, "| %d%% | ", w.Utilization(i))

			groups := w.sessionGroups[i]
			for _, group := range groups {
				fmt.Fprintf(wr, "%v,", strings.Join(group.students, ","))
			}
			fmt.Fprintf(wr, " |\n")
		}
	}
	fmt.Fprintf(wr, "\n---\n\n")
}

func SortWorkshopsByOverallUtilization(workshops map[string]*Workshop) []*Workshop {
	sortedWorkshops := maps.Values(workshops)

	slices.SortFunc(sortedWorkshops, func(a, b *Workshop) int {
		if n := cmp.Compare(a.OverallUtilization(), b.OverallUtilization()); n != 0 {
			return n
		}
		return cmp.Compare(a.SessionsOffered, b.SessionsOffered)
	})

	return sortedWorkshops
}

func GetUnderutilizedSessions(minUtilization int, workshops map[string]*Workshop) ([]*Workshop, []int) {
	var underutilizedWorkshops []*Workshop
	var underutilizedWorkshopSessions []int
	for _, workshop := range workshops {
		for i := 0; i < numSessions; i++ {
			if workshop.SpotsAvailable[i] == -1 {
				continue
			}
			if workshop.Utilization(i) < minUtilization {
				underutilizedWorkshops = append(underutilizedWorkshops, workshop)
				underutilizedWorkshopSessions = append(underutilizedWorkshopSessions, i)
			}
		}
	}

	return underutilizedWorkshops, underutilizedWorkshopSessions
}

func PrintWorkshops(wr io.Writer, workshops map[string]*Workshop) {
	for _, workshop := range workshops {
		workshop.Print(wr)
	}
}

func readAndParseCSV(file string) (*csv.Reader, error) {
	csvFile, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	//defer csvFile.Close()

	// Parse the file
	reader := csv.NewReader(csvFile)

	// Dump the header line
	_, err = reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("Empty csv file: %v", err)
	}
	if err != nil {
		log.Fatal(err)
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

func idToKind(id string) int {
	if id[0] == 'A' {
		return ArtWorkshop
	}

	return SciWorkshop
}

func remove(slice []*Group, s int) []*Group {
	return append(slice[:s], slice[s+1:]...)
}

func (w Workshop) PrintHTML(wr io.Writer) {
	fmt.Fprintf(wr, "<div class='workshop'>\n")
	fmt.Fprintf(wr, "<h3>%s</h3>\n", w.Name)
	fmt.Fprintf(wr, "<p><strong>ID:</strong> %s</p>\n", w.id)
	fmt.Fprintf(wr, "<p><strong>Capacity per session:</strong> %d</p>\n", w.Capacity)
	fmt.Fprintf(wr, "<p><strong>Overall utilization:</strong> %d%%</p>\n", w.OverallUtilization())

	fmt.Fprintf(wr, "<table class='schedule'>\n")
	fmt.Fprintf(wr, "<thead>\n<tr>\n<th>Session</th>\n<th>Utilization</th>\n<th>Students</th>\n</tr>\n</thead>\n")
	fmt.Fprintf(wr, "<tbody>\n")

	for i := 0; i < numSessions; i++ {
		fmt.Fprintf(wr, "<tr>\n")
		fmt.Fprintf(wr, "<td>%d</td>\n", i+1)

		if w.SpotsAvailable[i] == -1 {
			fmt.Fprintf(wr, "<td>-</td>\n<td>-</td>\n")
		} else {
			utilization := w.Utilization(i)
			utilizationClass := "normal"
			if utilization < 30 {
				utilizationClass = "low"
			} else if utilization > 80 {
				utilizationClass = "high"
			}

			fmt.Fprintf(wr, "<td class='utilization %s'>%d%%</td>\n", utilizationClass, utilization)
			fmt.Fprintf(wr, "<td class='students'>")

			groups := w.sessionGroups[i]
			var studentNames []string
			for _, group := range groups {
				studentNames = append(studentNames, group.students...)
			}
			fmt.Fprintf(wr, "%s", strings.Join(studentNames, ", "))
			fmt.Fprintf(wr, "</td>\n")
		}
		fmt.Fprintf(wr, "</tr>\n")
	}

	fmt.Fprintf(wr, "</tbody>\n</table>\n")
	fmt.Fprintf(wr, "</div>\n")
}

func writeWorkshopsHTMLStyle(wr io.Writer) {
	// Write CSS styles
	fmt.Fprintf(wr, `<style>
.workshop {
    margin-bottom: 2em;
    padding: 1em;
    border: 1px solid #ddd;
    border-radius: 4px;
}
.workshop h3 {
    margin-top: 0;
    color: #333;
}
.schedule {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 1em;
}
.schedule th, .schedule td {
    padding: 8px;
    text-align: left;
    border: 1px solid #ddd;
}
.schedule th {
    background-color: #f5f5f5;
}
.utilization {
    font-weight: bold;
}
.utilization.low {
    color: #dc3545;
}
.utilization.high {
    color: #28a745;
}
.students {
    font-size: 0.9em;
}
</style>
`)
}

func PrintWorkshopsHTML(wr io.Writer, workshops map[string]*Workshop) {
	// Write CSS first
	writeWorkshopsHTMLStyle(wr)

	// Sort workshops by ID for consistent display
	var sortedIDs []string
	for id := range workshops {
		sortedIDs = append(sortedIDs, id)
	}
	slices.Sort(sortedIDs)

	// Write each workshop
	for _, id := range sortedIDs {
		workshops[id].PrintHTML(wr)
	}
}
