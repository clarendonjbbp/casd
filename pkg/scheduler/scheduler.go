package scheduler

import (
	"errors"
	"fmt"
	"io"
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
	RandomRuns                int
	RandomSeed                int64
	RandomSeedSet             bool
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
	if opts.RandomRuns > 1 {
		return scheduleBestOfRandomRuns(groups, artWorkshops, sciWorkshops, opts)
	}

	rng, selectedSeed := randomSourceForOptions(opts, 0)
	return scheduleOnce(groups, artWorkshops, sciWorkshops, opts, opts.Random, rng, selectedSeed)
}

func scheduleOnce(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, opts ScheduleOptions, random bool, rng *rand.Rand, selectedSeed int64) (*booking.ScheduleState, error) {
	state := booking.NewScheduleState(groups, artWorkshops, sciWorkshops)
	state.SetRandomSelection(random)
	state.SetRandomSource(rng)
	state.RandomRuns = max(opts.RandomRuns, 1)
	state.SelectedRun = 1
	state.SelectedSeed = selectedSeed
	state.HasSelectedSeed = random

	log.Printf("====Booking Parent Classes===\n")
	if err := bookParentClasses(state, groups, artWorkshops, sciWorkshops, opts.ContinueOnParentLookupErr); err != nil {
		return nil, err
	}

	if random {
		shuffleGroups(groups, rng)
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

func scheduleBestOfRandomRuns(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, opts ScheduleOptions) (*booking.ScheduleState, error) {
	baseSeed := opts.RandomSeed
	if !opts.RandomSeedSet {
		baseSeed = time.Now().UnixNano()
	}

	var bestState *booking.ScheduleState
	var bestGroups []*groupPkg.Group
	var bestSummary booking.ScheduleSummary
	bestRun := 0
	bestSeed := int64(0)

	for run := 1; run <= opts.RandomRuns; run++ {
		runSeed := baseSeed + int64(run-1)
		candidateGroups := cloneGroups(groups)
		state, err := scheduleQuietly(func() (*booking.ScheduleState, error) {
			return scheduleOnce(candidateGroups, artWorkshops, sciWorkshops, opts, true, rand.New(rand.NewSource(runSeed)), runSeed)
		})
		if err != nil {
			return nil, err
		}

		summary := booking.CalculateScheduleSummary(candidateGroups, state)
		log.Printf("Optimization run %d of %d using seed %d: satisfaction=%d average=%d%% preferred_art=%d/%d preferred_science=%d/%d",
			run,
			opts.RandomRuns,
			runSeed,
			summary.OverallSatisfactionPoints,
			summary.AverageSatisfactionPercent,
			summary.GroupsWithPreferredArt,
			summary.TotalGroups,
			summary.GroupsWithPreferredScience,
			summary.TotalGroups,
		)
		if bestState == nil || isBetterSummary(summary, bestSummary) {
			bestState = state
			bestGroups = candidateGroups
			bestSummary = summary
			bestRun = run
			bestSeed = runSeed
		}
	}

	copyParentBookingIssues(groups, bestGroups)
	bestState.RandomRuns = opts.RandomRuns
	bestState.SelectedRun = bestRun
	bestState.SelectedSeed = bestSeed
	bestState.HasSelectedSeed = true
	log.Printf("Selected optimized run %d of %d using seed %d: satisfaction=%d average=%d%% preferred_art=%d/%d preferred_science=%d/%d",
		bestRun,
		opts.RandomRuns,
		bestSeed,
		bestSummary.OverallSatisfactionPoints,
		bestSummary.AverageSatisfactionPercent,
		bestSummary.GroupsWithPreferredArt,
		bestSummary.TotalGroups,
		bestSummary.GroupsWithPreferredScience,
		bestSummary.TotalGroups,
	)

	return bestState, nil
}

func scheduleQuietly(schedule func() (*booking.ScheduleState, error)) (*booking.ScheduleState, error) {
	originalWriter := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalWriter)

	return schedule()
}

func isBetterSummary(candidate, current booking.ScheduleSummary) bool {
	if candidate.OverallSatisfactionPoints != current.OverallSatisfactionPoints {
		return candidate.OverallSatisfactionPoints > current.OverallSatisfactionPoints
	}
	if candidate.AverageSatisfactionPercent != current.AverageSatisfactionPercent {
		return candidate.AverageSatisfactionPercent > current.AverageSatisfactionPercent
	}
	if candidate.GroupsWithPreferredArt != current.GroupsWithPreferredArt {
		return candidate.GroupsWithPreferredArt > current.GroupsWithPreferredArt
	}
	return candidate.GroupsWithPreferredScience > current.GroupsWithPreferredScience
}

func cloneGroups(groups []*groupPkg.Group) []*groupPkg.Group {
	cloned := make([]*groupPkg.Group, 0, len(groups))
	for _, group := range groups {
		clone := *group
		clone.Students = append([]string(nil), group.Students...)
		clone.ArtIDs = append([]string(nil), group.ArtIDs...)
		clone.SciIDs = append([]string(nil), group.SciIDs...)
		clone.ParentIDs = copyStringSet(group.ParentIDs)
		clone.ParentBookingIssues = copyStringMap(group.ParentBookingIssues)
		cloned = append(cloned, &clone)
	}
	return cloned
}

func copyParentBookingIssues(groups, sourceGroups []*groupPkg.Group) {
	issuesByID := make(map[string]map[string]string, len(sourceGroups))
	for _, group := range sourceGroups {
		issuesByID[group.ID] = group.ParentBookingIssues
	}

	for _, group := range groups {
		group.ParentBookingIssues = copyStringMap(issuesByID[group.ID])
	}
}

func copyStringSet(source map[string]struct{}) map[string]struct{} {
	copied := make(map[string]struct{}, len(source))
	for key := range source {
		copied[key] = struct{}{}
	}
	return copied
}

func copyStringMap(source map[string]string) map[string]string {
	copied := make(map[string]string, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func randomSourceForOptions(opts ScheduleOptions, runOffset int64) (*rand.Rand, int64) {
	if !opts.Random {
		return nil, 0
	}
	if opts.RandomSeedSet {
		seed := opts.RandomSeed + runOffset
		return rand.New(rand.NewSource(seed)), seed
	}
	seed := time.Now().UnixNano()
	return rand.New(rand.NewSource(seed)), seed
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

func shuffleGroups(groups []*groupPkg.Group, rng *rand.Rand) {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	for len(groups) > 0 {
		n := len(groups)
		randIndex := rng.Intn(n)
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
