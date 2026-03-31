package workshop

import "github.com/clarendonjbbp/casd/pkg/sorter"

type Workshop = sorter.Workshop

var ReadWorkshops = sorter.ReadWorkshops
var SortWorkshopsByOverallUtilization = sorter.SortWorkshopsByOverallUtilization
var GetUnderutilizedSessions = sorter.GetUnderutilizedSessions
var PrintWorkshops = sorter.PrintWorkshops
var PrintWorkshopsHTML = sorter.PrintWorkshopsHTML
