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
	minUtilization := flag.Int("min-utilization", 30, "Minimum utilization for a workshop session")
	flag.Parse()

	groups, artWorkshops, sciWorkshops, err := scheduler.ReadCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	state, err := scheduler.Schedule(groups, artWorkshops, sciWorkshops, scheduler.ScheduleOptions{
		Random:                    *random,
		MinUtilization:            *minUtilization,
		ContinueOnParentLookupErr: false,
	})
	if err != nil {
		log.Fatal(err)
	}

	if *printOutput {
		booking.PrintScheduleReport(os.Stdout, groups, artWorkshops, sciWorkshops, state)
	}
}
