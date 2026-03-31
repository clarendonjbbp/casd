package sorter

import (
	"cmp"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
)

const (
	numSessions = 4

	ArtWorkshop = iota
	SciWorkshop
)

// SessionTimes defines the schedule for all sessions including recess
var SessionTimes = []string{
	"9:40 - 10:10 am",
	"10:15 - 10:45 am",
	"10:50 - 11:05 am (Recess)",
	"11:10 - 11:40 am",
	"11:45 am - 12:15 pm",
}

type Workshop struct {
	Kind            int
	id              string
	Name            string
	MinGrade        int
	MaxGrade        int
	Capacity        int
	OfferedSessions []bool
	SessionsOffered int
	room            string
	schedule        *ScheduleState
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
			return nil, fmt.Errorf("invalid workshop name %q", record[0])
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

		offeredSessions := make([]bool, numSessions)
		var sessionsOffered int
		for i := 2; i < 6; i++ {
			if strings.ToLower(record[i]) == "y" {
				offeredSessions[i-2] = true
				sessionsOffered++
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
			OfferedSessions: offeredSessions,
			SessionsOffered: sessionsOffered,
			room:            record[7],
		}
	}

	return workshops, nil
}

func (w Workshop) GetAvailableSessions(group *Group) []int {
	if w.schedule == nil {
		return nil
	}
	return w.schedule.AvailableSessions(&w, group)
}

func (w Workshop) WithinGradeRange(grade int) bool {
	return w.MinGrade <= grade && grade <= w.MaxGrade
}

func (w Workshop) IsSessionOffered(session int) bool {
	return session >= 0 && session < len(w.OfferedSessions) && w.OfferedSessions[session]
}

func (w *Workshop) TakeSession(session int, group *Group) {
	if w.schedule == nil {
		w.schedule = &ScheduleState{}
	}
	if group.schedule == nil {
		group.schedule = w.schedule
	}
	w.schedule.Book(group, w, session)
}

func (w *Workshop) UnbookSession(session int, group *Group) {
	if w.schedule == nil {
		return
	}
	if groups := w.schedule.GroupsForWorkshopSession(w, session); len(groups) > 0 {
		log.Println("Found group to remove")
	}
	w.schedule.Unbook(group, w, session)
}

func (w Workshop) GetGroupsForSession(session int) []*Group {
	if w.schedule == nil {
		return nil
	}
	return w.schedule.GroupsForWorkshopSession(&w, session)
}

func (w Workshop) SpotsAvailable(sessionNumber int) int {
	if w.schedule == nil {
		return -1
	}
	return w.schedule.SpotsAvailable(&w, sessionNumber)
}

func (w Workshop) Utilization(sessionNumber int) int {
	if w.schedule == nil {
		return -1
	}
	return w.schedule.Utilization(&w, sessionNumber)
}

func (w Workshop) UtilizationWithoutGroup(sessionNumber int, group *Group) int {
	if w.schedule == nil {
		return -1
	}
	return w.schedule.UtilizationWithoutGroup(&w, sessionNumber, group)
}

func (w Workshop) OverallUtilization() int {
	if w.schedule == nil {
		return 0
	}
	return w.schedule.OverallUtilization(&w)
}

func (w Workshop) GetID() string {
	return w.id
}

func (w Workshop) Print(wr io.Writer) {
	_, _ = fmt.Fprintf(wr, "ID: %s  \n", w.id)
	_, _ = fmt.Fprintf(wr, "Name: %s  \n", w.Name)
	_, _ = fmt.Fprintf(wr, "Capacity per session: %d  \n", w.Capacity)
	_, _ = fmt.Fprintf(wr, "Overall utilization: %d%%  \n", w.OverallUtilization())
	_, _ = fmt.Fprintf(wr, "Schedule  \n")
	_, _ = fmt.Fprintln(wr, "| Utilization | Students |")
	_, _ = fmt.Fprintln(wr, "| --------- | -------- |")
	for i := 0; i < numSessions; i++ {
		if !w.IsSessionOffered(i) {
			_, _ = fmt.Fprintln(wr, "| - | - |")

		} else {
			_, _ = fmt.Fprintf(wr, "| %d%% | ", w.Utilization(i))

			groups := w.GetGroupsForSession(i)
			for _, group := range groups {
				_, _ = fmt.Fprintf(wr, "%v,", strings.Join(group.students, ","))
			}
			_, _ = fmt.Fprintf(wr, " |\n")
		}
	}
	_, _ = fmt.Fprintf(wr, "\n---\n\n")
}

func SortWorkshopsByOverallUtilization(workshops map[string]*Workshop) []*Workshop {
	sortedWorkshops := slices.Collect(maps.Values(workshops))

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
			if !workshop.IsSessionOffered(i) {
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
		return nil, fmt.Errorf("empty csv file: %v", err)
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

func percentOf(part, total int) int {
	if total == 0 {
		return 0
	}
	return part * 100 / total
}

func (w Workshop) PrintHTML(wr io.Writer) {
	_, _ = fmt.Fprintf(wr, "<div class='workshop'>\n")
	_, _ = fmt.Fprintf(wr, "<h3>%s</h3>\n", w.Name)
	_, _ = fmt.Fprintf(wr, "<p><strong>ID:</strong> %s</p>\n", w.id)
	_, _ = fmt.Fprintf(wr, "<p><strong>Capacity per session:</strong> %d</p>\n", w.Capacity)
	_, _ = fmt.Fprintf(wr, "<p><strong>Overall utilization:</strong> %d%%</p>\n", w.OverallUtilization())

	_, _ = fmt.Fprintf(wr, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(wr, "<thead>\n<tr>\n<th>Time</th>\n<th>Utilization</th>\n<th>Students</th>\n</tr>\n</thead>\n")
	_, _ = fmt.Fprintf(wr, "<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(SessionTimes); i++ {
		_, _ = fmt.Fprintf(wr, "<tr>\n")
		if i == 2 { // Recess row
			_, _ = fmt.Fprintf(wr, "<td colspan='3' class='recess'>%s</td>\n", SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(wr, "<td>%s</td>\n", SessionTimes[i])

			if workshopIndex < numSessions && w.IsSessionOffered(workshopIndex) {
				utilization := w.Utilization(workshopIndex)
				utilizationClass := "normal"
				if utilization < 30 {
					utilizationClass = "low"
				} else if utilization > 80 {
					utilizationClass = "high"
				}

				_, _ = fmt.Fprintf(wr, "<td class='utilization %s'>%d%%</td>\n", utilizationClass, utilization)
				_, _ = fmt.Fprintf(wr, "<td class='students'>")

				groups := w.GetGroupsForSession(workshopIndex)
				var studentNames []string
				for _, group := range groups {
					studentNames = append(studentNames, group.students...)
				}
				_, _ = fmt.Fprintf(wr, "%s", strings.Join(studentNames, ", "))
				_, _ = fmt.Fprintf(wr, "</td>\n")
			} else {
				_, _ = fmt.Fprintf(wr, "<td>-</td>\n<td>-</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(wr, "</tr>\n")
	}

	_, _ = fmt.Fprintf(wr, "</tbody>\n</table>\n")
	_, _ = fmt.Fprintf(wr, "</div>\n")
}

func writeWorkshopsHTMLStyle(wr io.Writer) {
	// Write CSS styles
	_, _ = fmt.Fprintf(wr, `<style>
.workshop {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.95), rgba(244,250,255,0.95));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.1);
}
.workshop h3 {
    margin-top: 0;
    color: #141f26;
    font-family: "Chewy", "Marker Felt", "Chalkboard SE", "Comic Sans MS", "Trebuchet MS", sans-serif;
    font-size: 1.35rem;
    text-transform: uppercase;
}
.schedule {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 1em;
}
.schedule th, .schedule td {
    padding: 10px 12px;
    text-align: left;
    border: 1px solid rgba(125, 138, 142, 0.2);
}
.schedule th {
    background-color: #e9f4fd;
    color: #1d4667;
}
.utilization {
    font-weight: bold;
}
.utilization.low {
    color: #c53a44;
}
.utilization.high {
    color: #2f8b51;
}
.students {
    font-size: 0.9em;
}
.schedule .recess {
    background-color: #eef7ec;
    text-align: center;
    font-style: italic;
    color: #4a6c55;
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
