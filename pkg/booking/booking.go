package booking

import (
	"cmp"
	"fmt"
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
	bookings                  []*Booking
	bookingsByGroup           map[string][]*Booking
	bookingsByGroupSession    map[groupSessionKey]*Booking
	bookingsByWorkshopSession map[workshopSessionKey][]*Booking
	randomSelection           bool
}

type groupSessionKey struct {
	groupID string
	session int
}

type workshopSessionKey struct {
	workshopID string
	session    int
}

type ScheduleSummary struct {
	OverallSatisfactionPoints  int
	AverageSatisfactionPercent int
	GroupsWithPreferredArt     int
	GroupsWithPreferredScience int
	TotalGroups                int
}

func NewScheduleState(_ []*groupPkg.Group, _ ...map[string]*workshopPkg.Workshop) *ScheduleState {
	return &ScheduleState{
		bookingsByGroup:           make(map[string][]*Booking),
		bookingsByGroupSession:    make(map[groupSessionKey]*Booking),
		bookingsByWorkshopSession: make(map[workshopSessionKey][]*Booking),
	}
}

func (s *ScheduleState) SetRandomSelection(enabled bool) {
	s.randomSelection = enabled
}

func (s *ScheduleState) Book(group *groupPkg.Group, workshop *workshopPkg.Workshop, session int) {
	booking := &Booking{
		Group:    group,
		Workshop: workshop,
		Session:  session,
	}
	s.bookings = append(s.bookings, booking)
	s.bookingsByGroup[group.ID] = append(s.bookingsByGroup[group.ID], booking)
	s.bookingsByGroupSession[groupSessionKey{groupID: group.ID, session: session}] = booking
	workshopKey := workshopSessionKey{workshopID: workshop.ID, session: session}
	s.bookingsByWorkshopSession[workshopKey] = append(s.bookingsByWorkshopSession[workshopKey], booking)
	delete(group.ParentBookingIssues, workshop.ID)
}

func (s *ScheduleState) Unbook(group *groupPkg.Group, workshop *workshopPkg.Workshop, session int) {
	for i := range s.bookings {
		booking := s.bookings[i]
		if booking.Group.ID == group.ID && booking.Workshop.ID == workshop.ID && booking.Session == session {
			s.bookings = slices.Delete(s.bookings, i, i+1)
			groupBookings := s.bookingsByGroup[group.ID]
			for j := range groupBookings {
				if groupBookings[j] == booking {
					groupBookings = slices.Delete(groupBookings, j, j+1)
					break
				}
			}
			if len(groupBookings) == 0 {
				delete(s.bookingsByGroup, group.ID)
			} else {
				s.bookingsByGroup[group.ID] = groupBookings
			}
			delete(s.bookingsByGroupSession, groupSessionKey{groupID: group.ID, session: session})
			workshopKey := workshopSessionKey{workshopID: workshop.ID, session: session}
			workshopBookings := s.bookingsByWorkshopSession[workshopKey]
			for j := range workshopBookings {
				if workshopBookings[j] == booking {
					workshopBookings = slices.Delete(workshopBookings, j, j+1)
					break
				}
			}
			if len(workshopBookings) == 0 {
				delete(s.bookingsByWorkshopSession, workshopKey)
			} else {
				s.bookingsByWorkshopSession[workshopKey] = workshopBookings
			}
			return
		}
	}
}

func (s *ScheduleState) WorkshopForGroupSession(group *groupPkg.Group, session int) *workshopPkg.Workshop {
	booking := s.bookingsByGroupSession[groupSessionKey{groupID: group.ID, session: session}]
	if booking == nil {
		return nil
	}
	return booking.Workshop
}

func (s *ScheduleState) GroupsForWorkshopSession(workshop *workshopPkg.Workshop, session int) []*groupPkg.Group {
	bookings := s.bookingsByWorkshopSession[workshopSessionKey{workshopID: workshop.ID, session: session}]
	groups := make([]*groupPkg.Group, 0, len(bookings))
	for _, booking := range bookings {
		groups = append(groups, booking.Group)
	}
	return groups
}

func (s *ScheduleState) IsEnrolledInWorkshop(group *groupPkg.Group, id string) bool {
	for _, booking := range s.bookingsByGroup[group.ID] {
		if booking.Workshop.ID == id {
			return true
		}
	}

	return false
}

func (s *ScheduleState) SessionsBooked(group *groupPkg.Group, kind int) int {
	booked := 0
	for _, booking := range s.bookingsByGroup[group.ID] {
		if booking.Workshop.Kind == kind {
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
	for _, booking := range s.bookingsByWorkshopSession[workshopSessionKey{workshopID: workshop.ID, session: session}] {
		spotsTaken += booking.Group.NumStudents()
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

func BookWorkshopIfAvailable(state *ScheduleState, workshop *workshopPkg.Workshop, group *groupPkg.Group) (bool, string) {
	if reason := BookingFailureReason(state, workshop, group); reason != "" {
		return false, reason
	}

	sessions := state.AvailableSessions(workshop, group)
	sessionIndex := 0
	if state.randomSelection {
		sessionIndex = rand.Intn(len(sessions))
	}
	state.Book(group, workshop, sessions[sessionIndex])

	return true, ""
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
	workshopIDs := make([]string, 0, len(workshops))
	for id := range workshops {
		workshopIDs = append(workshopIDs, id)
	}
	sort.Strings(workshopIDs)

	for _, id := range workshopIDs {
		workshop := workshops[id]
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
