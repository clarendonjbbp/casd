package scheduler

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/clarendonjbbp/casd/pkg/booking"
	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

type ScheduleOptions struct {
	Random                    bool
	MinUtilization            int
	ContinueOnParentLookupErr bool
}

func ReadCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile string) ([]*groupPkg.Group, map[string]*workshopPkg.Workshop, map[string]*workshopPkg.Workshop, error) {
	groups, err := groupPkg.ReadGroups(groupsFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read groups: %w", err)
	}

	artWorkshops, err := workshopPkg.ReadWorkshops(artWorkshopsFile, model.ArtWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read art workshop: %w", err)
	}

	sciWorkshops, err := workshopPkg.ReadWorkshops(sciWorkshopsFile, model.SciWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read science workshop: %w", err)
	}

	return groups, artWorkshops, sciWorkshops, nil
}

func Schedule(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, opts ScheduleOptions) (*booking.ScheduleState, error) {
	state := booking.NewScheduleState(groups, artWorkshops, sciWorkshops)
	state.SetRandomSelection(opts.Random)

	log.Printf("====Booking Parent Classes===\n")
	if err := bookParentClasses(state, groups, artWorkshops, sciWorkshops, opts.ContinueOnParentLookupErr); err != nil {
		return nil, err
	}

	if opts.Random {
		shuffleGroups(groups)
	}

	log.Printf("\n====Guaranteeing Preferred Art Classes===\n")
	bookGuaranteedPreferredWorkshop(state, groups, artWorkshops, model.ArtWorkshop)

	log.Printf("\n====Guaranteeing Preferred Science Classes===\n")
	bookGuaranteedPreferredWorkshop(state, groups, sciWorkshops, model.SciWorkshop)

	log.Printf("\n====Booking Art Classes===\n")
	bookWorkshopKind(state, groups, artWorkshops, sciWorkshops, model.ArtWorkshop)

	log.Printf("\n====Booking Science Classes===\n")
	bookWorkshopKind(state, groups, sciWorkshops, artWorkshops, model.SciWorkshop)

	log.Printf("\n====Rebalancing Workshops===\n")
	if err := RebalanceWorkshops(state, opts.MinUtilization, artWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance art workshops: %v", err)
	}
	if err := RebalanceWorkshops(state, opts.MinUtilization, sciWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance science workshops: %v", err)
	}

	return state, nil
}

func bookParentClasses(state *booking.ScheduleState, groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, continueOnLookupErr bool) error {
	for _, group := range groups {
		for _, parentID := range group.SortedParentIDs() {
			workshop, err := getWorkshopFromID(parentID, artWorkshops, sciWorkshops)
			if err != nil {
				group.ParentBookingIssues[parentID] = "workshop not found"
				if continueOnLookupErr {
					log.Printf("Error finding parent class for teacher=%s group=%s: %v", group.Teacher, group.Name, err)
					continue
				}
				return fmt.Errorf("error finding parent class for teacher=%s group=%s: %w", group.Teacher, group.Name, err)
			}

			if booked, reason := booking.BookWorkshopIfAvailable(state, workshop, group); !booked {
				group.ParentBookingIssues[parentID] = reason
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s reason=%s", parentID, group.Teacher, group.Name, group.ParentBookingIssues[parentID])
			}
		}
	}

	return nil
}

func bookGuaranteedPreferredWorkshop(state *booking.ScheduleState, groups []*groupPkg.Group, workshops map[string]*workshopPkg.Workshop, kind int) {
	for _, group := range groups {
		if state.SessionsBooked(group, kind) >= 1 {
			continue
		}

		for _, id := range preferencesForKind(group, kind) {
			workshop, ok := workshops[id]
			if !ok {
				if id != "" {
					log.Printf("%s workshop ID %s not found for teacher=%s group=%s during guarantee phase", kindLabel(kind), id, group.Teacher, group.Name)
				}
				continue
			}

			if booked, _ := booking.BookWorkshopIfAvailable(state, workshop, group); booked {
				break
			}
		}
	}
}

func bookWorkshopKind(state *booking.ScheduleState, groups []*groupPkg.Group, primaryWorkshops, fallbackWorkshops map[string]*workshopPkg.Workshop, kind int) {
	needsRandomBooking := bookPreferredWorkshops(state, groups, primaryWorkshops, kind)
	bookRandomWorkshops(state, needsRandomBooking, primaryWorkshops, fallbackWorkshops, kind)
}

func bookPreferredWorkshops(state *booking.ScheduleState, groups []*groupPkg.Group, workshops map[string]*workshopPkg.Workshop, kind int) []*groupPkg.Group {
	var needsRandomBooking []*groupPkg.Group

	for _, group := range groups {
		sessionsToBook := model.NumSciSessions - state.SessionsBooked(group, kind)
		if kind == model.ArtWorkshop {
			sessionsToBook = model.NumArtSessions - state.SessionsBooked(group, kind)
		}
		if sessionsToBook < 1 {
			continue
		}

		for _, id := range preferencesForKind(group, kind) {
			workshop, ok := workshops[id]
			if !ok {
				log.Printf("%s workshop ID %s not found for teacher=%s group=%s", kindLabel(kind), id, group.Teacher, group.Name)
				continue
			}

			if booked, _ := booking.BookWorkshopIfAvailable(state, workshop, group); booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		for range sessionsToBook {
			needsRandomBooking = append(needsRandomBooking, group)
		}
	}

	return needsRandomBooking
}

func bookRandomWorkshops(state *booking.ScheduleState, groupsNeedingBooking []*groupPkg.Group, primaryWorkshops, fallbackWorkshops map[string]*workshopPkg.Workshop, kind int) {
	log.Printf("\n====Booking Random %s Classes===\n", kindLabel(kind))

	sortedPrimary := booking.SortWorkshopsByOverallUtilization(state, primaryWorkshops)
	sortedFallback := booking.SortWorkshopsByOverallUtilization(state, fallbackWorkshops)

	for _, group := range groupsNeedingBooking {
		if bookFromSortedWorkshops(state, group, sortedPrimary) {
			continue
		}
		if bookFromSortedWorkshops(state, group, sortedFallback) {
			continue
		}

		log.Printf("Could not find available %s workshop for %s %s", kindLower(kind), group.Teacher, group.Name)
	}
}

func bookFromSortedWorkshops(state *booking.ScheduleState, group *groupPkg.Group, workshops []*workshopPkg.Workshop) bool {
	for _, workshop := range workshops {
		if booked, _ := booking.BookWorkshopIfAvailable(state, workshop, group); booked {
			return true
		}
	}

	return false
}

func RebalanceWorkshops(state *booking.ScheduleState, minUtilization int, workshops map[string]*workshopPkg.Workshop, groups []*groupPkg.Group) error {
	for maxPreference := 1; maxPreference < 6; maxPreference++ {
		underutilizedWorkshops, underutilizedWorkshopSessions := booking.GetUnderutilizedSessions(state, minUtilization, workshops)
		if len(underutilizedWorkshops) == 0 {
			return nil
		}

		for i := range underutilizedWorkshops {
			workshop := underutilizedWorkshops[i]
			session := underutilizedWorkshopSessions[i]
			log.Printf("Rebalancing %s at %d%% utilization for session %d", workshop.Name, state.Utilization(workshop, session), session)

			for _, group := range groups {
				if !workshop.WithinGradeRange(group.Grade) {
					continue
				}
				if state.IsEnrolledInWorkshop(group, workshop.GetID()) {
					continue
				}
				if state.SpotsAvailable(workshop, session) < group.NumStudents() {
					continue
				}

				oldWorkshop := state.WorkshopForGroupSession(group, session)
				if oldWorkshop == nil || oldWorkshop.Kind != workshop.Kind {
					continue
				}
				if state.UtilizationWithoutGroup(oldWorkshop, session, group) < minUtilization {
					continue
				}
				if wouldBreakPreferredGuarantee(state, group, oldWorkshop, workshop) {
					continue
				}

				preference := booking.HowPreferredIsBookedWorkshop(group, state, session)
				if preference < maxPreference {
					log.Printf("Rebalancing with group teacher=%s name=%s", group.Teacher, group.Name)
					state.Unbook(group, oldWorkshop, session)
					state.Book(group, workshop, session)
					break
				}
			}
		}
	}

	return errors.New("unable to rebalance workshop")
}

func wouldBreakPreferredGuarantee(state *booking.ScheduleState, group *groupPkg.Group, oldWorkshop, newWorkshop *workshopPkg.Workshop) bool {
	kind := oldWorkshop.Kind
	currentPreferredCount := booking.CountPreferredWorkshopsOfKind(group, state, kind)
	if currentPreferredCount > 1 {
		return false
	}

	oldRank := booking.PreferenceRankForWorkshopID(group, oldWorkshop.GetID())
	if oldRank < 1 || oldRank > 4 {
		return false
	}

	newRank := booking.PreferenceRankForWorkshopID(group, newWorkshop.GetID())
	return newRank < 1 || newRank > 4
}

func shuffleGroups(groups []*groupPkg.Group) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(groups) > 0 {
		n := len(groups)
		randIndex := r.Intn(n)
		groups[n-1], groups[randIndex] = groups[randIndex], groups[n-1]
		groups = groups[:n-1]
	}
}

func preferencesForKind(group *groupPkg.Group, kind int) []string {
	if kind == model.ArtWorkshop {
		return group.ArtIDs
	}
	return group.SciIDs
}

func kindLabel(kind int) string {
	if kind == model.ArtWorkshop {
		return "Art"
	}
	return "Science"
}

func kindLower(kind int) string {
	if kind == model.ArtWorkshop {
		return "art"
	}
	return "science"
}

func getWorkshopFromID(id string, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop) (*workshopPkg.Workshop, error) {
	kind := model.SciWorkshop
	if id != "" && id[0] == 'A' {
		kind = model.ArtWorkshop
	}
	if kind == model.ArtWorkshop {
		workshop, ok := artWorkshops[id]
		if !ok {
			return nil, fmt.Errorf("art workshop ID %s not found", id)
		}
		return workshop, nil
	}

	workshop, ok := sciWorkshops[id]
	if !ok {
		return nil, fmt.Errorf("science workshop ID %s not found", id)
	}
	return workshop, nil
}
