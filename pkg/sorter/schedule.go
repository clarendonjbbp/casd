package sorter

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"
)

const (
	numArtSessions = 2
	numSciSessions = 2
)

type ScheduleOptions struct {
	Random                    bool
	MinUtilization            int
	ContinueOnParentLookupErr bool
}

func ReadCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile string) ([]*Group, map[string]*Workshop, map[string]*Workshop, error) {
	groups, err := ReadGroups(groupsFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read groups: %v", err)
	}

	artWorkshops, err := ReadWorkshops(artWorkshopsFile, ArtWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read art workshop: %v", err)
	}

	sciWorkshops, err := ReadWorkshops(sciWorkshopsFile, SciWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read science workshop: %v", err)
	}

	return groups, artWorkshops, sciWorkshops, nil
}

func Schedule(groups []*Group, artWorkshops, sciWorkshops map[string]*Workshop, opts ScheduleOptions) error {
	NewScheduleState(groups, artWorkshops, sciWorkshops)

	log.Printf("====Booking Parent Classes===\n")
	if err := bookParentClasses(groups, artWorkshops, sciWorkshops, opts.ContinueOnParentLookupErr); err != nil {
		return err
	}

	if opts.Random {
		shuffleGroups(groups)
	}

	log.Printf("\n====Guaranteeing Preferred Art Classes===\n")
	bookGuaranteedPreferredWorkshop(groups, artWorkshops, ArtWorkshop)

	log.Printf("\n====Guaranteeing Preferred Science Classes===\n")
	bookGuaranteedPreferredWorkshop(groups, sciWorkshops, SciWorkshop)

	log.Printf("\n====Booking Art Classes===\n")
	bookWorkshopKind(groups, artWorkshops, sciWorkshops, ArtWorkshop)

	log.Printf("\n====Booking Science Classes===\n")
	bookWorkshopKind(groups, sciWorkshops, artWorkshops, SciWorkshop)

	log.Printf("\n====Rebalancing Workshops===\n")
	if err := RebalanceWorkshops(opts.MinUtilization, artWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance art workshops: %v", err)
	}
	if err := RebalanceWorkshops(opts.MinUtilization, sciWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance science workshops: %v", err)
	}

	return nil
}

func bookGuaranteedPreferredWorkshop(groups []*Group, workshops map[string]*Workshop, kind int) {
	for _, group := range groups {
		if group.SessionsBooked(kind) >= 1 {
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

			if BookWorkshopIfAvailable(workshop, group) {
				break
			}
		}
	}
}

func bookParentClasses(groups []*Group, artWorkshops, sciWorkshops map[string]*Workshop, continueOnLookupErr bool) error {
	for _, group := range groups {
		for parentID := range group.ParentIDs {
			workshop, err := GetWorkshopFromID(parentID, artWorkshops, sciWorkshops)
			if err != nil {
				group.ParentBookingIssues[parentID] = "workshop not found"
				if continueOnLookupErr {
					log.Printf("Error finding parent class for teacher=%s group=%s: %v", group.Teacher, group.Name, err)
					continue
				}
				return fmt.Errorf("error finding parent class for teacher=%s group=%s: %w", group.Teacher, group.Name, err)
			}

			if booked := BookWorkshopIfAvailable(workshop, group); !booked {
				group.ParentBookingIssues[parentID] = BookingFailureReason(workshop, group)
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s reason=%s", parentID, group.Teacher, group.Name, group.ParentBookingIssues[parentID])
			}
		}
	}

	return nil
}

func bookWorkshopKind(groups []*Group, primaryWorkshops, fallbackWorkshops map[string]*Workshop, kind int) {
	needsRandomBooking := bookPreferredWorkshops(groups, primaryWorkshops, kind)
	bookRandomWorkshops(needsRandomBooking, primaryWorkshops, fallbackWorkshops, kind)
}

func bookPreferredWorkshops(groups []*Group, workshops map[string]*Workshop, kind int) []*Group {
	var needsRandomBooking []*Group

	for _, group := range groups {
		sessionsToBook := sessionsRequired(kind) - group.SessionsBooked(kind)
		if sessionsToBook < 1 {
			continue
		}

		for _, id := range preferencesForKind(group, kind) {
			workshop, ok := workshops[id]
			if !ok {
				log.Printf("%s workshop ID %s not found for teacher=%s group=%s", kindLabel(kind), id, group.Teacher, group.Name)
				continue
			}

			if booked := BookWorkshopIfAvailable(workshop, group); booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		for i := 0; i < sessionsToBook; i++ {
			needsRandomBooking = append(needsRandomBooking, group)
		}
	}

	return needsRandomBooking
}

func bookRandomWorkshops(groupsNeedingBooking []*Group, primaryWorkshops, fallbackWorkshops map[string]*Workshop, kind int) {
	log.Printf("\n====Booking Random %s Classes===\n", kindLabel(kind))

	sortedPrimary := SortWorkshopsByOverallUtilization(primaryWorkshops)
	sortedFallback := SortWorkshopsByOverallUtilization(fallbackWorkshops)

	for _, group := range groupsNeedingBooking {
		if bookFromSortedWorkshops(group, sortedPrimary) {
			continue
		}
		if bookFromSortedWorkshops(group, sortedFallback) {
			continue
		}

		log.Printf("Could not find available %s workshop for %s %s", kindLower(kind), group.Teacher, group.Name)
	}
}

func bookFromSortedWorkshops(group *Group, workshops []*Workshop) bool {
	for _, workshop := range workshops {
		if BookWorkshopIfAvailable(workshop, group) {
			return true
		}
	}

	return false
}

func RebalanceWorkshops(minUtilization int, workshops map[string]*Workshop, groups []*Group) error {
	for maxPreference := 1; maxPreference < 6; maxPreference++ {
		underutilizedWorkshops, underutilizedWorkshopSessions := GetUnderutilizedSessions(minUtilization, workshops)
		if len(underutilizedWorkshops) == 0 {
			return nil
		}

		for i := range underutilizedWorkshops {
			workshop := underutilizedWorkshops[i]
			session := underutilizedWorkshopSessions[i]
			log.Printf("Rebalancing %s at %d%% utilization for session %d", workshop.Name, workshop.Utilization(session), session)

			for _, group := range groups {
				if !workshop.WithinGradeRange(group.Grade) {
					continue
				}
				if group.IsEnrolledInWorkshop(workshop.GetID()) {
					continue
				}
				if workshop.SpotsAvailable(session) < group.NumStudents() {
					continue
				}

				oldWorkshop := group.GetWorkshop(session)
				if oldWorkshop == nil || oldWorkshop.Kind != workshop.Kind {
					continue
				}
				if oldWorkshop.UtilizationWithoutGroup(session, group) < minUtilization {
					continue
				}
				if wouldBreakPreferredGuarantee(group, oldWorkshop, workshop) {
					continue
				}

				preference := group.HowPreferredIsBookedWorkshop(session)
				if preference < maxPreference {
					log.Printf("Rebalancing with group teacher=%s name=%s", group.Teacher, group.Name)
					oldWorkshop.UnbookSession(session, group)
					group.BookWorkshop(session, workshop)
					break
				}
			}
		}
	}

	return errors.New("unable to rebalance workshop")
}

func wouldBreakPreferredGuarantee(group *Group, oldWorkshop, newWorkshop *Workshop) bool {
	kind := oldWorkshop.Kind
	currentPreferredCount := group.CountPreferredWorkshopsOfKind(kind)
	if currentPreferredCount > 1 {
		return false
	}

	oldRank := group.PreferenceRankForWorkshopID(oldWorkshop.GetID())
	if oldRank < 1 || oldRank > 4 {
		return false
	}

	newRank := group.PreferenceRankForWorkshopID(newWorkshop.GetID())
	return newRank < 1 || newRank > 4
}

func shuffleGroups(groups []*Group) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(groups) > 0 {
		n := len(groups)
		randIndex := r.Intn(n)
		groups[n-1], groups[randIndex] = groups[randIndex], groups[n-1]
		groups = groups[:n-1]
	}
}

func sessionsRequired(kind int) int {
	if kind == ArtWorkshop {
		return numArtSessions
	}
	return numSciSessions
}

func preferencesForKind(group *Group, kind int) []string {
	if kind == ArtWorkshop {
		return group.ArtIDs
	}
	return group.SciIDs
}

func kindLabel(kind int) string {
	if kind == ArtWorkshop {
		return "Art"
	}
	return "Science"
}

func kindLower(kind int) string {
	if kind == ArtWorkshop {
		return "art"
	}
	return "science"
}
