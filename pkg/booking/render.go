package booking

import (
	"embed"
	htmltmpl "html/template"
	"io"
	"log"
	"sort"
	"strings"
	texttmpl "text/template"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

//go:embed templates/*.tmpl
var renderTemplateFS embed.FS

var htmlRenderTemplates = htmltmpl.Must(htmltmpl.ParseFS(renderTemplateFS, "templates/*.html.tmpl"))
var textRenderTemplates = texttmpl.Must(texttmpl.ParseFS(renderTemplateFS, "templates/*.txt.tmpl"))

type scheduleSummaryView struct {
	OverallSatisfactionPoints  int
	AverageSatisfactionPercent int
	GroupsWithPreferredArt     int
	GroupsWithPreferredScience int
	TotalGroups                int
}

type groupScheduleRowView struct {
	Time         string
	IsRecess     bool
	WorkshopID   string
	WorkshopName string
	Room         string
	Match        string
	Scheduled    bool
}

type groupView struct {
	Teacher               string
	Grade                 string
	ID                    string
	SatisfactionPercent   int
	Students              string
	ParentWorkshops       string
	HasParentWorkshops    bool
	ParentBookingIssues   string
	HasParentBookingIssue bool
	ScheduleRows          []groupScheduleRowView
}

type workshopScheduleRowView struct {
	Time             string
	IsRecess         bool
	Offered          bool
	Utilization      int
	UtilizationClass string
	Students         string
}

type workshopView struct {
	ID                 string
	Name               string
	Capacity           int
	OverallUtilization int
	ScheduleRows       []workshopScheduleRowView
}

type groupsSectionView struct {
	Summary     scheduleSummaryView
	Groups      []groupView
	ShowScoring bool
	ShowSummary bool
}

type workshopsSectionView struct {
	Workshops []workshopView
}

type resultsHTMLView struct {
	Groups  groupsSectionView
	Art     workshopsSectionView
	Science workshopsSectionView
}

type cliReportView struct {
	Groups  groupsSectionView
	Art     workshopsSectionView
	Science workshopsSectionView
}

func PrintScheduleReport(w io.Writer, groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	view := cliReportView{
		Groups:  buildGroupsSectionView(groups, state, true, true),
		Art:     buildWorkshopsSectionView(artWorkshops, state),
		Science: buildWorkshopsSectionView(sciWorkshops, state),
	}
	executeTextTemplate(w, "report.txt.tmpl", view)
}

func PrintScheduleSummary(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	executeTextTemplate(w, "summary.txt.tmpl", buildGroupsSectionView(groups, state, true, true))
}

func PrintGroups(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	executeTextTemplate(w, "groups.txt.tmpl", buildGroupsSectionView(groups, state, true, true))
}

func PrintWorkshops(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	executeTextTemplate(w, "workshops.txt.tmpl", buildWorkshopsSectionView(workshops, state))
}

func PrintResultsHTML(w io.Writer, groups []*groupPkg.Group, artWorkshops, sciWorkshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	view := resultsHTMLView{
		Groups:  buildGroupsSectionView(groups, state, true, true),
		Art:     buildWorkshopsSectionView(artWorkshops, state),
		Science: buildWorkshopsSectionView(sciWorkshops, state),
	}
	executeHTMLTemplate(w, "results_sections.html.tmpl", view)
}

func PrintFriendlyResultsHTML(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	executeHTMLTemplate(w, "groups.html.tmpl", buildGroupsSectionView(groups, state, false, false))
}

func PrintGroupsHTML(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	executeHTMLTemplate(w, "groups.html.tmpl", buildGroupsSectionView(groups, state, true, true))
}

func PrintWorkshopsHTML(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	executeHTMLTemplate(w, "workshops.html.tmpl", buildWorkshopsSectionView(workshops, state))
}

func buildGroupsSectionView(groups []*groupPkg.Group, state *ScheduleState, showScoring, showSummary bool) groupsSectionView {
	sortedGroups := append([]*groupPkg.Group(nil), groups...)
	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].ID < sortedGroups[j].ID
	})

	groupViews := make([]groupView, 0, len(sortedGroups))
	for _, group := range sortedGroups {
		groupViews = append(groupViews, buildGroupView(group, state))
	}

	summary := CalculateScheduleSummary(groups, state)
	return groupsSectionView{
		Summary:     scheduleSummaryView(summary),
		Groups:      groupViews,
		ShowScoring: showScoring,
		ShowSummary: showSummary,
	}
}

func buildGroupView(group *groupPkg.Group, state *ScheduleState) groupView {
	parentWorkshops := strings.Join(group.SortedParentIDs(), ", ")
	parentIssues := strings.Join(SortedParentBookingIssues(group), ", ")

	rows := make([]groupScheduleRowView, 0, len(model.SessionTimes))
	workshopIndex := 0
	for i, timeLabel := range model.SessionTimes {
		if i == 2 {
			rows = append(rows, groupScheduleRowView{
				Time:     timeLabel,
				IsRecess: true,
			})
			continue
		}

		row := groupScheduleRowView{Time: timeLabel}
		if workshopIndex < model.NumSessions {
			workshop := GetWorkshop(group, state, workshopIndex)
			if workshop != nil {
				row.Scheduled = true
				row.WorkshopID = workshop.ID
				row.WorkshopName = workshop.Name
				row.Room = workshop.Room
				row.Match = SessionSatisfactionLabel(group, state, workshopIndex)
			}
		}
		rows = append(rows, row)
		workshopIndex++
	}

	return groupView{
		Teacher:               group.Teacher,
		Grade:                 gradeLabel(group.Grade),
		ID:                    group.ID,
		SatisfactionPercent:   SatisfactionPercent(group, state),
		Students:              strings.Join(group.Students, ", "),
		ParentWorkshops:       parentWorkshops,
		HasParentWorkshops:    parentWorkshops != "",
		ParentBookingIssues:   parentIssues,
		HasParentBookingIssue: parentIssues != "",
		ScheduleRows:          rows,
	}
}

func buildWorkshopsSectionView(workshops map[string]*workshopPkg.Workshop, state *ScheduleState) workshopsSectionView {
	ids := make([]string, 0, len(workshops))
	for id := range workshops {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	views := make([]workshopView, 0, len(ids))
	for _, id := range ids {
		views = append(views, buildWorkshopView(workshops[id], state))
	}

	return workshopsSectionView{Workshops: views}
}

func buildWorkshopView(workshop *workshopPkg.Workshop, state *ScheduleState) workshopView {
	rows := make([]workshopScheduleRowView, 0, len(model.SessionTimes))
	workshopIndex := 0
	for i, timeLabel := range model.SessionTimes {
		if i == 2 {
			rows = append(rows, workshopScheduleRowView{
				Time:     timeLabel,
				IsRecess: true,
			})
			continue
		}

		row := workshopScheduleRowView{Time: timeLabel}
		if workshopIndex < model.NumSessions && workshop.IsSessionOffered(workshopIndex) {
			utilization := state.Utilization(workshop, workshopIndex)
			row.Offered = true
			row.Utilization = utilization
			row.UtilizationClass = utilizationClass(utilization)

			groups := state.GroupsForWorkshopSession(workshop, workshopIndex)
			studentNames := make([]string, 0)
			for _, group := range groups {
				studentNames = append(studentNames, group.Students...)
			}
			if len(studentNames) == 0 {
				row.Students = "-"
			} else {
				row.Students = strings.Join(studentNames, ", ")
			}
		}

		rows = append(rows, row)
		workshopIndex++
	}

	return workshopView{
		ID:                 workshop.ID,
		Name:               workshop.Name,
		Capacity:           workshop.Capacity,
		OverallUtilization: state.OverallUtilization(workshop),
		ScheduleRows:       rows,
	}
}

func gradeLabel(grade int) string {
	return model.GradeLabel(grade)
}

func utilizationClass(utilization int) string {
	switch {
	case utilization < 30:
		return "low"
	case utilization > 80:
		return "high"
	default:
		return "normal"
	}
}

func executeHTMLTemplate(w io.Writer, name string, data any) {
	if err := htmlRenderTemplates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("unable to execute html template %q: %v", name, err)
	}
}

func executeTextTemplate(w io.Writer, name string, data any) {
	if err := textRenderTemplates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("unable to execute text template %q: %v", name, err)
	}
}
