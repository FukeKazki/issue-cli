package tui

import (
	"fmt"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusPickerResult struct {
	Status   model.Status
	Selected bool
}

type statusPickerModel struct {
	header   string
	statuses []model.Status
	cursor   int
	current  model.Status
	result   StatusPickerResult
}

func RunStatusPicker(header string, current model.Status) (StatusPickerResult, error) {
	statuses := model.AllStatuses()
	idx := 0
	for i, s := range statuses {
		if s == current {
			idx = i
			break
		}
	}
	m := statusPickerModel{
		header:   header,
		statuses: statuses,
		cursor:   idx,
		current:  current,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return StatusPickerResult{}, err
	}
	return final.(statusPickerModel).result, nil
}

func (m statusPickerModel) Init() tea.Cmd { return nil }

func (m statusPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "ctrl+c", "esc", "q":
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.statuses)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "1", "2", "3", "4":
		i := int(keyMsg.Runes[0] - '1')
		if i >= 0 && i < len(m.statuses) {
			m.result = StatusPickerResult{Status: m.statuses[i], Selected: true}
			return m, tea.Quit
		}
	case "enter":
		m.result = StatusPickerResult{Status: m.statuses[m.cursor], Selected: true}
		return m, tea.Quit
	}
	return m, nil
}

func (m statusPickerModel) View() string {
	var b strings.Builder
	b.WriteString(panelHeaderStyle.Render(m.header))
	b.WriteString("\n\n")
	for i, s := range m.statuses {
		marker := " "
		if s == m.current {
			marker = "*"
		}
		badge := lipgloss.NewStyle().Foreground(statusColor[s]).Bold(true).Render(string(s))
		raw := fmt.Sprintf("%d %s %s", i+1, marker, badge)
		prefix := "  "
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render("▸ ")
		}
		b.WriteString(prefix + raw + "\n")
	}
	b.WriteString("\n")
	b.WriteString(footerStyle.Render("1-4 quick pick  ·  enter select  ·  esc cancel"))
	return panelStyle.Render(b.String())
}
