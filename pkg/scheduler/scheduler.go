package scheduler

import (
	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/sorter"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

type ScheduleOptions = sorter.ScheduleOptions

func ReadCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile string) ([]*groupPkg.Group, map[string]*workshopPkg.Workshop, map[string]*workshopPkg.Workshop, error) {
	groups, artWorkshops, sciWorkshops, err := sorter.ReadCSVFiles(groupsFile, artWorkshopsFile, sciWorkshopsFile)
	if err != nil {
		return nil, nil, nil, err
	}

	return castGroups(groups), castWorkshops(artWorkshops), castWorkshops(sciWorkshops), nil
}

func Schedule(groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, opts ScheduleOptions) error {
	return sorter.Schedule(castSorterGroups(groups), castSorterWorkshops(artWorkshops), castSorterWorkshops(sciWorkshops), opts)
}

func castGroups(groups []*sorter.Group) []*groupPkg.Group {
	out := make([]*groupPkg.Group, len(groups))
	for i, group := range groups {
		out[i] = (*groupPkg.Group)(group)
	}
	return out
}

func castWorkshops(workshops map[string]*sorter.Workshop) map[string]*workshopPkg.Workshop {
	out := make(map[string]*workshopPkg.Workshop, len(workshops))
	for id, workshop := range workshops {
		out[id] = (*workshopPkg.Workshop)(workshop)
	}
	return out
}

func castSorterGroups(groups []*groupPkg.Group) []*sorter.Group {
	out := make([]*sorter.Group, len(groups))
	for i, group := range groups {
		out[i] = (*sorter.Group)(group)
	}
	return out
}

func castSorterWorkshops(workshops map[string]*workshopPkg.Workshop) map[string]*sorter.Workshop {
	out := make(map[string]*sorter.Workshop, len(workshops))
	for id, workshop := range workshops {
		out[id] = (*sorter.Workshop)(workshop)
	}
	return out
}
