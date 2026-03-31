package booking

import (
	"fmt"
	"io"
	"log"
	"slices"
	"sort"
	"strings"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

func PrintScheduleSummary(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	summary := CalculateScheduleSummary(groups, state)

	_, _ = fmt.Fprintln(w, "## Schedule Summary")
	_, _ = fmt.Fprintf(w, "- Overall satisfaction points: %d\n", summary.OverallSatisfactionPoints)
	_, _ = fmt.Fprintf(w, "- Average satisfaction: %d%%\n", summary.AverageSatisfactionPercent)
	_, _ = fmt.Fprintf(w, "- Groups with at least 1 preferred art workshop: %d / %d\n", summary.GroupsWithPreferredArt, summary.TotalGroups)
	_, _ = fmt.Fprintf(w, "- Groups with at least 1 preferred science workshop: %d / %d\n", summary.GroupsWithPreferredScience, summary.TotalGroups)
	_, _ = fmt.Fprintln(w)
}

func PrintGroups(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	PrintScheduleSummary(w, groups, state)

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		printGroup(w, group, state)
	}
}

func printGroup(w io.Writer, group *groupPkg.Group, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "Teacher = %s  \n", group.Teacher)
	if group.Grade == 0 {
		_, _ = fmt.Fprintf(w, "Grade = K  \n")
	} else {
		_, _ = fmt.Fprintf(w, "Grade = %d  \n", group.Grade)
	}
	_, _ = fmt.Fprintf(w, "ID = %s  \n", group.ID)
	_, _ = fmt.Fprintf(w, "Satisfaction = %d%%  \n", SatisfactionPercent(group, state))
	_, _ = fmt.Fprintf(w, "Students =  %v  \n", strings.Join(group.Students, ","))
	if len(group.ParentIDs) > 0 {
		parentIDs := make([]string, 0, len(group.ParentIDs))
		for parentID := range group.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		sort.Strings(parentIDs)
		_, _ = fmt.Fprintf(w, "Group contains child of presenter or assistant of workshop = %v  \n", strings.Join(parentIDs, ","))
	}
	if issues := SortedParentBookingIssues(group); len(issues) > 0 {
		_, _ = fmt.Fprintf(w, "Unbooked parent workshops = %s  \n", strings.Join(issues, ", "))
	}
	_, _ = fmt.Fprintln(w, "Schedule")
	_, _ = fmt.Fprintln(w, "| ID | Class | Room | Match |")
	_, _ = fmt.Fprintln(w, "| -- | ----- | ---- | ----- |")
	for session := 0; session < model.NumSessions; session++ {
		workshop := GetWorkshop(group, state, session)
		if workshop != nil {
			_, _ = fmt.Fprintf(w, "| %s | %s | %s | %s |\n", workshop.ID, workshop.Name, workshop.Room, SessionSatisfactionLabel(group, state, session))
		} else {
			_, _ = fmt.Fprintf(w, "| - | - | - | Not Scheduled |\n")
			log.Printf("====UNFILLED SLOT====\n")
		}
	}
	_, _ = fmt.Fprintf(w, "\n---\n\n")
}

func PrintWorkshops(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	var sortedIDs []string
	for id := range workshops {
		sortedIDs = append(sortedIDs, id)
	}
	slices.Sort(sortedIDs)
	for _, id := range sortedIDs {
		printWorkshop(w, workshops[id], state)
	}
}

func printWorkshop(w io.Writer, workshop *workshopPkg.Workshop, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "ID: %s  \n", workshop.ID)
	_, _ = fmt.Fprintf(w, "Name: %s  \n", workshop.Name)
	_, _ = fmt.Fprintf(w, "Capacity per session: %d  \n", workshop.Capacity)
	_, _ = fmt.Fprintf(w, "Overall utilization: %d%%  \n", state.OverallUtilization(workshop))
	_, _ = fmt.Fprintf(w, "Schedule  \n")
	_, _ = fmt.Fprintln(w, "| Utilization | Students |")
	_, _ = fmt.Fprintln(w, "| --------- | -------- |")
	for i := 0; i < model.NumSessions; i++ {
		if !workshop.IsSessionOffered(i) {
			_, _ = fmt.Fprintln(w, "| - | - |")
			continue
		}

		_, _ = fmt.Fprintf(w, "| %d%% | ", state.Utilization(workshop, i))
		groups := state.GroupsForWorkshopSession(workshop, i)
		for _, group := range groups {
			_, _ = fmt.Fprintf(w, "%v,", strings.Join(group.Students, ","))
		}
		_, _ = fmt.Fprintf(w, " |\n")
	}
	_, _ = fmt.Fprintf(w, "\n---\n\n")
}
