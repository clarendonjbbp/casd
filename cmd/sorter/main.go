package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/clarendonjbbp/casd/pkg/sorter"
)

func main() {
	groupsFile := flag.String("groups", "groups.csv", "CSV file with groups")
	artWorkshopsFile := flag.String("art-workshops", "artworkshops.csv", "CSV file with Art workshops")
	sciWorkshopsFile := flag.String("science-workshops", "scienceworkshops.csv", "CSV file with Science workshops")
	printOutput := flag.Bool("print-output", true, "Print list of groups and workshop assignments")
	random := flag.Bool("random", false, "Randomize input")
	minUtilization := flag.Int("min-utilization", 30, "Minimum utilization for a workshop session")
	flag.Parse()

	groups, artWorkshops, sciWorkshops, err := sorter.ReadCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	if err := sorter.Schedule(groups, artWorkshops, sciWorkshops, sorter.ScheduleOptions{
		Random:                    *random,
		MinUtilization:            *minUtilization,
		ContinueOnParentLookupErr: false,
	}); err != nil {
		log.Fatal(err)
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
