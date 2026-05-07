package main

import (
	"flag"
	"log"
	"os"

	"github.com/clarendonjbbp/casd/pkg/booking"
	"github.com/clarendonjbbp/casd/pkg/scheduler"
)

func main() {
	groupsFile := flag.String("groups", "groups.csv", "CSV file with groups")
	artWorkshopsFile := flag.String("art-workshops", "artworkshops.csv", "CSV file with Art workshops")
	sciWorkshopsFile := flag.String("science-workshops", "scienceworkshops.csv", "CSV file with Science workshops")
	printOutput := flag.Bool("print-output", true, "Print list of groups and workshop assignments")
	random := flag.Bool("random", false, "Randomize input")
	randomRuns := flag.Int("random-runs", 1, "Run N randomized schedules and keep the best result")
	randomSeed := flag.Int64("random-seed", 0, "Seed for reproducible random or optimized runs")
	minUtilization := flag.Int("min-utilization", 30, "Minimum utilization for a workshop session")
	flag.Parse()
	randomSeedSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "random-seed" {
			randomSeedSet = true
		}
	})

	groups, artWorkshops, sciWorkshops, err := scheduler.ReadCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	state, err := scheduler.Schedule(groups, artWorkshops, sciWorkshops, scheduler.ScheduleOptions{
		Random:                    *random,
		MinUtilization:            *minUtilization,
		ContinueOnParentLookupErr: false,
		RandomRuns:                *randomRuns,
		RandomSeed:                *randomSeed,
		RandomSeedSet:             randomSeedSet,
	})
	if err != nil {
		log.Fatal(err)
	}

	if *printOutput {
		booking.PrintScheduleReport(os.Stdout, groups, artWorkshops, sciWorkshops, state)
	}
}
