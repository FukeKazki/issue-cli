package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/FukeKazki/issue-cli/internal/model"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	labelStyle  = lipgloss.NewStyle().Faint(true)
	statusColor = map[model.Status]lipgloss.Color{
		model.StatusTODO:       "8",
		model.StatusInProgress: "11",
		model.StatusReviews:    "13",
		model.StatusDone:       "10",
	}
)

func RenderDetail(iss *model.Issue) string {
	var b strings.Builder

	fmt.Fprintln(&b, titleStyle.Render(fmt.Sprintf("#%d  %s", iss.ID, iss.Title)))
	fmt.Fprintln(&b)

	statusBadge := lipgloss.NewStyle().Foreground(statusColor[iss.Status]).Bold(true).Render(string(iss.Status))
	fmt.Fprintln(&b, labelStyle.Render("Status: ")+statusBadge)
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, labelStyle.Render("Description:"))
	if strings.TrimSpace(iss.Description) == "" {
		fmt.Fprintln(&b, "  (none)")
	} else {
		for _, line := range strings.Split(iss.Description, "\n") {
			fmt.Fprintln(&b, "  "+line)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, labelStyle.Render("References:"))
	if len(iss.References) == 0 {
		fmt.Fprintln(&b, "  (none)")
	} else {
		for _, r := range iss.References {
			fmt.Fprintln(&b, "  - "+r)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, labelStyle.Render("Scope:"))
	if len(iss.Scope) == 0 {
		fmt.Fprintln(&b, "  (none)")
	} else {
		for _, s := range iss.Scope {
			fmt.Fprintln(&b, "  - "+s)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, labelStyle.Render(fmt.Sprintf("Created: %s", fmtTime(iss.CreatedAt))))
	fmt.Fprintln(&b, labelStyle.Render(fmt.Sprintf("Updated: %s", fmtTime(iss.UpdatedAt))))
	return b.String()
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05 -0700")
}
