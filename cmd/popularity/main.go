package main

import (
	"flag"
	"log"
	"os"

	popularityPkg "github.com/clarendonjbbp/casd/pkg/popularity"
	"github.com/clarendonjbbp/casd/pkg/scheduler"
)

func main() {
	groupsFile := flag.String("groups", "groups.csv", "CSV file with groups")
	artWorkshopsFile := flag.String("art-workshops", "artworkshops.csv", "CSV file with Art workshops")
	sciWorkshopsFile := flag.String("science-workshops", "scienceworkshops.csv", "CSV file with Science workshops")
	limit := flag.Int("limit", 5, "Number of most and least popular workshops to print")
	format := flag.String("format", string(textFormat), "Output format: text, markdown, or html")
	flag.Parse()

	groups, artWorkshops, sciWorkshops, err := scheduler.ReadCSVFiles(*groupsFile, *artWorkshopsFile, *sciWorkshopsFile)
	if err != nil {
		log.Fatal(err)
	}

	popularity := popularityPkg.Calculate(groups, artWorkshops, sciWorkshops)
	reportFormat, err := parseReportFormat(*format)
	if err != nil {
		log.Fatal(err)
	}
	printPopularityReport(os.Stdout, popularity, *limit, reportFormat)
}
