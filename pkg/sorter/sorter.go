package sorter

import (
	"fmt"
	//"log"
	"math/rand"
)

func BookWorkshopIfAvailable(workshop *Workshop, group *Group) bool {
	if reason := BookingFailureReason(workshop, group); reason != "" {
		return false
	}

	sessions := workshop.GetAvailableSessions(group)
	randSession := sessions[rand.Intn(len(sessions))]
	workshop.TakeSession(randSession, group)
	group.BookWorkshop(randSession, workshop)

	return true
}

func BookingFailureReason(workshop *Workshop, group *Group) string {
	if !workshop.WithinGradeRange(group.Grade) {
		return "grade mismatch"
	}
	if group.IsEnrolledInWorkshop(workshop.id) {
		return "duplicate workshop"
	}
	if len(workshop.GetAvailableSessions(group)) == 0 {
		return "no available session with enough capacity"
	}

	return ""
}

func GetWorkshopFromID(id string, artWorkshops, sciWorkshops map[string]*Workshop) (*Workshop, error) {
	var workshop *Workshop
	var ok bool
	kind := idToKind(id)
	if kind == ArtWorkshop {
		workshop, ok = artWorkshops[id]
		if !ok {
			return nil, fmt.Errorf("art workshop ID %s not found", id)
		}
	} else {
		workshop, ok = sciWorkshops[id]
		if !ok {
			return nil, fmt.Errorf("science workshop ID %s not found", id)
		}
	}

	return workshop, nil
}
