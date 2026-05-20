package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// IssueCandidate is the minimal shape RunForm needs to drive the blocked_by
// completion popup. Passed in by the cli layer so internal/tui keeps zero
// dependency on internal/store.
type IssueCandidate struct {
	ID    int
	Title string
}

// ErrCanceled is returned by RunForm when the user aborts with Esc/Ctrl+C.
var ErrCanceled = errors.New("canceled")

const (
	focusTitle = iota
	focusStatus
	focusType
	focusDescription
	focusRefs
	focusScope
	focusBlockedBy
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
	header                string
	iss                   *model.Issue
	titleInput            textinput.Model
	descArea              textarea.Model
	refsArea              textarea.Model
	scopeArea             textarea.Model
	blockedByArea         textarea.Model
	statuses              []model.Status
	statusIdx             int
	types                 []model.Type
	typeIdx               int
	focus                 int
	width                 int
	height                int
	submitted             bool
	canceled              bool
	errMsg                string
	repoFiles             []string
	scopeCompletion       completionState
	candidates            []IssueCandidate
	blockedByCompletion   completionState
	blockedByCandidates   []IssueCandidate
	confirmOnCancel       bool
	awaitingCancelConfirm bool
}

func newFormModel(iss *model.Issue, header string, candidates []IssueCandidate) formModel {
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
	fitDescHeight(&desc)

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

	blockedBy := textarea.New()
	blockedBy.Placeholder = "1"
	blockedBy.Prompt = "│ "
	blockedBy.ShowLineNumbers = false
	blockedBy.SetValue(joinIssueIDs(iss.BlockedBy))
	blockedBy.SetWidth(defaultFormWidth - 2)
	blockedBy.SetHeight(textareaHeight)
	blockedBy.CharLimit = 0
	blockedBy.Blur()

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

	// types[0] is the empty "(none)" sentinel so that an Issue without a Type
	// (existing on-disk issues created before Type was introduced) round-trips
	// through the form unchanged. Selecting it from the picker writes "" back
	// onto Issue.Type.
	types := append([]model.Type{""}, model.AllTypes()...)
	tIdx := 0
	for i, tp := range types {
		if tp == iss.Type {
			tIdx = i
			break
		}
	}

	return formModel{
		header:        header,
		iss:           iss,
		titleInput:    ti,
		descArea:      desc,
		refsArea:      refs,
		scopeArea:     scope,
		blockedByArea: blockedBy,
		statuses:      statuses,
		statusIdx:     idx,
		types:         types,
		typeIdx:       tIdx,
		focus:         focusTitle,
		repoFiles:     listRepoFiles(),
		candidates:    candidates,
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
		m.blockedByArea.SetWidth(fw)
		fitDescHeight(&m.descArea)
		return m, nil

	case tea.KeyMsg:
		if m.awaitingCancelConfirm {
			switch msg.String() {
			case "ctrl+c":
				m.canceled = true
				return m, tea.Quit
			case "y", "Y", "enter":
				m.canceled = true
				return m, tea.Quit
			case "n", "N", "esc":
				m.awaitingCancelConfirm = false
				return m, nil
			}
			return m, nil
		}
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
		if m.focus == focusBlockedBy && m.blockedByCompletion.active {
			switch msg.String() {
			case "esc":
				m.blockedByCompletion.active = false
				return m, nil
			case "up", "ctrl+p":
				if n := len(m.blockedByCompletion.items); n > 0 {
					m.blockedByCompletion.idx = (m.blockedByCompletion.idx - 1 + n) % n
				}
				return m, nil
			case "down", "ctrl+n":
				if n := len(m.blockedByCompletion.items); n > 0 {
					m.blockedByCompletion.idx = (m.blockedByCompletion.idx + 1) % n
				}
				return m, nil
			case "tab", "enter":
				if len(m.blockedByCompletion.items) > 0 {
					m.applyBlockedByCompletion()
				}
				m.blockedByCompletion.active = false
				return m, nil
			}
		}
		switch msg.String() {
		case "ctrl+c":
			m.canceled = true
			return m, tea.Quit
		case "esc":
			if m.confirmOnCancel && m.isDirty() {
				m.awaitingCancelConfirm = true
				return m, nil
			}
			m.canceled = true
			return m, tea.Quit
		case "ctrl+s":
			if strings.TrimSpace(m.titleInput.Value()) == "" {
				m.errMsg = "title is required"
				m.focus = focusTitle
				m.applyFocus()
				return m, nil
			}
			parsed, err := parseIssueIDs(m.blockedByArea.Value())
			if err != nil {
				m.errMsg = err.Error()
				m.focus = focusBlockedBy
				m.applyFocus()
				return m, nil
			}
			for _, id := range parsed {
				if id == m.iss.ID {
					m.errMsg = "blocked by cannot reference self"
					m.focus = focusBlockedBy
					m.applyFocus()
					return m, nil
				}
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
			if m.focus == focusTitle || m.focus == focusStatus || m.focus == focusType {
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

		if m.focus == focusType {
			switch msg.String() {
			case "left", "h", "up", "k":
				m.typeIdx = (m.typeIdx - 1 + len(m.types)) % len(m.types)
			case "right", "l", "down", "j", " ":
				m.typeIdx = (m.typeIdx + 1) % len(m.types)
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
		fitDescHeight(&m.descArea)
	case focusRefs:
		m.refsArea, cmd = m.refsArea.Update(msg)
	case focusScope:
		m.scopeArea, cmd = m.scopeArea.Update(msg)
		m.recomputeScopeCompletion()
	case focusBlockedBy:
		m.blockedByArea, cmd = m.blockedByArea.Update(msg)
		m.recomputeBlockedByCompletion()
	}
	return m, cmd
}

// fitDescHeight grows the description textarea to fit its current content,
// so long descriptions are fully visible instead of scrolling inside a
// fixed-height viewport. The height never drops below descAreaHeight.
func fitDescHeight(ta *textarea.Model) {
	rows := wrappedLineCount(ta.Value(), ta.Width())
	if rows < descAreaHeight {
		rows = descAreaHeight
	}
	ta.SetHeight(rows)
}

func wrappedLineCount(s string, width int) int {
	if width <= 0 {
		return 1
	}
	rows := 0
	for _, line := range strings.Split(s, "\n") {
		w := runewidth.StringWidth(line)
		if w == 0 {
			rows++
			continue
		}
		rows += (w + width - 1) / width
	}
	if rows < 1 {
		rows = 1
	}
	return rows
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
	if m.focus == focusBlockedBy {
		m.blockedByArea.Focus()
		m.recomputeBlockedByCompletion()
	} else {
		m.blockedByArea.Blur()
		m.blockedByCompletion.active = false
	}
}

func (m formModel) View() string {
	panel := m.renderFormPanel()
	if m.awaitingCancelConfirm {
		prompt := errorStyle.Render("Discard your changes?") + " " +
			footerStyle.Render("(y/Enter: discard, n/Esc: back)")
		return panel + "\n\n" + prompt
	}
	return panel + "\n" + m.renderFooter()
}

// isDirty reports whether the form's current values differ from the initial
// issue. Comparisons mirror the normalization in RunForm so that whitespace-
// only edits do not count as changes.
func (m formModel) isDirty() bool {
	if strings.TrimSpace(m.titleInput.Value()) != m.iss.Title {
		return true
	}
	if m.statuses[m.statusIdx] != m.iss.Status {
		return true
	}
	if m.types[m.typeIdx] != m.iss.Type {
		return true
	}
	if strings.TrimRight(m.descArea.Value(), "\n") != m.iss.Description {
		return true
	}
	if !stringSliceEqual(splitLines(m.refsArea.Value()), m.iss.References) {
		return true
	}
	if !stringSliceEqual(normalizeScope(splitLines(m.scopeArea.Value())), m.iss.Scope) {
		return true
	}
	if strings.TrimRight(m.blockedByArea.Value(), "\n") != joinIssueIDs(m.iss.BlockedBy) {
		return true
	}
	return false
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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

	b.WriteString(m.fieldLabel("TYPE", focusType))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("←/→ change"))
	b.WriteString("\n")
	b.WriteString(m.renderTypeRow())
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
	b.WriteString("\n\n")

	b.WriteString(m.fieldLabel("BLOCKED BY", focusBlockedBy))
	b.WriteString("  ")
	b.WriteString(hintStyle.Render("one issue id per line; type to search"))
	b.WriteString("\n")
	b.WriteString(m.blockedByArea.View())
	if m.focus == focusBlockedBy && m.blockedByCompletion.active {
		b.WriteString("\n")
		b.WriteString(m.renderBlockedByCompletion())
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

// renderTypeRow mirrors renderStatusRow but draws the optional Type picker.
// The first entry is the empty "(none)" sentinel so the user can clear the
// field; the remaining entries are the canonical model.AllTypes() values.
func (m formModel) renderTypeRow() string {
	parts := make([]string, 0, len(m.types))
	for i, tp := range m.types {
		text := string(tp)
		if text == "" {
			text = "(none)"
		}
		if i == m.typeIdx {
			style := lipgloss.NewStyle().
				Foreground(colAccent).
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

func RunForm(iss *model.Issue, header string, confirmOnCancel bool, candidates []IssueCandidate) error {
	m := newFormModel(iss, header, candidates)
	m.confirmOnCancel = confirmOnCancel
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
	iss.Type = fm.types[fm.typeIdx]
	iss.Description = strings.TrimRight(fm.descArea.Value(), "\n")
	iss.References = splitLines(fm.refsArea.Value())
	iss.Scope = normalizeScope(splitLines(fm.scopeArea.Value()))
	// blockedByArea was validated by the ctrl+s handler before submission.
	ids, _ := parseIssueIDs(fm.blockedByArea.Value())
	iss.BlockedBy = ids
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

// joinIssueIDs renders []int as one decimal id per line (no `#` prefix).
// Empty / nil input returns an empty string so SetValue produces a blank field.
func joinIssueIDs(ids []int) string {
	if len(ids) == 0 {
		return ""
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}
	return strings.Join(parts, "\n")
}

// parseIssueIDs parses the blockedByArea raw value into a slice of issue IDs.
// One id per non-blank line; an optional leading "#" is tolerated. Any token
// that fails strconv.Atoi or is <= 0 yields an error. Returns nil (not []int{})
// on empty input so callers can distinguish "no ids" from a zero-length slice
// without extra checks — YAML marshaling treats both as `blocked_by: []`.
func parseIssueIDs(raw string) ([]int, error) {
	var out []int
	for _, line := range strings.Split(raw, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			continue
		}
		s = strings.TrimPrefix(s, "#")
		s = strings.TrimSpace(s)
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("blocked by: %q is not a valid issue id", line)
		}
		if n <= 0 {
			return nil, fmt.Errorf("blocked by: id must be positive (got %d)", n)
		}
		out = append(out, n)
	}
	return out, nil
}

// filterIssueCandidates ranks candidates for the blocked_by completion popup.
// Rank 0: id prefix match (e.g. query "1" matches #1, #10, #11). Rank 1: case-
// insensitive title substring match. Ties break by id ascending. Up to 50
// items are returned. `excludeID` removes the candidate matching the current
// issue (self-block guard).
func filterIssueCandidates(cands []IssueCandidate, query string, excludeID int) []IssueCandidate {
	if len(cands) == 0 {
		return nil
	}
	const limit = 50
	q := strings.TrimSpace(query)
	if q == "" {
		out := make([]IssueCandidate, 0, len(cands))
		for _, c := range cands {
			if c.ID == excludeID {
				continue
			}
			out = append(out, c)
			if len(out) >= limit {
				break
			}
		}
		return out
	}
	q = strings.TrimPrefix(q, "#")
	lq := strings.ToLower(q)
	type scored struct {
		cand IssueCandidate
		rank int
	}
	var hits []scored
	for _, c := range cands {
		if c.ID == excludeID {
			continue
		}
		idStr := strconv.Itoa(c.ID)
		switch {
		case strings.HasPrefix(idStr, q):
			hits = append(hits, scored{c, 0})
		case strings.Contains(strings.ToLower(c.Title), lq):
			hits = append(hits, scored{c, 1})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].rank != hits[j].rank {
			return hits[i].rank < hits[j].rank
		}
		return hits[i].cand.ID < hits[j].cand.ID
	})
	if len(hits) == 0 {
		return nil
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]IssueCandidate, len(hits))
	for i, h := range hits {
		out[i] = h.cand
	}
	return out
}

// recomputeBlockedByCompletion mirrors the scope variant but uses no prefix
// trigger — the field is dedicated to ids. We *do* require the current row to
// have at least one non-whitespace character before activating the popup, so
// that pressing Enter on an empty row still inserts a newline (the popup's
// Enter handler would otherwise swallow it and the user could never reach a
// second id). On a non-empty row the popup is always shown.
func (m *formModel) recomputeBlockedByCompletion() {
	val := m.blockedByArea.Value()
	rows := strings.Split(val, "\n")
	rowIdx := m.blockedByArea.Line()
	if rowIdx < 0 || rowIdx >= len(rows) {
		m.blockedByCompletion.active = false
		return
	}
	row := []rune(rows[rowIdx])
	query := strings.TrimSpace(string(row))
	if query == "" {
		m.blockedByCompletion.active = false
		return
	}
	query = strings.TrimPrefix(query, "#")
	cands := filterIssueCandidates(m.candidates, query, m.iss.ID)
	display := make([]string, len(cands))
	for i, c := range cands {
		display[i] = fmt.Sprintf("#%d  %s", c.ID, c.Title)
	}
	prevActive := m.blockedByCompletion.active
	m.blockedByCompletion.active = true
	m.blockedByCompletion.row = rowIdx
	m.blockedByCompletion.atCol = 0
	m.blockedByCompletion.queryEnd = len(row)
	m.blockedByCompletion.items = display
	// store ids alongside displayed labels: keep the candidates in a parallel
	// slice so applyBlockedByCompletion can look up the id by the current idx.
	m.blockedByCandidates = cands
	if !prevActive || m.blockedByCompletion.idx >= len(display) {
		m.blockedByCompletion.idx = 0
	}
}

// applyBlockedByCompletion replaces the current line with the selected
// candidate's decimal id (no `#` prefix in the buffer — parseIssueIDs accepts
// either form, but the simpler form keeps the stored YAML clean).
func (m *formModel) applyBlockedByCompletion() {
	if m.blockedByCompletion.idx < 0 || m.blockedByCompletion.idx >= len(m.blockedByCandidates) {
		return
	}
	sel := m.blockedByCandidates[m.blockedByCompletion.idx]
	// Delete the current row contents (queryEnd was set to len(row)).
	for i := 0; i < m.blockedByCompletion.queryEnd; i++ {
		m.blockedByArea, _ = m.blockedByArea.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	m.blockedByArea.InsertString(strconv.Itoa(sel.ID))
}

func (m formModel) renderBlockedByCompletion() string {
	items := m.blockedByCompletion.items
	box := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(colMuted).
		Padding(0, 1)
	if len(items) == 0 {
		return box.Render(hintStyle.Render("no issues match"))
	}
	start, end := windowAround(m.blockedByCompletion.idx, len(items), completionMaxItems)
	var lines []string
	for i := start; i < end; i++ {
		marker := "  "
		text := items[i]
		if i == m.blockedByCompletion.idx {
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
