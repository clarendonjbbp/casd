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
		artIDs := normalizePreferences(record[5:9])
		sciIDs := normalizePreferences(record[9:13])

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

	for session := range g.workshops {
		if g.workshops[session] == nil {
			return 0
		}
		satisfaction += g.SessionSatisfactionPoints(session)
	}

	return satisfaction
}

func (g *Group) MaxSatisfaction() int {
	artParentCount := 0
	sciParentCount := 0
	for parentID := range g.ParentIDs {
		if idToKind(parentID) == ArtWorkshop {
			artParentCount++
		} else {
			sciParentCount++
		}
	}

	artParentCount = min(artParentCount, numArtSessions)
	sciParentCount = min(sciParentCount, numSciSessions)

	maxSatisfaction := artParentCount*5 + sciParentCount*5
	maxSatisfaction += maxPreferencePoints(g.ArtIDs, numArtSessions-artParentCount)
	maxSatisfaction += maxPreferencePoints(g.SciIDs, numSciSessions-sciParentCount)

	return maxSatisfaction
}

func (g *Group) SatisfactionPercent() int {
	maxSatisfaction := g.MaxSatisfaction()
	if maxSatisfaction == 0 {
		return 0
	}

	return g.GetSatisfaction() * 100 / maxSatisfaction
}

func (g *Group) SessionSatisfactionPoints(session int) int {
	workshop := g.GetWorkshop(session)
	if workshop == nil {
		return 0
	}

	if _, ok := g.ParentIDs[workshop.id]; ok {
		return 5
	}

	preferenceRank := g.HowPreferredIsBookedWorkshop(session)
	if preferenceRank < 1 || preferenceRank > 4 {
		return 0
	}

	return 5 - preferenceRank
}

func (g *Group) SessionSatisfactionLabel(session int) string {
	workshop := g.GetWorkshop(session)
	if workshop == nil {
		return "Not Scheduled"
	}

	if _, ok := g.ParentIDs[workshop.id]; ok {
		return "Parent Workshop"
	}

	switch g.HowPreferredIsBookedWorkshop(session) {
	case 1:
		return "1st Choice"
	case 2:
		return "2nd Choice"
	case 3:
		return "3rd Choice"
	case 4:
		return "4th Choice"
	default:
		return "Fallback"
	}
}

func (g *Group) GetWorkshop(session int) *Workshop {
	return g.workshops[session]
}

func (g *Group) HowPreferredIsBookedWorkshop(session int) int {
	workshop := g.workshops[session]

	if _, ok := g.ParentIDs[workshop.id]; ok {
		return 5
	}

	preferences := g.preferencesForKind(idToKind(workshop.id))
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

func normalizePreferences(preferences []string) []string {
	normalized := make([]string, 0, len(preferences))
	seen := make(map[string]struct{}, len(preferences))

	for _, preference := range preferences {
		preference = strings.TrimSpace(preference)
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

func (g *Group) preferencesForKind(kind int) []string {
	if kind == ArtWorkshop {
		return g.ArtIDs
	}
	return g.SciIDs
}

func maxPreferencePoints(preferences []string, slots int) int {
	points := 0
	counted := 0

	for i, preference := range preferences {
		if counted == slots {
			break
		}
		if strings.TrimSpace(preference) == "" {
			continue
		}

		points += 4 - i
		counted++
	}

	return points
}

func (g *Group) Print(w io.Writer) {
	_, _ = fmt.Fprintf(w, "Teacher = %s  \n", g.Teacher)
	if g.Grade == 0 {
		_, _ = fmt.Fprintf(w, "Grade = K  \n")
	} else {
		_, _ = fmt.Fprintf(w, "Grade = %d  \n", g.Grade)
	}
	_, _ = fmt.Fprintf(w, "ID = %s  \n", g.id)
	_, _ = fmt.Fprintf(w, "Satisfaction = %d%%  \n", g.SatisfactionPercent())
	_, _ = fmt.Fprintf(w, "Students =  %v  \n", strings.Join(g.students, ","))
	//fmt.Fprintf(w, "Art Preferences = %v  \n", g.ArtIDs)
	if len(g.ParentIDs) > 0 {
		parentIDs := []string{}
		for parentID := range g.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		_, _ = fmt.Fprintf(w, "Group contains child of presenter or assistant of workshop = %v  \n", strings.Join(parentIDs, ","))
	}
	_, _ = fmt.Fprintln(w, "Schedule")
	_, _ = fmt.Fprintln(w, "| ID | Class | Room | Match |")
	_, _ = fmt.Fprintln(w, "| -- | ----- | ---- | ----- |")
	for session, workshop := range g.workshops {
		if workshop != nil {
			_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s |\n", workshop.id, workshop.Name, workshop.room, g.SessionSatisfactionLabel(session))
		} else {
			_, _ = fmt.Fprintf(w, "| - | - | - | Not Scheduled |\n")
			log.Printf("====UNFILLED SLOT====\n")
		}
	}
	_, _ = fmt.Fprintf(w, "\n---\n\n")
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

func (g *Group) PrintHTML(w io.Writer) {
	_, _ = fmt.Fprintf(w, "<div class='group'>\n")
	_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n", g.id)

	_, _ = fmt.Fprintf(w, "<div class='group-details'>\n")
	_, _ = fmt.Fprintf(w, "<p><strong>Teacher:</strong> %s</p>\n", g.Teacher)
	if g.Grade == 0 {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> K</p>\n")
	} else {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> %d</p>\n", g.Grade)
	}
	_, _ = fmt.Fprintf(w, "<p><strong>Students:</strong> %s</p>\n", strings.Join(g.students, ", "))
	_, _ = fmt.Fprintf(w, "<p><strong>Satisfaction:</strong> %d%%</p>\n", g.SatisfactionPercent())

	if len(g.ParentIDs) > 0 {
		parentIDs := make([]string, 0, len(g.ParentIDs))
		for parentID := range g.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		sort.Strings(parentIDs)
		_, _ = fmt.Fprintf(w, "<p><strong>Parent Workshops:</strong> %s</p>\n", strings.Join(parentIDs, ", "))
	}
	_, _ = fmt.Fprintf(w, "</div>\n")

	_, _ = fmt.Fprintf(w, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(w, "<thead>\n<tr>\n<th>Time</th>\n<th>Workshop ID</th>\n<th>Workshop Name</th>\n<th>Room</th>\n<th>Match</th>\n</tr>\n</thead>\n")
	_, _ = fmt.Fprintf(w, "<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(SessionTimes); i++ {
		_, _ = fmt.Fprintf(w, "<tr>\n")
		if i == 2 { // Recess row
			_, _ = fmt.Fprintf(w, "<td colspan='4' class='recess'>%s</td>\n", SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(w, "<td>%s</td>\n", SessionTimes[i])
			if workshopIndex < len(g.workshops) && g.workshops[workshopIndex] != nil {
				workshop := g.workshops[workshopIndex]
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.id)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.Name)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.room)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", g.SessionSatisfactionLabel(workshopIndex))
			} else {
				_, _ = fmt.Fprintf(w, "<td colspan='4' class='unfilled'>Not Scheduled</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(w, "</tr>\n")
	}

	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n")
	_, _ = fmt.Fprintf(w, "</div>\n")
}

func PrintGroupsHTML(w io.Writer, groups []*Group) {
	// Write CSS styles
	_, _ = fmt.Fprintf(w, `<style>
.group {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.95), rgba(244,250,255,0.95));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.1);
}
.group h3 {
    margin-top: 0;
    color: #141f26;
    border-bottom: 1px solid rgba(36, 55, 76, 0.14);
    padding-bottom: 0.6em;
    font-family: "Chewy", "Marker Felt", "Chalkboard SE", "Comic Sans MS", "Trebuchet MS", sans-serif;
    font-size: 1.35rem;
    text-transform: uppercase;
}
.group-details {
    margin: 1em 0;
    display: grid;
    gap: 0.35em;
}
.group-details p {
    margin: 0;
    color: #465963;
}
.group .schedule {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 1em;
    overflow: hidden;
    border-radius: 18px;
}
.group .schedule th,
.group .schedule td {
    padding: 10px 12px;
    text-align: left;
    border: 1px solid rgba(125, 138, 142, 0.2);
}
.group .schedule th {
    background-color: #e9f4fd;
    font-weight: bold;
    color: #1d4667;
}
.group .schedule .unfilled {
    color: #c53a44;
    text-align: center;
    font-style: italic;
    background: #fff2f3;
}
.group .schedule .recess {
    background-color: #eef7ec;
    text-align: center;
    font-style: italic;
    color: #4a6c55;
}
</style>
`)

	// Sort groups by teacher name for consistent display
	sortedGroups := make([]*Group, len(groups))
	copy(sortedGroups, groups)
	slices.SortFunc(sortedGroups, func(a, b *Group) int {
		return strings.Compare(fmt.Sprintf("%d-%s-%s", a.Grade, a.Teacher, a.Name), fmt.Sprintf("%d-%s-%s", b.Grade, b.Teacher, b.Name))
	})

	// Write each group
	for _, group := range sortedGroups {
		group.PrintHTML(w)
	}
}
