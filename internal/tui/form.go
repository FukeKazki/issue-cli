package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"unicode"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCanceled is returned by RunForm when the user aborts with Esc/Ctrl+C.
var ErrCanceled = errors.New("canceled")

const (
	focusTitle = iota
	focusStatus
	focusDescription
	focusRefs
	focusScope
	focusCount
)

const (
	defaultFormWidth = 56
	textareaHeight   = 4
	descAreaHeight   = 6
)

var (
	colAccent = lipgloss.Color("13")
	colTitle  = lipgloss.Color("12")
	colMuted  = lipgloss.Color("240")
	colError  = lipgloss.Color("9")

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colMuted).
			Padding(0, 1)

	panelHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colTitle).
				MarginBottom(1)

	labelBlur = lipgloss.NewStyle().
			Bold(true).
			Foreground(colMuted)

	labelFocus = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent)

	hintStyle = lipgloss.NewStyle().Faint(true)

	footerStyle = lipgloss.NewStyle().Faint(true)

	errorStyle = lipgloss.NewStyle().Foreground(colError).Bold(true)
)

type completionState struct {
	active   bool
	items    []string
	idx      int
	row      int
	atCol    int
	queryEnd int
}

type formModel struct {
	header          string
	iss             *model.Issue
	titleInput      textinput.Model
	descArea        textarea.Model
	refsArea        textarea.Model
	scopeArea       textarea.Model
	statuses        []model.Status
	statusIdx       int
	focus           int
	width           int
	height          int
	submitted       bool
	canceled        bool
	errMsg          string
	repoFiles       []string
	scopeCompletion completionState
}

func newFormModel(iss *model.Issue, header string) formModel {
	ti := textinput.New()
	ti.Placeholder = "Concise title"
	ti.CharLimit = 200
	ti.Prompt = "▏ "
	ti.Width = defaultFormWidth - 6
	ti.SetValue(iss.Title)
	ti.Focus()

	desc := textarea.New()
	desc.Placeholder = "What needs to happen and why?"
	desc.Prompt = "│ "
	desc.ShowLineNumbers = false
	desc.SetValue(iss.Description)
	desc.SetWidth(defaultFormWidth - 2)
	desc.SetHeight(descAreaHeight)
	desc.CharLimit = 0
	desc.Blur()

	refs := textarea.New()
	refs.Placeholder = "https://example.com"
	refs.Prompt = "│ "
	refs.ShowLineNumbers = false
	refs.SetValue(strings.Join(iss.References, "\n"))
	refs.SetWidth(defaultFormWidth - 2)
	refs.SetHeight(textareaHeight)
	refs.CharLimit = 0
	refs.Blur()

	scope := textarea.New()
	scope.Placeholder = "@apps/web/foo.tsx"
	scope.Prompt = "│ "
	scope.ShowLineNumbers = false
	scope.SetValue(strings.Join(iss.Scope, "\n"))
	scope.SetWidth(defaultFormWidth - 2)
	scope.SetHeight(textareaHeight)
	scope.CharLimit = 0
	scope.Blur()

	statuses := model.AllStatuses()
	idx := 0
	if iss.Status != "" {
		for i, s := range statuses {
			if s == iss.Status {
				idx = i
				break
			}
		}
	}

	return formModel{
		header:     header,
		iss:        iss,
		titleInput: ti,
		descArea:   desc,
		refsArea:   refs,
		scopeArea:  scope,
		statuses:   statuses,
		statusIdx:  idx,
		focus:      focusTitle,
		repoFiles:  listRepoFiles(),
	}
}

func (m formModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m formModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		fw := m.formColWidth() - 4 // border + padding
		if fw < 20 {
			fw = 20
		}
		m.titleInput.Width = fw - 2
		m.descArea.SetWidth(fw)
		m.refsArea.SetWidth(fw)
		m.scopeArea.SetWidth(fw)
		return m, nil

	case tea.KeyMsg:
		if m.focus == focusScope && m.scopeCompletion.active {
			switch msg.String() {
			case "esc":
				m.scopeCompletion.active = false
				return m, nil
			case "up", "ctrl+p":
				if n := len(m.scopeCompletion.items); n > 0 {
					m.scopeCompletion.idx = (m.scopeCompletion.idx - 1 + n) % n
				}
				return m, nil
			case "down", "ctrl+n":
				if n := len(m.scopeCompletion.items); n > 0 {
					m.scopeCompletion.idx = (m.scopeCompletion.idx + 1) % n
				}
				return m, nil
			case "tab", "enter":
				if len(m.scopeCompletion.items) > 0 {
					m.applyScopeCompletion()
				}
				m.scopeCompletion.active = false
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "ctrl+s":
			if strings.TrimSpace(m.titleInput.Value()) == "" {
				m.errMsg = "title is required"
				m.focus = focusTitle
				m.applyFocus()
				return m, nil
			}
			m.submitted = true
			return m, tea.Quit
		case "tab":
			m.focus = (m.focus + 1) % focusCount
			m.applyFocus()
			return m, nil
		case "shift+tab":
			m.focus = (m.focus - 1 + focusCount) % focusCount
			m.applyFocus()
			return m, nil
		case "enter":
			if m.focus == focusTitle || m.focus == focusStatus {
				m.focus = (m.focus + 1) % focusCount
				m.applyFocus()
				return m, nil
			}
		}

		if m.focus == focusStatus {
			switch msg.String() {
			case "left", "h", "up", "k":
				m.statusIdx = (m.statusIdx - 1 + len(m.statuses)) % len(m.statuses)
			case "right", "l", "down", "j", " ":
				m.statusIdx = (m.statusIdx + 1) % len(m.statuses)
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	switch m.focus {
	case focusTitle:
		m.titleInput, cmd = m.titleInput.Update(msg)
	case focusDescription:
		m.descArea, cmd = m.descArea.Update(msg)
	case focusRefs:
		m.refsArea, cmd = m.refsArea.Update(msg)
	case focusScope:
		m.scopeArea, cmd = m.scopeArea.Update(msg)
		m.recomputeScopeCompletion()
	}
	return m, cmd
}

func (m *formModel) applyFocus() {
	m.errMsg = ""
	if m.focus == focusTitle {
		m.titleInput.Focus()
	} else {
		m.titleInput.Blur()
	}
	if m.focus == focusDescription {
		m.descArea.Focus()
	} else {
		m.descArea.Blur()
	}
	if m.focus == focusRefs {
		m.refsArea.Focus()
	} else {
		m.refsArea.Blur()
	}
	if m.focus == focusScope {
		m.scopeArea.Focus()
	} else {
		m.scopeArea.Blur()
		m.scopeCompletion.active = false
	}
}

func (m formModel) View() string {
	return m.renderFormPanel() + "\n" + m.renderFooter()
}

func (m formModel) formColWidth() int {
	if m.width <= 0 {
		return defaultFormWidth
	}
	w := m.width - 2
	if w < defaultFormWidth {
		w = defaultFormWidth
	}
	return w
}

func (m formModel) renderFormPanel() string {
	header := m.header
	if header == "" {
		header = "Issue"
	}
	if m.iss.ID > 0 {
		header = fmt.Sprintf("%s  #%d", header, m.iss.ID)
	}

	var b strings.Builder
	b.WriteString(panelHeaderStyle.Render(header))
	b.WriteString("\n")

	b.WriteString(m.fieldLabel("TITLE", focusTitle))
	b.WriteString("\n")
	b.WriteString(m.titleInput.View())
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("STATUS", focusStatus))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("←/→ change"))
	b.WriteString("\n")
	b.WriteString(m.renderStatusRow())
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("DESCRIPTION", focusDescription))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("multi-line"))
	b.WriteString("\n")
	b.WriteString(m.descArea.View())
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("REFERENCES", focusRefs))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("one per line"))
	b.WriteString("\n")
	b.WriteString(m.refsArea.View())
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("SCOPE", focusScope))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("type @ to search files; one path per line"))
	b.WriteString("\n")
	b.WriteString(m.scopeArea.View())
	if m.focus == focusScope && m.scopeCompletion.active {
		b.WriteString("\n")
		b.WriteString(m.renderScopeCompletion())
	}

	if m.errMsg != "" {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("✗ " + m.errMsg))
	}

	return panelStyle.Width(m.formColWidth()).Render(b.String())
}

func (m formModel) fieldLabel(label string, target int) string {
	if m.focus == target {
		return labelFocus.Render("▸ " + label)
	}
	return labelBlur.Render("  " + label)
}

func (m formModel) renderStatusRow() string {
	parts := make([]string, 0, len(m.statuses))
	for i, s := range m.statuses {
		text := string(s)
		if i == m.statusIdx {
			style := lipgloss.NewStyle().
				Foreground(statusColor[s]).
				Bold(true).
				Background(lipgloss.Color("236")).
				Padding(0, 1)
			parts = append(parts, style.Render("● "+text))
		} else {
			style := lipgloss.NewStyle().
				Foreground(colMuted).
				Padding(0, 1)
			parts = append(parts, style.Render("○ "+text))
		}
	}
	return strings.Join(parts, " ")
}

func (m formModel) renderFooter() string {
	keys := []string{
		"tab/shift+tab move",
		"ctrl+s save",
		"esc cancel",
	}
	return footerStyle.Render(strings.Join(keys, "  •  "))
}

func RunForm(iss *model.Issue, header string) error {
	m := newFormModel(iss, header)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	fm := finalModel.(formModel)
	if fm.canceled || !fm.submitted {
		return ErrCanceled
	}
	iss.Title = strings.TrimSpace(fm.titleInput.Value())
	iss.Status = fm.statuses[fm.statusIdx]
	iss.Description = strings.TrimRight(fm.descArea.Value(), "\n")
	iss.References = splitLines(fm.refsArea.Value())
	iss.Scope = normalizeScope(splitLines(fm.scopeArea.Value()))
	return nil
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func normalizeScope(items []string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		if !strings.HasPrefix(s, "@") {
			s = "@" + s
		}
		out = append(out, s)
	}
	return out
}

const completionMaxItems = 8

func (m *formModel) recomputeScopeCompletion() {
	val := m.scopeArea.Value()
	rows := strings.Split(val, "\n")
	rowIdx := m.scopeArea.Line()
	if rowIdx < 0 || rowIdx >= len(rows) {
		m.scopeCompletion.active = false
		return
	}
	li := m.scopeArea.LineInfo()
	col := li.StartColumn + li.ColumnOffset
	row := []rune(rows[rowIdx])
	if col > len(row) {
		col = len(row)
	}
	atCol := -1
	for i := col - 1; i >= 0; i-- {
		r := row[i]
		if r == '@' {
			atCol = i
			break
		}
		if unicode.IsSpace(r) {
			break
		}
	}
	if atCol < 0 {
		m.scopeCompletion.active = false
		return
	}
	if atCol > 0 && !unicode.IsSpace(row[atCol-1]) {
		m.scopeCompletion.active = false
		return
	}
	query := string(row[atCol+1 : col])
	items := filterRepoFiles(m.repoFiles, query)
	prevActive := m.scopeCompletion.active
	m.scopeCompletion.active = true
	m.scopeCompletion.row = rowIdx
	m.scopeCompletion.atCol = atCol
	m.scopeCompletion.queryEnd = col
	m.scopeCompletion.items = items
	if !prevActive || m.scopeCompletion.idx >= len(items) {
		m.scopeCompletion.idx = 0
	}
}

func (m *formModel) applyScopeCompletion() {
	if m.scopeCompletion.idx < 0 || m.scopeCompletion.idx >= len(m.scopeCompletion.items) {
		return
	}
	sel := m.scopeCompletion.items[m.scopeCompletion.idx]
	n := m.scopeCompletion.queryEnd - m.scopeCompletion.atCol
	for i := 0; i < n; i++ {
		m.scopeArea, _ = m.scopeArea.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m.scopeArea.InsertString("@" + sel)
}

func (m formModel) renderScopeCompletion() string {
	items := m.scopeCompletion.items
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colMuted).
		Padding(0, 1)
	if len(items) == 0 {
		return box.Render(hintStyle.Render("no files match"))
	}
	start, end := windowAround(m.scopeCompletion.idx, len(items), completionMaxItems)
	var lines []string
	for i := start; i < end; i++ {
		marker := "  "
		text := items[i]
		if i == m.scopeCompletion.idx {
			marker = labelFocus.Render("▸ ")
			text = labelFocus.Render(text)
		} else {
			text = labelBlur.Render(text)
		}
		lines = append(lines, marker+text)
	}
	if end < len(items) {
		lines = append(lines, hintStyle.Render(fmt.Sprintf("  +%d more", len(items)-end)))
	}
	lines = append(lines, hintStyle.Render("↑/↓ select • tab/enter insert • esc dismiss"))
	return box.Render(strings.Join(lines, "\n"))
}

func windowAround(idx, total, size int) (int, int) {
	if total <= size {
		return 0, total
	}
	start := idx - size/2
	if start < 0 {
		start = 0
	}
	end := start + size
	if end > total {
		end = total
		start = end - size
	}
	return start, end
}

func listRepoFiles() []string {
	out, err := exec.Command("git", "ls-files", "-z").Output()
	if err != nil {
		return nil
	}
	parts := strings.Split(string(out), "\x00")
	files := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		files = append(files, p)
	}
	return files
}

func filterRepoFiles(files []string, query string) []string {
	if len(files) == 0 {
		return nil
	}
	limit := 50
	if query == "" {
		n := len(files)
		if n > limit {
			n = limit
		}
		out := make([]string, n)
		copy(out, files[:n])
		return out
	}
	q := strings.ToLower(query)
	type scored struct {
		path string
		rank int
	}
	var hits []scored
	for _, f := range files {
		lf := strings.ToLower(f)
		switch {
		case strings.HasPrefix(lf, q):
			hits = append(hits, scored{f, 0})
		case strings.Contains(lf, "/"+q):
			hits = append(hits, scored{f, 1})
		case strings.Contains(lf, q):
			hits = append(hits, scored{f, 2})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].path < hits[j].path
	})
	if len(hits) == 0 {
		return nil
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.path
	}
	return out
}
