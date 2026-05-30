package tui

import (
	"fmt"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type ListAction int

const (
	ListActionQuit ListAction = iota
	ListActionShow
	ListActionCheckout
	ListActionEdit
	ListActionNew
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
	header        string
	issues        []model.Issue
	filtered      []int
	cursor        int
	topIdx        int
	showPreview   bool
	filtering     bool
	filterInput   textinput.Model
	width         int
	height        int
	result        ListResult
	previewOffset int
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
	m.previewOffset = 0
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
			m.previewOffset = 0
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.previewOffset = 0
		}
	case "g", "home":
		m.cursor = 0
		m.previewOffset = 0
	case "G", "end":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.previewOffset = 0
		}
	case "J":
		if iss := m.currentIssue(); iss != nil {
			lines := strings.Split(RenderDetail(iss, m.parentOf(iss), m.childrenOf(iss.ID)), "\n")
			if len(lines) > 0 && lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1]
			}
			headerLines := strings.Count(panelHeaderStyle.Render("PREVIEW")+"\n", "\n")
			maxVisible := m.panelInnerHeight() - headerLines
			if maxVisible < 1 {
				maxVisible = 1
			}
			maxOffset := len(lines) - maxVisible
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.previewOffset < maxOffset {
				m.previewOffset++
			}
		}
	case "K":
		if m.previewOffset > 0 {
			m.previewOffset--
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
		m.result = ListResult{Action: ListActionNew}
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

// listColumns splits the terminal width into the list panel width and the
// preview panel width. The preview is only shown when enabled and the terminal
// is at least listMinWidthForPreview wide. The list width is floored at 30 for
// readability but never allowed to exceed the terminal width, so a narrow
// terminal shrinks the panel instead of overflowing past the right edge.
func listColumns(width int, showPreview bool) (listW, previewW int) {
	listW = width
	if showPreview && width >= listMinWidthForPreview {
		previewW = width * listPreviewPct / 100
		listW = width - previewW - 1
	}
	if listW < 30 {
		listW = 30
	}
	if width > 0 && listW > width {
		listW = width
		previewW = 0
	}
	return listW, previewW
}

func (m listModel) View() string {
	listW, previewW := listColumns(m.width, m.showPreview)

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
		badgeText := fmt.Sprintf("[%s]", iss.Status)
		statusBadge := lipgloss.NewStyle().
			Foreground(statusColor[iss.Status]).
			Bold(true).
			Render(badgeText)
		// 行構成: prefix(2) + idCol(5) + 半角空白(1) + statusBadge + 半角空白(2) + title
		// パネル幅を超えて折り返さないよう、title 側を表示幅で切り詰める。
		titleSpace := innerW - 2 - 5 - 1 - runewidth.StringWidth(badgeText) - 2
		if titleSpace < 1 {
			titleSpace = 1
		}
		displayTitle := truncateDisplay(iss.Title, titleSpace)
		raw := fmt.Sprintf("%-5s %s  %s", idCol, statusBadge, displayTitle)
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
	header := panelHeaderStyle.Render("PREVIEW") + "\n"
	headerLines := strings.Count(header, "\n")

	iss := m.currentIssue()
	if iss == nil {
		body := header + hintStyle.Render("(no selection)")
		return panelStyle.Width(w).Render(body)
	}

	detail := RenderDetail(iss, m.parentOf(iss), m.childrenOf(iss.ID))
	lines := strings.Split(detail, "\n")
	// Remove trailing empty line from Split if detail ends with "\n"
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	maxVisible := m.panelInnerHeight() - headerLines
	if maxVisible < 1 {
		maxVisible = 1
	}

	totalLines := len(lines)
	offset := m.previewOffset

	// Clamp offset
	maxOffset := totalLines - maxVisible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}

	hasAbove := offset > 0
	hasBelow := offset+maxVisible < totalLines

	// Reserve lines for indicators
	contentLines := maxVisible
	if hasAbove {
		contentLines--
	}
	if hasBelow {
		contentLines--
	}
	if contentLines < 1 {
		contentLines = 1
	}

	end := offset + contentLines
	if end > totalLines {
		end = totalLines
	}

	var b strings.Builder
	b.WriteString(header)

	if hasAbove {
		b.WriteString(hintStyle.Render("▲") + "\n")
	}

	for i := offset; i < end; i++ {
		b.WriteString(lines[i])
		b.WriteString("\n")
	}

	if hasBelow {
		b.WriteString(hintStyle.Render("▼"))
	}

	return panelStyle.Width(w).Render(b.String())
}

// truncateDisplay は表示幅 max を超える文字列を末尾 "..." 付きで切り詰める。
// max <= 3 のときは ellipsis を入れる余裕がないので素朴に切り詰める。
func truncateDisplay(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= max {
		return s
	}
	if max <= 3 {
		return runewidth.Truncate(s, max, "")
	}
	return runewidth.Truncate(s, max, "...")
}

func (m listModel) parentOf(iss *model.Issue) *model.Issue {
	if iss.Parent == nil {
		return nil
	}
	for i := range m.issues {
		if m.issues[i].ID == *iss.Parent {
			return &m.issues[i]
		}
	}
	return nil
}

func (m listModel) childrenOf(id int) []model.Issue {
	var out []model.Issue
	for i := range m.issues {
		if m.issues[i].Parent != nil && *m.issues[i].Parent == id {
			out = append(out, m.issues[i])
		}
	}
	return out
}

func (m listModel) renderFooter() string {
	if m.filtering {
		return footerStyle.Render("enter accept  ·  esc clear")
	}
	keys := []string{
		"enter show",
		"c checkout",
		"n new",
		"e edit",
		"s status",
		"d delete",
		"v preview",
		"/ filter",
		"q quit",
	}
	if m.showPreview && m.width >= listMinWidthForPreview {
		keys = append(keys[:len(keys)-1], "J/K scroll", keys[len(keys)-1])
	}
	line := strings.Join(keys, "  ·  ")
	// On a terminal too narrow to hold every key hint, fall back to a compact
	// set so the footer stays a single line instead of wrapping.
	if m.width > 0 && runewidth.StringWidth(line) > m.width {
		line = strings.Join([]string{"j/k move", "enter show", "n new", "/ filter", "q quit"}, "  ·  ")
		if runewidth.StringWidth(line) > m.width {
			line = truncateDisplay(line, m.width)
		}
	}
	return footerStyle.Render(line)
}
