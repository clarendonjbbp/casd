package booking

import (
	"cmp"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand"
	"slices"
	"sort"
	"strings"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

type Booking struct {
	Group    *groupPkg.Group
	Workshop *workshopPkg.Workshop
	Session  int
}

type ScheduleState struct {
	bookings []*Booking
}

type ScheduleSummary struct {
	OverallSatisfactionPoints  int
	AverageSatisfactionPercent int
	GroupsWithPreferredArt     int
	GroupsWithPreferredScience int
	TotalGroups                int
}

func NewScheduleState(_ []*groupPkg.Group, _ ...map[string]*workshopPkg.Workshop) *ScheduleState {
	return &ScheduleState{}
}

func (s *ScheduleState) Book(group *groupPkg.Group, workshop *workshopPkg.Workshop, session int) {
	s.bookings = append(s.bookings, &Booking{
		Group:    group,
		Workshop: workshop,
		Session:  session,
	})
	delete(group.ParentBookingIssues, workshop.ID)
}

func (s *ScheduleState) Unbook(group *groupPkg.Group, workshop *workshopPkg.Workshop, session int) {
	for i := range s.bookings {
		booking := s.bookings[i]
		if booking.Group.ID == group.ID && booking.Workshop.ID == workshop.ID && booking.Session == session {
			s.bookings = slices.Delete(s.bookings, i, i+1)
			return
		}
	}
}

func (s *ScheduleState) WorkshopForGroupSession(group *groupPkg.Group, session int) *workshopPkg.Workshop {
	for _, booking := range s.bookings {
		if booking.Group.ID == group.ID && booking.Session == session {
			return booking.Workshop
		}
	}

	return nil
}

func (s *ScheduleState) GroupsForWorkshopSession(workshop *workshopPkg.Workshop, session int) []*groupPkg.Group {
	var groups []*groupPkg.Group
	for _, booking := range s.bookings {
		if booking.Workshop.ID == workshop.ID && booking.Session == session {
			groups = append(groups, booking.Group)
		}
	}

	return groups
}

func (s *ScheduleState) IsEnrolledInWorkshop(group *groupPkg.Group, id string) bool {
	for _, booking := range s.bookings {
		if booking.Group.ID == group.ID && booking.Workshop.ID == id {
			return true
		}
	}

	return false
}

func (s *ScheduleState) SessionsBooked(group *groupPkg.Group, kind int) int {
	booked := 0
	for _, booking := range s.bookings {
		if booking.Group.ID == group.ID && booking.Workshop.Kind == kind {
			booked++
		}
	}

	return booked
}

func (s *ScheduleState) SessionOccupied(group *groupPkg.Group, session int) bool {
	return s.WorkshopForGroupSession(group, session) != nil
}

func (s *ScheduleState) SpotsAvailable(workshop *workshopPkg.Workshop, session int) int {
	if !workshop.IsSessionOffered(session) {
		return -1
	}

	spotsTaken := 0
	for _, booking := range s.bookings {
		if booking.Workshop.ID == workshop.ID && booking.Session == session {
			spotsTaken += booking.Group.NumStudents()
		}
	}

	return workshop.Capacity - spotsTaken
}

func (s *ScheduleState) Utilization(workshop *workshopPkg.Workshop, session int) int {
	spotsAvailable := s.SpotsAvailable(workshop, session)
	if spotsAvailable == -1 {
		return -1
	}

	return percentOf(workshop.Capacity-spotsAvailable, workshop.Capacity)
}

func (s *ScheduleState) UtilizationWithoutGroup(workshop *workshopPkg.Workshop, session int, group *groupPkg.Group) int {
	spotsAvailable := s.SpotsAvailable(workshop, session)
	if spotsAvailable == -1 {
		return -1
	}

	return percentOf(workshop.Capacity-(spotsAvailable+group.NumStudents()), workshop.Capacity)
}

func (s *ScheduleState) OverallUtilization(workshop *workshopPkg.Workshop) int {
	overallSpotsTaken := 0
	overallCapacity := 0
	for session := 0; session < model.NumSessions; session++ {
		if !workshop.IsSessionOffered(session) {
			continue
		}

		overallCapacity += workshop.Capacity
		overallSpotsTaken += workshop.Capacity - s.SpotsAvailable(workshop, session)
	}

	return percentOf(overallSpotsTaken, overallCapacity)
}

func (s *ScheduleState) AvailableSessions(workshop *workshopPkg.Workshop, group *groupPkg.Group) []int {
	var availableSessions []int
	maxRemainingSlots := 0
	for session := 0; session < model.NumSessions; session++ {
		if s.SessionOccupied(group, session) {
			continue
		}
		spotsAvailable := s.SpotsAvailable(workshop, session)
		if spotsAvailable == -1 {
			continue
		}

		remainingSlots := spotsAvailable - group.NumStudents()
		if remainingSlots > maxRemainingSlots {
			maxRemainingSlots = remainingSlots
			availableSessions = []int{session}
		} else if remainingSlots == maxRemainingSlots {
			availableSessions = append(availableSessions, session)
		}
	}

	return availableSessions
}

func BookWorkshopIfAvailable(state *ScheduleState, workshop *workshopPkg.Workshop, group *groupPkg.Group) bool {
	if reason := BookingFailureReason(state, workshop, group); reason != "" {
		return false
	}

	sessions := state.AvailableSessions(workshop, group)
	randSession := sessions[rand.Intn(len(sessions))]
	state.Book(group, workshop, randSession)

	return true
}

func BookingFailureReason(state *ScheduleState, workshop *workshopPkg.Workshop, group *groupPkg.Group) string {
	if !workshop.WithinGradeRange(group.Grade) {
		return "grade mismatch"
	}
	if state.IsEnrolledInWorkshop(group, workshop.ID) {
		return "duplicate workshop"
	}
	if len(state.AvailableSessions(workshop, group)) == 0 {
		return "no available session with enough capacity"
	}

	return ""
}

func GetUnderutilizedSessions(state *ScheduleState, minUtilization int, workshops map[string]*workshopPkg.Workshop) ([]*workshopPkg.Workshop, []int) {
	var underutilizedWorkshops []*workshopPkg.Workshop
	var underutilizedWorkshopSessions []int
	for _, workshop := range workshops {
		for i := 0; i < model.NumSessions; i++ {
			if !workshop.IsSessionOffered(i) {
				continue
			}
			if state.Utilization(workshop, i) < minUtilization {
				underutilizedWorkshops = append(underutilizedWorkshops, workshop)
				underutilizedWorkshopSessions = append(underutilizedWorkshopSessions, i)
			}
		}
	}

	return underutilizedWorkshops, underutilizedWorkshopSessions
}

func SortWorkshopsByOverallUtilization(state *ScheduleState, workshops map[string]*workshopPkg.Workshop) []*workshopPkg.Workshop {
	sortedWorkshops := slices.Collect(maps.Values(workshops))

	slices.SortFunc(sortedWorkshops, func(a, b *workshopPkg.Workshop) int {
		if n := cmp.Compare(state.OverallUtilization(a), state.OverallUtilization(b)); n != 0 {
			return n
		}
		return cmp.Compare(a.SessionsOffered, b.SessionsOffered)
	})

	return sortedWorkshops
}

func GetWorkshop(group *groupPkg.Group, state *ScheduleState, session int) *workshopPkg.Workshop {
	return state.WorkshopForGroupSession(group, session)
}

func HowPreferredIsBookedWorkshop(group *groupPkg.Group, state *ScheduleState, session int) int {
	workshop := GetWorkshop(group, state, session)
	if workshop == nil {
		return 0
	}

	if _, ok := group.ParentIDs[workshop.ID]; ok {
		return 5
	}

	preferences := group.PreferencesForKind(workshop.Kind)
	for i := range preferences {
		if preferences[i] == workshop.ID {
			return i + 1
		}
	}

	return 0
}

func SessionSatisfactionPoints(group *groupPkg.Group, state *ScheduleState, session int) int {
	workshop := GetWorkshop(group, state, session)
	if workshop == nil {
		return 0
	}

	if _, ok := group.ParentIDs[workshop.ID]; ok {
		return 5
	}

	preferenceRank := HowPreferredIsBookedWorkshop(group, state, session)
	if preferenceRank < 1 || preferenceRank > 4 {
		return 0
	}

	return 5 - preferenceRank
}

func GetSatisfaction(group *groupPkg.Group, state *ScheduleState) int {
	satisfaction := 0

	for session := 0; session < model.NumSessions; session++ {
		if GetWorkshop(group, state, session) == nil {
			return 0
		}
		satisfaction += SessionSatisfactionPoints(group, state, session)
	}

	return satisfaction
}

func MaxSatisfaction(group *groupPkg.Group) int {
	artParentCount := 0
	sciParentCount := 0
	for parentID := range group.ParentIDs {
		if idToKind(parentID) == model.ArtWorkshop {
			artParentCount++
		} else {
			sciParentCount++
		}
	}

	artParentCount = min(artParentCount, model.NumArtSessions)
	sciParentCount = min(sciParentCount, model.NumSciSessions)

	maxSatisfaction := artParentCount*5 + sciParentCount*5
	maxSatisfaction += maxPreferencePoints(group.ArtIDs, model.NumArtSessions-artParentCount)
	maxSatisfaction += maxPreferencePoints(group.SciIDs, model.NumSciSessions-sciParentCount)

	return maxSatisfaction
}

func SatisfactionPercent(group *groupPkg.Group, state *ScheduleState) int {
	maxSatisfaction := MaxSatisfaction(group)
	if maxSatisfaction == 0 {
		return 0
	}

	return GetSatisfaction(group, state) * 100 / maxSatisfaction
}

func SessionSatisfactionLabel(group *groupPkg.Group, state *ScheduleState, session int) string {
	workshop := GetWorkshop(group, state, session)
	if workshop == nil {
		return "Not Scheduled"
	}

	if _, ok := group.ParentIDs[workshop.ID]; ok {
		return "Parent Workshop"
	}

	switch HowPreferredIsBookedWorkshop(group, state, session) {
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

func PreferenceRankForWorkshopID(group *groupPkg.Group, id string) int {
	preferences := group.PreferencesForKind(idToKind(id))
	for i := range preferences {
		if preferences[i] == id {
			return i + 1
		}
	}

	return 0
}

func CountPreferredWorkshopsOfKind(group *groupPkg.Group, state *ScheduleState, kind int) int {
	count := 0

	for session := 0; session < model.NumSessions; session++ {
		workshop := GetWorkshop(group, state, session)
		if workshop == nil || workshop.Kind != kind {
			continue
		}

		preferenceRank := HowPreferredIsBookedWorkshop(group, state, session)
		if preferenceRank >= 1 && preferenceRank <= 4 {
			count++
		}
	}

	return count
}

func HasPreferredWorkshopOfKind(group *groupPkg.Group, state *ScheduleState, kind int) bool {
	return CountPreferredWorkshopsOfKind(group, state, kind) > 0
}

func SortedParentBookingIssues(group *groupPkg.Group) []string {
	if len(group.ParentBookingIssues) == 0 {
		return nil
	}

	parentIDs := make([]string, 0, len(group.ParentBookingIssues))
	for parentID := range group.ParentBookingIssues {
		parentIDs = append(parentIDs, parentID)
	}
	sort.Strings(parentIDs)

	issues := make([]string, 0, len(parentIDs))
	for _, parentID := range parentIDs {
		issues = append(issues, fmt.Sprintf("%s (%s)", parentID, group.ParentBookingIssues[parentID]))
	}

	return issues
}

func CalculateScheduleSummary(groups []*groupPkg.Group, state *ScheduleState) ScheduleSummary {
	summary := ScheduleSummary{
		TotalGroups: len(groups),
	}

	if len(groups) == 0 {
		return summary
	}

	for _, group := range groups {
		summary.OverallSatisfactionPoints += GetSatisfaction(group, state)
		summary.AverageSatisfactionPercent += SatisfactionPercent(group, state)
		if HasPreferredWorkshopOfKind(group, state, model.ArtWorkshop) {
			summary.GroupsWithPreferredArt++
		}
		if HasPreferredWorkshopOfKind(group, state, model.SciWorkshop) {
			summary.GroupsWithPreferredScience++
		}
	}

	summary.AverageSatisfactionPercent /= len(groups)

	return summary
}

func PrintScheduleSummary(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	summary := CalculateScheduleSummary(groups, state)

	_, _ = fmt.Fprintln(w, "## Schedule Summary")
	_, _ = fmt.Fprintf(w, "- Overall satisfaction points: %d\n", summary.OverallSatisfactionPoints)
	_, _ = fmt.Fprintf(w, "- Average satisfaction: %d%%\n", summary.AverageSatisfactionPercent)
	_, _ = fmt.Fprintf(w, "- Groups with at least 1 preferred art workshop: %d / %d\n", summary.GroupsWithPreferredArt, summary.TotalGroups)
	_, _ = fmt.Fprintf(w, "- Groups with at least 1 preferred science workshop: %d / %d\n", summary.GroupsWithPreferredScience, summary.TotalGroups)
	_, _ = fmt.Fprintln(w)
}

func PrintGroups(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	PrintScheduleSummary(w, groups, state)

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		printGroup(w, group, state)
	}
}

func printGroup(w io.Writer, group *groupPkg.Group, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "Teacher = %s  \n", group.Teacher)
	if group.Grade == 0 {
		_, _ = fmt.Fprintf(w, "Grade = K  \n")
	} else {
		_, _ = fmt.Fprintf(w, "Grade = %d  \n", group.Grade)
	}
	_, _ = fmt.Fprintf(w, "ID = %s  \n", group.ID)
	_, _ = fmt.Fprintf(w, "Satisfaction = %d%%  \n", SatisfactionPercent(group, state))
	_, _ = fmt.Fprintf(w, "Students =  %v  \n", strings.Join(group.Students, ","))
	if len(group.ParentIDs) > 0 {
		parentIDs := make([]string, 0, len(group.ParentIDs))
		for parentID := range group.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		sort.Strings(parentIDs)
		_, _ = fmt.Fprintf(w, "Group contains child of presenter or assistant of workshop = %v  \n", strings.Join(parentIDs, ","))
	}
	if issues := SortedParentBookingIssues(group); len(issues) > 0 {
		_, _ = fmt.Fprintf(w, "Unbooked parent workshops = %s  \n", strings.Join(issues, ", "))
	}
	_, _ = fmt.Fprintln(w, "Schedule")
	_, _ = fmt.Fprintln(w, "| ID | Class | Room | Match |")
	_, _ = fmt.Fprintln(w, "| -- | ----- | ---- | ----- |")
	for session := 0; session < model.NumSessions; session++ {
		workshop := GetWorkshop(group, state, session)
		if workshop != nil {
			_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s |\n", workshop.ID, workshop.Name, workshop.Room, SessionSatisfactionLabel(group, state, session))
		} else {
			_, _ = fmt.Fprintf(w, "| - | - | - | Not Scheduled |\n")
			log.Printf("====UNFILLED SLOT====\n")
		}
	}
	_, _ = fmt.Fprintf(w, "\n---\n\n")
}

func PrintGroupsHTML(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	summary := CalculateScheduleSummary(groups, state)

	_, _ = fmt.Fprintf(w, `<style>
.schedule-summary {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.96), rgba(237,248,255,0.94));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.08);
}
.schedule-summary h3 {
    margin: 0 0 0.8em;
    color: #141f26;
    font-family: "Montserrat", "Avenir Next", "Futura", "Trebuchet MS", sans-serif;
    font-size: 1.15rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}
.schedule-summary-grid {
    display: grid;
    gap: 0.85em;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}
.schedule-summary-card {
    padding: 0.95em 1em;
    border-radius: 18px;
    background: rgba(233, 244, 253, 0.9);
    border: 1px solid rgba(73, 135, 87, 0.12);
}
.schedule-summary-card .label {
    display: block;
    margin-bottom: 0.3em;
    color: #52717c;
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
}
.schedule-summary-card .value {
    color: #19364a;
    font-size: 1.45rem;
    font-weight: 700;
}
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
    color: #1d4667;
}
.group .schedule .recess {
    background-color: #eef7ec;
    text-align: center;
    font-style: italic;
    color: #4a6c55;
}
.group .schedule .unfilled {
    color: #8a5c5c;
    font-style: italic;
}
</style>`)

	_, _ = fmt.Fprintf(w, "<div class='schedule-summary'>\n")
	_, _ = fmt.Fprintf(w, "<h3>Schedule Summary</h3>\n")
	_, _ = fmt.Fprintf(w, "<div class='schedule-summary-grid'>\n")
	writeSummaryCard(w, "Overall Satisfaction Points", fmt.Sprintf("%d", summary.OverallSatisfactionPoints))
	writeSummaryCard(w, "Average Satisfaction", fmt.Sprintf("%d%%", summary.AverageSatisfactionPercent))
	writeSummaryCard(w, "Groups With Preferred Art", fmt.Sprintf("%d / %d", summary.GroupsWithPreferredArt, summary.TotalGroups))
	writeSummaryCard(w, "Groups With Preferred Science", fmt.Sprintf("%d / %d", summary.GroupsWithPreferredScience, summary.TotalGroups))
	_, _ = fmt.Fprintf(w, "</div>\n</div>\n")

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		printGroupHTML(w, group, state)
	}
}

func printGroupHTML(w io.Writer, group *groupPkg.Group, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "<div class='group'>\n")
	_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n", group.ID)
	_, _ = fmt.Fprintf(w, "<div class='group-details'>\n")
	_, _ = fmt.Fprintf(w, "<p><strong>Teacher:</strong> %s</p>\n", group.Teacher)
	if group.Grade == 0 {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> K</p>\n")
	} else {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> %d</p>\n", group.Grade)
	}
	_, _ = fmt.Fprintf(w, "<p><strong>Students:</strong> %s</p>\n", strings.Join(group.Students, ", "))
	_, _ = fmt.Fprintf(w, "<p><strong>Satisfaction:</strong> %d%%</p>\n", SatisfactionPercent(group, state))

	if len(group.ParentIDs) > 0 {
		parentIDs := make([]string, 0, len(group.ParentIDs))
		for parentID := range group.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		sort.Strings(parentIDs)
		_, _ = fmt.Fprintf(w, "<p><strong>Parent Workshops:</strong> %s</p>\n", strings.Join(parentIDs, ", "))
	}
	if issues := SortedParentBookingIssues(group); len(issues) > 0 {
		_, _ = fmt.Fprintf(w, "<p><strong>Unbooked Parent Workshops:</strong> %s</p>\n", strings.Join(issues, ", "))
	}
	_, _ = fmt.Fprintf(w, "</div>\n")

	_, _ = fmt.Fprintf(w, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(w, "<thead>\n<tr>\n<th>Time</th>\n<th>Workshop ID</th>\n<th>Workshop Name</th>\n<th>Room</th>\n<th>Match</th>\n</tr>\n</thead>\n")
	_, _ = fmt.Fprintf(w, "<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(model.SessionTimes); i++ {
		_, _ = fmt.Fprintf(w, "<tr>\n")
		if i == 2 {
			_, _ = fmt.Fprintf(w, "<td colspan='5' class='recess'>%s</td>\n", model.SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(w, "<td>%s</td>\n", model.SessionTimes[i])
			if workshopIndex < model.NumSessions && GetWorkshop(group, state, workshopIndex) != nil {
				workshop := GetWorkshop(group, state, workshopIndex)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.ID)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.Name)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.Room)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", SessionSatisfactionLabel(group, state, workshopIndex))
			} else {
				_, _ = fmt.Fprintf(w, "<td colspan='4' class='unfilled'>Not Scheduled</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(w, "</tr>\n")
	}

	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n</div>\n")
}

func PrintWorkshops(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	var sortedIDs []string
	for id := range workshops {
		sortedIDs = append(sortedIDs, id)
	}
	slices.Sort(sortedIDs)
	for _, id := range sortedIDs {
		printWorkshop(w, workshops[id], state)
	}
}

func printWorkshop(w io.Writer, workshop *workshopPkg.Workshop, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "ID: %s  \n", workshop.ID)
	_, _ = fmt.Fprintf(w, "Name: %s  \n", workshop.Name)
	_, _ = fmt.Fprintf(w, "Capacity per session: %d  \n", workshop.Capacity)
	_, _ = fmt.Fprintf(w, "Overall utilization: %d%%  \n", state.OverallUtilization(workshop))
	_, _ = fmt.Fprintf(w, "Schedule  \n")
	_, _ = fmt.Fprintln(w, "| Utilization | Students |")
	_, _ = fmt.Fprintln(w, "| --------- | -------- |")
	for i := 0; i < model.NumSessions; i++ {
		if !workshop.IsSessionOffered(i) {
			_, _ = fmt.Fprintln(w, "| - | - |")
			continue
		}

		_, _ = fmt.Fprintf(w, "| %d%% | ", state.Utilization(workshop, i))
		groups := state.GroupsForWorkshopSession(workshop, i)
		for _, group := range groups {
			_, _ = fmt.Fprintf(w, "%v,", strings.Join(group.Students, ","))
		}
		_, _ = fmt.Fprintf(w, " |\n")
	}
	_, _ = fmt.Fprintf(w, "\n---\n\n")
}

func PrintWorkshopsHTML(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	writeWorkshopsHTMLStyle(w)

	var sortedIDs []string
	for id := range workshops {
		sortedIDs = append(sortedIDs, id)
	}
	slices.Sort(sortedIDs)

	for _, id := range sortedIDs {
		printWorkshopHTML(w, workshops[id], state)
	}
}

func printWorkshopHTML(w io.Writer, workshop *workshopPkg.Workshop, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "<div class='workshop'>\n")
	_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n", workshop.Name)
	_, _ = fmt.Fprintf(w, "<p><strong>ID:</strong> %s</p>\n", workshop.ID)
	_, _ = fmt.Fprintf(w, "<p><strong>Capacity per session:</strong> %d</p>\n", workshop.Capacity)
	_, _ = fmt.Fprintf(w, "<p><strong>Overall utilization:</strong> %d%%</p>\n", state.OverallUtilization(workshop))
	_, _ = fmt.Fprintf(w, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(w, "<thead>\n<tr>\n<th>Time</th>\n<th>Utilization</th>\n<th>Students</th>\n</tr>\n</thead>\n<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(model.SessionTimes); i++ {
		_, _ = fmt.Fprintf(w, "<tr>\n")
		if i == 2 {
			_, _ = fmt.Fprintf(w, "<td colspan='3' class='recess'>%s</td>\n", model.SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(w, "<td>%s</td>\n", model.SessionTimes[i])
			if workshopIndex < model.NumSessions && workshop.IsSessionOffered(workshopIndex) {
				utilization := state.Utilization(workshop, workshopIndex)
				utilizationClass := "normal"
				if utilization < 30 {
					utilizationClass = "low"
				} else if utilization > 80 {
					utilizationClass = "high"
				}

				_, _ = fmt.Fprintf(w, "<td class='utilization %s'>%d%%</td>\n", utilizationClass, utilization)
				_, _ = fmt.Fprintf(w, "<td class='students'>")
				groups := state.GroupsForWorkshopSession(workshop, workshopIndex)
				var studentNames []string
				for _, group := range groups {
					studentNames = append(studentNames, group.Students...)
				}
				_, _ = fmt.Fprintf(w, "%s", strings.Join(studentNames, ", "))
				_, _ = fmt.Fprintf(w, "</td>\n")
			} else {
				_, _ = fmt.Fprintf(w, "<td>-</td>\n<td>-</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(w, "</tr>\n")
	}

	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n</div>\n")
}

func writeWorkshopsHTMLStyle(w io.Writer) {
	_, _ = fmt.Fprintf(w, `<style>
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

func writeSummaryCard(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "<div class='schedule-summary-card'><span class='label'>%s</span><span class='value'>%s</span></div>\n", label, value)
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

func idToKind(id string) int {
	if id != "" && id[0] == 'A' {
		return model.ArtWorkshop
	}
	return model.SciWorkshop
}

func percentOf(part, total int) int {
	if total == 0 {
		return 0
	}
	return part * 100 / total
}
