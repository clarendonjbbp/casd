package sorter

import "slices"

type Booking struct {
	Group    *Group
	Workshop *Workshop
	Session  int
}

type ScheduleState struct {
	bookings []*Booking
}

func NewScheduleState(groups []*Group, workshopSets ...map[string]*Workshop) *ScheduleState {
	state := &ScheduleState{}
	for _, group := range groups {
		group.schedule = state
	}
	for _, workshops := range workshopSets {
		for _, workshop := range workshops {
			workshop.schedule = state
		}
	}

	return state
}

func (s *ScheduleState) Book(group *Group, workshop *Workshop, session int) {
	s.bookings = append(s.bookings, &Booking{
		Group:    group,
		Workshop: workshop,
		Session:  session,
	})
}

func (s *ScheduleState) Unbook(group *Group, workshop *Workshop, session int) {
	for i := range s.bookings {
		booking := s.bookings[i]
		if booking.Group.id == group.id && booking.Workshop.id == workshop.id && booking.Session == session {
			s.bookings = slices.Delete(s.bookings, i, i+1)
			return
		}
	}
}

func (s *ScheduleState) WorkshopForGroupSession(group *Group, session int) *Workshop {
	for _, booking := range s.bookings {
		if booking.Group.id == group.id && booking.Session == session {
			return booking.Workshop
		}
	}

	return nil
}

func (s *ScheduleState) GroupsForWorkshopSession(workshop *Workshop, session int) []*Group {
	var groups []*Group
	for _, booking := range s.bookings {
		if booking.Workshop.id == workshop.id && booking.Session == session {
			groups = append(groups, booking.Group)
		}
	}

	return groups
}

func (s *ScheduleState) IsEnrolledInWorkshop(group *Group, id string) bool {
	for _, booking := range s.bookings {
		if booking.Group.id == group.id && booking.Workshop.id == id {
			return true
		}
	}

	return false
}

func (s *ScheduleState) SessionsBooked(group *Group, kind int) int {
	booked := 0
	for _, booking := range s.bookings {
		if booking.Group.id == group.id && booking.Workshop.Kind == kind {
			booked++
		}
	}

	return booked
}

func (s *ScheduleState) SessionOccupied(group *Group, session int) bool {
	return s.WorkshopForGroupSession(group, session) != nil
}

func (s *ScheduleState) SpotsAvailable(workshop *Workshop, session int) int {
	if !workshop.IsSessionOffered(session) {
		return -1
	}

	spotsTaken := 0
	for _, booking := range s.bookings {
		if booking.Workshop.id == workshop.id && booking.Session == session {
			spotsTaken += booking.Group.NumStudents()
		}
	}

	return workshop.Capacity - spotsTaken
}

func (s *ScheduleState) Utilization(workshop *Workshop, session int) int {
	spotsAvailable := s.SpotsAvailable(workshop, session)
	if spotsAvailable == -1 {
		return -1
	}

	return percentOf(workshop.Capacity-spotsAvailable, workshop.Capacity)
}

func (s *ScheduleState) UtilizationWithoutGroup(workshop *Workshop, session int, group *Group) int {
	spotsAvailable := s.SpotsAvailable(workshop, session)
	if spotsAvailable == -1 {
		return -1
	}

	return percentOf(workshop.Capacity-(spotsAvailable+group.NumStudents()), workshop.Capacity)
}

func (s *ScheduleState) OverallUtilization(workshop *Workshop) int {
	overallSpotsTaken := 0
	overallCapacity := 0
	for session := 0; session < numSessions; session++ {
		if !workshop.IsSessionOffered(session) {
			continue
		}

		overallCapacity += workshop.Capacity
		overallSpotsTaken += workshop.Capacity - s.SpotsAvailable(workshop, session)
	}

	return percentOf(overallSpotsTaken, overallCapacity)
}

func (s *ScheduleState) AvailableSessions(workshop *Workshop, group *Group) []int {
	var availableSessions []int
	maxRemainingSlots := 0
	for session := 0; session < numSessions; session++ {
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
