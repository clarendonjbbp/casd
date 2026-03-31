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
	"github.com/clarendonjbbp/casd/pkg/sorter"
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
		return nil, nil, nil, fmt.Errorf("couldn't read groups: %v", err)
	}

	artWorkshops, err := workshopPkg.ReadWorkshops(artWorkshopsFile, model.ArtWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read art workshop: %v", err)
	}

	sciWorkshops, err := workshopPkg.ReadWorkshops(sciWorkshopsFile, model.SciWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("couldn't read science workshop: %v", err)
	}

	return groups, artWorkshops, sciWorkshops, nil
}

func Schedule(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, opts ScheduleOptions) error {
	booking.NewScheduleState(groups, artWorkshops, sciWorkshops)

	log.Printf("====Booking Parent Classes===\n")
	if err := bookParentClasses(groups, artWorkshops, sciWorkshops, opts.ContinueOnParentLookupErr); err != nil {
		return err
	}

	if opts.Random {
		shuffleGroups(groups)
	}

	log.Printf("\n====Guaranteeing Preferred Art Classes===\n")
	bookGuaranteedPreferredWorkshop(groups, artWorkshops, model.ArtWorkshop)

	log.Printf("\n====Guaranteeing Preferred Science Classes===\n")
	bookGuaranteedPreferredWorkshop(groups, sciWorkshops, model.SciWorkshop)

	log.Printf("\n====Booking Art Classes===\n")
	bookWorkshopKind(groups, artWorkshops, sciWorkshops, model.ArtWorkshop)

	log.Printf("\n====Booking Science Classes===\n")
	bookWorkshopKind(groups, sciWorkshops, artWorkshops, model.SciWorkshop)

	log.Printf("\n====Rebalancing Workshops===\n")
	if err := RebalanceWorkshops(opts.MinUtilization, artWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance art workshops: %v", err)
	}
	if err := RebalanceWorkshops(opts.MinUtilization, sciWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance science workshops: %v", err)
	}

	return nil
}

func bookParentClasses(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, continueOnLookupErr bool) error {
	for _, group := range groups {
		for parentID := range group.ParentIDs {
			workshop, err := getWorkshopFromID(parentID, artWorkshops, sciWorkshops)
			if err != nil {
				group.ParentBookingIssues[parentID] = "workshop not found"
				if continueOnLookupErr {
					log.Printf("Error finding parent class for teacher=%s group=%s: %v", group.Teacher, group.Name, err)
					continue
				}
				return fmt.Errorf("error finding parent class for teacher=%s group=%s: %w", group.Teacher, group.Name, err)
			}

			if booked := sorter.BookWorkshopIfAvailable((*sorter.Workshop)(workshop), (*sorter.Group)(group)); !booked {
				group.ParentBookingIssues[parentID] = sorter.BookingFailureReason((*sorter.Workshop)(workshop), (*sorter.Group)(group))
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s reason=%s", parentID, group.Teacher, group.Name, group.ParentBookingIssues[parentID])
			}
		}
	}

	return nil
}

func bookGuaranteedPreferredWorkshop(groups []*groupPkg.Group, workshops map[string]*workshopPkg.Workshop, kind int) {
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

			if sorter.BookWorkshopIfAvailable((*sorter.Workshop)(workshop), (*sorter.Group)(group)) {
				break
			}
		}
	}
}

func bookWorkshopKind(groups []*groupPkg.Group, primaryWorkshops, fallbackWorkshops map[string]*workshopPkg.Workshop, kind int) {
	needsRandomBooking := bookPreferredWorkshops(groups, primaryWorkshops, kind)
	bookRandomWorkshops(needsRandomBooking, primaryWorkshops, fallbackWorkshops, kind)
}

func bookPreferredWorkshops(groups []*groupPkg.Group, workshops map[string]*workshopPkg.Workshop, kind int) []*groupPkg.Group {
	var needsRandomBooking []*groupPkg.Group

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

			if booked := sorter.BookWorkshopIfAvailable((*sorter.Workshop)(workshop), (*sorter.Group)(group)); booked {
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

func bookRandomWorkshops(groupsNeedingBooking []*groupPkg.Group, primaryWorkshops, fallbackWorkshops map[string]*workshopPkg.Workshop, kind int) {
	log.Printf("\n====Booking Random %s Classes===\n", kindLabel(kind))

	sortedPrimary := workshopPkg.SortWorkshopsByOverallUtilization(primaryWorkshops)
	sortedFallback := workshopPkg.SortWorkshopsByOverallUtilization(fallbackWorkshops)

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

func bookFromSortedWorkshops(group *groupPkg.Group, workshops []*workshopPkg.Workshop) bool {
	for _, workshop := range workshops {
		if sorter.BookWorkshopIfAvailable((*sorter.Workshop)(workshop), (*sorter.Group)(group)) {
			return true
		}
	}

	return false
}

func RebalanceWorkshops(minUtilization int, workshops map[string]*workshopPkg.Workshop, groups []*groupPkg.Group) error {
	for maxPreference := 1; maxPreference < 6; maxPreference++ {
		underutilizedWorkshops, underutilizedWorkshopSessions := workshopPkg.GetUnderutilizedSessions(minUtilization, workshops)
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
				if oldWorkshop.UtilizationWithoutGroup(session, (*sorter.Group)(group)) < minUtilization {
					continue
				}
				if wouldBreakPreferredGuarantee(group, oldWorkshop, workshop) {
					continue
				}

				preference := group.HowPreferredIsBookedWorkshop(session)
				if preference < maxPreference {
					log.Printf("Rebalancing with group teacher=%s name=%s", group.Teacher, group.Name)
					oldWorkshop.UnbookSession(session, (*sorter.Group)(group))
					group.BookWorkshop(session, (*sorter.Workshop)(workshop))
					break
				}
			}
		}
	}

	return errors.New("unable to rebalance workshop")
}

func wouldBreakPreferredGuarantee(group *groupPkg.Group, oldWorkshop, newWorkshop *workshopPkg.Workshop) bool {
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

func shuffleGroups(groups []*groupPkg.Group) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(groups) > 0 {
		n := len(groups)
		randIndex := r.Intn(n)
		groups[n-1], groups[randIndex] = groups[randIndex], groups[n-1]
		groups = groups[:n-1]
	}
}

func sessionsRequired(kind int) int {
	if kind == model.ArtWorkshop {
		return model.NumArtSessions
	}
	return model.NumSciSessions
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
