package model

const (
	NumSessions    = 4
	NumArtSessions = 2
	NumSciSessions = 2

	ArtWorkshop = iota
	SciWorkshop
)

var SessionTimes = []string{
	"9:40 - 10:10 am",
	"10:15 - 10:45 am",
	"10:50 - 11:05 am (Recess)",
	"11:10 - 11:40 am",
	"11:45 am - 12:15 pm",
}
