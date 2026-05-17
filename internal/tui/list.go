package tui

import (
	"fmt"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ListAction int

const (
	ListActionQuit ListAction = iota
	ListActionShow
	ListActionCheckout
	ListActionEdit
	ListActionCreate
	ListActionStatus
	ListActionDelete
)

type ListResult struct {
	Action  ListAction
	IssueID int
}

const (
	listMinWidthForPreview = 80
	listPreviewPct         = 50
)

type listModel struct {
	header      string
	issues      []model.Issue
	filtered    []int
	cursor      int
	topIdx      int
	showPreview bool
	filtering   bool
	filterInput textinput.Model
	width       int
	height      int
	result      ListResult
}

func RunList(issues []model.Issue, header string, initialID int) (ListResult, error) {
	m := newListModel(issues, header, initialID)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return ListResult{}, err
	}
	return final.(listModel).result, nil
}

func newListModel(issues []model.Issue, header string, initialID int) listModel {
	ti := textinput.New()
	ti.Placeholder = "filter title or status"
	ti.Prompt = "/ "
	ti.CharLimit = 100
	ti.Width = 40

	m := listModel{
		header:      header,
		issues:      issues,
		filterInput: ti,
		showPreview: true,
	}
	m.applyFilter()
	if initialID > 0 {
		for i, idx := range m.filtered {
			if m.issues[idx].ID == initialID {
				m.cursor = i
				break
			}
		}
	}
	return m
}

func (m *listModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filterInput.Value()))
	m.filtered = m.filtered[:0]
	for i, iss := range m.issues {
		if q == "" ||
			strings.Contains(strings.ToLower(iss.Title), q) ||
			strings.Contains(strings.ToLower(string(iss.Status)), q) ||
			strings.Contains(fmt.Sprintf("#%d", iss.ID), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m listModel) Init() tea.Cmd { return nil }

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.filtering {
			return m.updateFiltering(msg)
		}
		return m.updateBrowsing(msg)
	}
	return m, nil
}

func (m listModel) updateFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filterInput.SetValue("")
		m.filterInput.Blur()
		m.applyFilter()
		return m, nil
	case "enter":
		m.filtering = false
		m.filterInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.applyFilter()
	return m, cmd
}

func (m listModel) updateBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.result = ListResult{Action: ListActionQuit}
		return m, tea.Quit
	case "esc":
		if m.filterInput.Value() != "" {
			m.filterInput.SetValue("")
			m.applyFilter()
			return m, nil
		}
		m.result = ListResult{Action: ListActionQuit}
		return m, tea.Quit
	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "g", "home":
		m.cursor = 0
	case "G", "end":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}
	case "/":
		m.filtering = true
		m.filterInput.Focus()
		return m, textinput.Blink
	case "v":
		m.showPreview = !m.showPreview
	case "enter":
		if id := m.currentID(); id > 0 {
			m.result = ListResult{Action: ListActionShow, IssueID: id}
			return m, tea.Quit
		}
	case "c":
		if id := m.currentID(); id > 0 {
			m.result = ListResult{Action: ListActionCheckout, IssueID: id}
			return m, tea.Quit
		}
	case "e":
		if id := m.currentID(); id > 0 {
			m.result = ListResult{Action: ListActionEdit, IssueID: id}
			return m, tea.Quit
		}
	case "n":
		m.result = ListResult{Action: ListActionCreate}
		return m, tea.Quit
	case "s":
		if id := m.currentID(); id > 0 {
			m.result = ListResult{Action: ListActionStatus, IssueID: id}
			return m, tea.Quit
		}
	case "d":
		if id := m.currentID(); id > 0 {
			m.result = ListResult{Action: ListActionDelete, IssueID: id}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m listModel) currentID() int {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return 0
	}
	return m.issues[m.filtered[m.cursor]].ID
}

func (m listModel) currentIssue() *model.Issue {
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return nil
	}
	iss := m.issues[m.filtered[m.cursor]]
	return &iss
}

func (m listModel) View() string {
	listW := m.width
	previewW := 0
	if m.showPreview && m.width >= listMinWidthForPreview {
		previewW = m.width * listPreviewPct / 100
		listW = m.width - previewW - 1
	}
	if listW < 30 {
		listW = 30
	}

	listPanel := m.renderListPanel(listW)
	if previewW > 0 {
		previewPanel := m.renderPreviewPanel(previewW)
		body := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, previewPanel)
		return body + "\n" + m.renderFooter()
	}
	return listPanel + "\n" + m.renderFooter()
}

func (m listModel) panelInnerHeight() int {
	h := m.height - 4
	if h < 5 {
		h = 5
	}
	return h
}

func (m listModel) renderListPanel(w int) string {
	header := m.header
	if header == "" {
		header = "Issues"
	}

	var b strings.Builder
	b.WriteString(panelHeaderStyle.Render(header))
	b.WriteString("\n")

	if m.filtering || m.filterInput.Value() != "" {
		b.WriteString(m.filterInput.View())
		b.WriteString("\n")
	}

	if len(m.filtered) == 0 {
		if m.filterInput.Value() != "" {
			b.WriteString(hintStyle.Render("(no issues match filter)"))
		} else {
			b.WriteString(hintStyle.Render("(no issues — press 'n' to create, 'q' to quit)"))
		}
		return panelStyle.Width(w).Render(b.String())
	}

	visible := m.panelInnerHeight() - 2
	if m.filtering || m.filterInput.Value() != "" {
		visible--
	}
	if visible < 3 {
		visible = 3
	}

	top := m.topIdx
	if m.cursor < top {
		top = m.cursor
	}
	if m.cursor >= top+visible {
		top = m.cursor - visible + 1
	}
	if top < 0 {
		top = 0
	}
	end := top + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	innerW := w - 4
	if innerW < 20 {
		innerW = 20
	}

	for i := top; i < end; i++ {
		iss := m.issues[m.filtered[i]]
		idCol := fmt.Sprintf("#%d", iss.ID)
		statusBadge := lipgloss.NewStyle().
			Foreground(statusColor[iss.Status]).
			Bold(true).
			Render(fmt.Sprintf("[%s]", iss.Status))
		raw := fmt.Sprintf("%-5s %s  %s", idCol, statusBadge, iss.Title)
		prefix := "  "
		line := raw
		if i == m.cursor {
			prefix = lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render("▸ ")
			line = lipgloss.NewStyle().Foreground(colAccent).Bold(true).Render(raw)
		}
		b.WriteString(prefix + line + "\n")
	}

	if top > 0 || end < len(m.filtered) {
		b.WriteString(hintStyle.Render(fmt.Sprintf("\n[%d/%d]", m.cursor+1, len(m.filtered))))
	}

	return panelStyle.Width(w).Render(b.String())
}

func (m listModel) renderPreviewPanel(w int) string {
	body := panelHeaderStyle.Render("PREVIEW") + "\n"
	if iss := m.currentIssue(); iss != nil {
		body += RenderDetail(iss)
	} else {
		body += hintStyle.Render("(no selection)")
	}
	return panelStyle.Width(w).Render(body)
}

func (m listModel) renderFooter() string {
	if m.filtering {
		return footerStyle.Render("enter accept  ·  esc clear")
	}
	keys := []string{
		"enter show",
		"c checkout",
		"n create",
		"e edit",
		"s status",
		"d delete",
		"v preview",
		"/ filter",
		"q quit",
	}
	return footerStyle.Render(strings.Join(keys, "  ·  "))
}
