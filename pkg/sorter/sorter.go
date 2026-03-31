package sorter

import (
	"fmt"
	//"log"
	"math/rand"
)

func BookWorkshopIfAvailable(workshop *Workshop, group *Group) bool {
	if !workshop.WithinGradeRange(group.Grade) {
		//log.Printf("Mismatched grade id=%s teacher=%s group=%s\n", workshop.id, group.Teacher, group.Name)
		return false
	}
	if group.IsEnrolledInWorkshop(workshop.id) {
		//log.Printf("Duplicate workshop id=%s teacher=%s group=%s\n", workshop.id, group.Teacher, group.Name)
		return false
	}
	sessions := workshop.GetAvailableSessions(group)
	if len(sessions) > 0 {
		randSession := sessions[rand.Intn(len(sessions))]
		workshop.TakeSession(randSession, group)
		group.BookWorkshop(randSession, workshop)

		return true
	}

	//log.Printf("Unable to book session, its full. workshop id=%s teacher=%s group=%s\n", workshop.id, group.Teacher, group.Name)

	return false
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
