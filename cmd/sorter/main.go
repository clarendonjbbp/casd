package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/clarendonjbbp/casd/pkg/sorter"
)

const (
	numArtSessions = 2
	numSciSessions = 2
)

func main() {
	groupsFile := flag.String("groups", "groups.csv", "CSV file with groups")
	artWorkshopsFile := flag.String("art-workshops", "artworkshops.csv", "CSV file with Art workshops")
	sciWorkshopsFile := flag.String("science-workshops", "scienceworkshops.csv", "CSV file with Science workshops")
	printOutput := flag.Bool("print-output", true, "Print list of groups and workshop assignments")
	random := flag.Bool("random", false, "Randomize input")
	minUtilization := flag.Int("min-utilization", 30, "Minimum utilization for a workshop session")
	flag.Parse()

	groups, artWorkshops, sciWorkshops, err := readCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("====Booking Parent Classes===\n")
	for _, group := range groups {
		for parentID := range group.ParentIDs {
			workshop, err := sorter.GetWorkshopFromID(parentID, artWorkshops, sciWorkshops)
			if err != nil {
				log.Fatal(fmt.Errorf("Error finding parent class for teacher=%s group=%s: %w", group.Teacher, group.Name, err))
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if !booked {
				log.Printf("Unable to book parent ID=%s. teacher=%s group=%s\n", parentID, group.Teacher, group.Name)
			}
		}
	}

	if *random {
		shuffle(groups)
	}

	log.Printf("\n====Booking Art Classes===\n")
	var needsRandomArt []*sorter.Group
	for _, group := range groups {
		sessionsToBook := numArtSessions - group.SessionsBooked(sorter.ArtWorkshop)
		if sessionsToBook < 1 {
			continue
		}
		for _, id := range group.ArtIDs {
			workshop, ok := artWorkshops[id]
			if !ok {
				log.Printf("ID %s not found teacher=%s group=%s\n", id, group.Teacher, group.Name)
				continue
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomArt = append(needsRandomArt, group)
		}
	}

	log.Printf("\n\n====Booking Science Classes===\n")
	var needsRandomSci []*sorter.Group
	for _, group := range groups {
		sessionsToBook := numSciSessions - group.SessionsBooked(sorter.SciWorkshop)
		if sessionsToBook < 1 {
			continue
		}
		for _, id := range group.SciIDs {
			workshop, ok := sciWorkshops[id]
			if !ok {
				log.Printf("ID %s not found teacher=%s group=%s\n", id, group.Teacher, group.Name)
				continue
			}
			booked := sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				sessionsToBook--
				if sessionsToBook == 0 {
					break
				}
			}
		}

		// Select random session
		for i := 0; i < sessionsToBook; i++ {
			needsRandomSci = append(needsRandomSci, group)
		}
	}

	sortedArtWorkshops := sorter.SortWorkshopsByOverallUtilization(artWorkshops)
	sortedSciWorkshops := sorter.SortWorkshopsByOverallUtilization(sciWorkshops)

	// Assign random sessions if needed
	log.Printf("\n\n====Booking Random Art Classes===\n")
	for _, group := range needsRandomArt {
		booked := false
		for _, workshop := range sortedArtWorkshops {
			booked = sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				break
			}
		}
		if !booked {
			for _, workshop := range sortedSciWorkshops {
				booked = sorter.BookWorkshopIfAvailable(workshop, group)
				if booked {
					break
				}
			}
			if !booked {
				log.Printf("Still not found Art for %s %s\n", group.Teacher, group.Name)
			}
		}
	}

	sortedArtWorkshops = sorter.SortWorkshopsByOverallUtilization(artWorkshops)
	sortedSciWorkshops = sorter.SortWorkshopsByOverallUtilization(sciWorkshops)

	log.Printf("\n\n====Booking Random Science Classes===\n")
	for _, group := range needsRandomSci {
		booked := false
		for _, workshop := range sortedSciWorkshops {
			booked = sorter.BookWorkshopIfAvailable(workshop, group)
			if booked {
				break
			}
		}
		if !booked {
			for _, workshop := range sortedArtWorkshops {
				booked = sorter.BookWorkshopIfAvailable(workshop, group)
				if booked {
					break
				}
			}
			if !booked {
				log.Printf("Still not found Sci for %s %s\n", group.Teacher, group.Name)
			}
		}
	}

	if err := rebalanceWorkshop(*minUtilization, artWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance art workshops: %v", err)
	}
	if err := rebalanceWorkshop(*minUtilization, sciWorkshops, groups); err != nil {
		log.Printf("Unable to rebalance science workshops: %v", err)
	}

	if *printOutput {
		fmt.Printf("# Community Arts and Sciences Day Assignments  \n\n")
		fmt.Printf("## Groups  \n\n")
		sorter.PrintGroups(os.Stdout, groups)
		fmt.Printf("## Art Workshops \n\n")
		sorter.PrintWorkshops(os.Stdout, artWorkshops)
		fmt.Printf("## Science Workshops  \n\n")
		sorter.PrintWorkshops(os.Stdout, sciWorkshops)
	}
}

func readCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile string) ([]*sorter.Group, map[string]*sorter.Workshop, map[string]*sorter.Workshop, error) {
	groups, err := sorter.ReadGroups(groupsFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't read groups: %v", err)
	}

	artWorkshops, err := sorter.ReadWorkshops(artWorkshopsFile, sorter.ArtWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't read art workshop: %v", err)
	}

	sciWorkshops, err := sorter.ReadWorkshops(sciWorkshopsFile, sorter.SciWorkshop)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Couldn't read science workshop: %v", err)
	}

	return groups, artWorkshops, sciWorkshops, nil
}

func rebalanceWorkshop(minUtilization int, workshops map[string]*sorter.Workshop, groups []*sorter.Group) error {
	for maxPreferance := 1; maxPreferance < 6; maxPreferance++ {
		underutilizedWorkshops, underutilizedWorkshopSessions := sorter.GetUnderutilizedSessions(minUtilization, workshops)
		if len(underutilizedWorkshops) == 0 {
			return nil
		}
		for i := range underutilizedWorkshops {
			workshop := underutilizedWorkshops[i]
			session := underutilizedWorkshopSessions[i]
			log.Printf("Rebalancing %s at %d%% utilization for session %d\n", workshop.Name, workshop.Utilization(session), session)
			for _, group := range groups {
				if !workshop.WithinGradeRange(group.Grade) {
					continue
				}
				if group.IsEnrolledInWorkshop(workshop.GetID()) {
					continue
				}
				if workshop.SpotsAvailable[session] < group.NumStudents() {
					continue
				}
				oldWorkshop := group.GetWorkshop(session)
				if oldWorkshop.Kind != workshop.Kind {
					continue
				}

				if oldWorkshop.UtilizationWithoutGroup(session, group) < minUtilization {
					continue
				}
				preferance := group.HowPreferredIsBookedWorkshop(session)
				if preferance < maxPreferance {
					log.Printf("Rebalancing with group teacher=%s name=%s", group.Teacher, group.Name)
					oldWorkshop.UnbookSession(session, group)
					workshop.TakeSession(session, group)
					group.BookWorkshop(session, workshop)
					break
				}

			}
		}
	}

	return errors.New("Unable to rebalance workshop")
}

func shuffle(vals []*sorter.Group) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(vals) > 0 {
		n := len(vals)
		randIndex := r.Intn(n)
		vals[n-1], vals[randIndex] = vals[randIndex], vals[n-1]
		vals = vals[:n-1]
	}
}
