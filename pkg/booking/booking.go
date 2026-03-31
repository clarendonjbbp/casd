package booking

import (
	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/sorter"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

type Booking = sorter.Booking
type ScheduleState = sorter.ScheduleState

func NewScheduleState(groups []*groupPkg.Group, workshopSets ...map[string]*workshopPkg.Workshop) *ScheduleState {
	sorterGroups := make([]*sorter.Group, len(groups))
	for i, group := range groups {
		sorterGroups[i] = (*sorter.Group)(group)
	}

	sorterWorkshopSets := make([]map[string]*sorter.Workshop, len(workshopSets))
	for i, workshops := range workshopSets {
		sorterWorkshopSets[i] = make(map[string]*sorter.Workshop, len(workshops))
		for id, workshop := range workshops {
			sorterWorkshopSets[i][id] = (*sorter.Workshop)(workshop)
		}
	}

	return sorter.NewScheduleState(sorterGroups, sorterWorkshopSets...)
}
