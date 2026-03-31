package booking

import (
	"fmt"
	"io"
	"sort"
	"strings"

	groupPkg "github.com/clarendonjbbp/casd/pkg/group"
	"github.com/clarendonjbbp/casd/pkg/model"
	workshopPkg "github.com/clarendonjbbp/casd/pkg/workshop"
)

func PrintGroupsHTML(w io.Writer, groups []*groupPkg.Group, state *ScheduleState) {
	summary := CalculateScheduleSummary(groups, state)

	_, _ = fmt.Fprintf(w, `<style>
.schedule-summary {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.96), rgba(237,248,255,0.94));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.08);
}
.schedule-summary h3 {
    margin: 0 0 0.8em;
    color: #141f26;
    font-family: "Montserrat", "Avenir Next", "Futura", "Trebuchet MS", sans-serif;
    font-size: 1.15rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}
.schedule-summary-grid {
    display: grid;
    gap: 0.85em;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}
.schedule-summary-card {
    padding: 0.95em 1em;
    border-radius: 18px;
    background: rgba(233, 244, 253, 0.9);
    border: 1px solid rgba(73, 135, 87, 0.12);
}
.schedule-summary-card .label {
    display: block;
    margin-bottom: 0.3em;
    color: #52717c;
    font-size: 0.85rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
}
.schedule-summary-card .value {
    color: #19364a;
    font-size: 1.45rem;
    font-weight: 700;
}
.group {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.95), rgba(244,250,255,0.95));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.1);
}
.group h3 {
    margin-top: 0;
    color: #141f26;
    border-bottom: 1px solid rgba(36, 55, 76, 0.14);
    padding-bottom: 0.6em;
    font-family: "Chewy", "Marker Felt", "Chalkboard SE", "Comic Sans MS", "Trebuchet MS", sans-serif;
    font-size: 1.35rem;
    text-transform: uppercase;
}
.group-details {
    margin: 1em 0;
    display: grid;
    gap: 0.35em;
}
.group-details p {
    margin: 0;
    color: #465963;
}
.group .schedule {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 1em;
    overflow: hidden;
    border-radius: 18px;
}
.group .schedule th,
.group .schedule td {
    padding: 10px 12px;
    text-align: left;
    border: 1px solid rgba(125, 138, 142, 0.2);
}
.group .schedule th {
    background-color: #e9f4fd;
    color: #1d4667;
}
.group .schedule .recess {
    background-color: #eef7ec;
    text-align: center;
    font-style: italic;
    color: #4a6c55;
}
.group .schedule .unfilled {
    color: #8a5c5c;
    font-style: italic;
}
</style>`)

	_, _ = fmt.Fprintf(w, "<div class='schedule-summary'>\n")
	_, _ = fmt.Fprintf(w, "<h3>Schedule Summary</h3>\n")
	_, _ = fmt.Fprintf(w, "<div class='schedule-summary-grid'>\n")
	writeSummaryCard(w, "Overall Satisfaction Points", fmt.Sprintf("%d", summary.OverallSatisfactionPoints))
	writeSummaryCard(w, "Average Satisfaction", fmt.Sprintf("%d%%", summary.AverageSatisfactionPercent))
	writeSummaryCard(w, "Groups With Preferred Art", fmt.Sprintf("%d / %d", summary.GroupsWithPreferredArt, summary.TotalGroups))
	writeSummaryCard(w, "Groups With Preferred Science", fmt.Sprintf("%d / %d", summary.GroupsWithPreferredScience, summary.TotalGroups))
	_, _ = fmt.Fprintf(w, "</div>\n</div>\n")

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	for _, group := range groups {
		printGroupHTML(w, group, state)
	}
}

func printGroupHTML(w io.Writer, group *groupPkg.Group, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "<div class='group'>\n")
	_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n", group.ID)
	_, _ = fmt.Fprintf(w, "<div class='group-details'>\n")
	_, _ = fmt.Fprintf(w, "<p><strong>Teacher:</strong> %s</p>\n", group.Teacher)
	if group.Grade == 0 {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> K</p>\n")
	} else {
		_, _ = fmt.Fprintf(w, "<p><strong>Grade:</strong> %d</p>\n", group.Grade)
	}
	_, _ = fmt.Fprintf(w, "<p><strong>Students:</strong> %s</p>\n", strings.Join(group.Students, ", "))
	_, _ = fmt.Fprintf(w, "<p><strong>Satisfaction:</strong> %d%%</p>\n", SatisfactionPercent(group, state))

	if len(group.ParentIDs) > 0 {
		parentIDs := make([]string, 0, len(group.ParentIDs))
		for parentID := range group.ParentIDs {
			parentIDs = append(parentIDs, parentID)
		}
		sort.Strings(parentIDs)
		_, _ = fmt.Fprintf(w, "<p><strong>Parent Workshops:</strong> %s</p>\n", strings.Join(parentIDs, ", "))
	}
	if issues := SortedParentBookingIssues(group); len(issues) > 0 {
		_, _ = fmt.Fprintf(w, "<p><strong>Unbooked Parent Workshops:</strong> %s</p>\n", strings.Join(issues, ", "))
	}
	_, _ = fmt.Fprintf(w, "</div>\n")

	_, _ = fmt.Fprintf(w, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(w, "<thead>\n<tr>\n<th>Time</th>\n<th>Workshop ID</th>\n<th>Workshop Name</th>\n<th>Room</th>\n<th>Match</th>\n</tr>\n</thead>\n")
	_, _ = fmt.Fprintf(w, "<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(model.SessionTimes); i++ {
		_, _ = fmt.Fprintf(w, "<tr>\n")
		if i == 2 {
			_, _ = fmt.Fprintf(w, "<td colspan='5' class='recess'>%s</td>\n", model.SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(w, "<td>%s</td>\n", model.SessionTimes[i])
			if workshopIndex < model.NumSessions && GetWorkshop(group, state, workshopIndex) != nil {
				workshop := GetWorkshop(group, state, workshopIndex)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.ID)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.Name)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", workshop.Room)
				_, _ = fmt.Fprintf(w, "<td>%s</td>\n", SessionSatisfactionLabel(group, state, workshopIndex))
			} else {
				_, _ = fmt.Fprintf(w, "<td colspan='4' class='unfilled'>Not Scheduled</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(w, "</tr>\n")
	}

	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n</div>\n")
}

func PrintWorkshopsHTML(w io.Writer, workshops map[string]*workshopPkg.Workshop, state *ScheduleState) {
	writeWorkshopsHTMLStyle(w)

	var sortedIDs []string
	for id := range workshops {
		sortedIDs = append(sortedIDs, id)
	}
	sort.Strings(sortedIDs)

	for _, id := range sortedIDs {
		printWorkshopHTML(w, workshops[id], state)
	}
}

func printWorkshopHTML(w io.Writer, workshop *workshopPkg.Workshop, state *ScheduleState) {
	_, _ = fmt.Fprintf(w, "<div class='workshop'>\n")
	_, _ = fmt.Fprintf(w, "<h3>%s</h3>\n", workshop.Name)
	_, _ = fmt.Fprintf(w, "<p><strong>ID:</strong> %s</p>\n", workshop.ID)
	_, _ = fmt.Fprintf(w, "<p><strong>Capacity per session:</strong> %d</p>\n", workshop.Capacity)
	_, _ = fmt.Fprintf(w, "<p><strong>Overall utilization:</strong> %d%%</p>\n", state.OverallUtilization(workshop))
	_, _ = fmt.Fprintf(w, "<table class='schedule'>\n")
	_, _ = fmt.Fprintf(w, "<thead>\n<tr>\n<th>Time</th>\n<th>Utilization</th>\n<th>Students</th>\n</tr>\n</thead>\n<tbody>\n")

	workshopIndex := 0
	for i := 0; i < len(model.SessionTimes); i++ {
		_, _ = fmt.Fprintf(w, "<tr>\n")
		if i == 2 {
			_, _ = fmt.Fprintf(w, "<td colspan='3' class='recess'>%s</td>\n", model.SessionTimes[i])
		} else {
			_, _ = fmt.Fprintf(w, "<td>%s</td>\n", model.SessionTimes[i])
			if workshopIndex < model.NumSessions && workshop.IsSessionOffered(workshopIndex) {
				utilization := state.Utilization(workshop, workshopIndex)
				utilizationClass := "normal"
				if utilization < 30 {
					utilizationClass = "low"
				} else if utilization > 80 {
					utilizationClass = "high"
				}

				_, _ = fmt.Fprintf(w, "<td class='utilization %s'>%d%%</td>\n", utilizationClass, utilization)
				_, _ = fmt.Fprintf(w, "<td class='students'>")
				groups := state.GroupsForWorkshopSession(workshop, workshopIndex)
				var studentNames []string
				for _, group := range groups {
					studentNames = append(studentNames, group.Students...)
				}
				_, _ = fmt.Fprintf(w, "%s", strings.Join(studentNames, ", "))
				_, _ = fmt.Fprintf(w, "</td>\n")
			} else {
				_, _ = fmt.Fprintf(w, "<td>-</td>\n<td>-</td>\n")
			}
			workshopIndex++
		}
		_, _ = fmt.Fprintf(w, "</tr>\n")
	}

	_, _ = fmt.Fprintf(w, "</tbody>\n</table>\n</div>\n")
}

func writeWorkshopsHTMLStyle(w io.Writer) {
	_, _ = fmt.Fprintf(w, `<style>
.workshop {
    margin-bottom: 1.5em;
    padding: 1.2em;
    border: 1px solid rgba(36, 55, 76, 0.14);
    border-radius: 24px;
    background: linear-gradient(180deg, rgba(255,255,255,0.95), rgba(244,250,255,0.95));
    box-shadow: 0 14px 38px rgba(36, 55, 76, 0.1);
}
.workshop h3 {
    margin-top: 0;
    color: #141f26;
    font-family: "Chewy", "Marker Felt", "Chalkboard SE", "Comic Sans MS", "Trebuchet MS", sans-serif;
    font-size: 1.35rem;
    text-transform: uppercase;
}
.schedule {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 1em;
}
.schedule th, .schedule td {
    padding: 10px 12px;
    text-align: left;
    border: 1px solid rgba(125, 138, 142, 0.2);
}
.schedule th {
    background-color: #e9f4fd;
    color: #1d4667;
}
.utilization {
    font-weight: bold;
}
.utilization.low {
    color: #c53a44;
}
.utilization.high {
    color: #2f8b51;
}
.students {
    font-size: 0.9em;
}
.schedule .recess {
    background-color: #eef7ec;
    text-align: center;
    font-style: italic;
    color: #4a6c55;
}
</style>
`)
}

func writeSummaryCard(w io.Writer, label, value string) {
	_, _ = fmt.Fprintf(w, "<div class='schedule-summary-card'><span class='label'>%s</span><span class='value'>%s</span></div>\n", label, value)
}
