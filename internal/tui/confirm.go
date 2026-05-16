package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type confirmModel struct {
	header    string
	yes       bool
	confirmed bool
}

func RunConfirm(header string) (bool, error) {
	m := confirmModel{header: header}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return false, err
	}
	fm := final.(confirmModel)
	return fm.confirmed && fm.yes, nil
}

func (m confirmModel) Init() tea.Cmd { return nil }

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "ctrl+c", "esc", "q":
		return m, tea.Quit
	case "left", "h":
		m.yes = false
	case "right", "l":
		m.yes = true
	case "tab":
		m.yes = !m.yes
	case "y", "Y":
		m.yes = true
		m.confirmed = true
		return m, tea.Quit
	case "n", "N":
		m.yes = false
		m.confirmed = true
		return m, tea.Quit
	case "enter":
		m.confirmed = true
		return m, tea.Quit
	}
	return m, nil
}

func (m confirmModel) View() string {
	var b strings.Builder
	b.WriteString(panelHeaderStyle.Render(m.header))
	b.WriteString("\n\n")

	selected := lipgloss.NewStyle().
		Foreground(colAccent).
		Bold(true).
		Background(lipgloss.Color("236")).
		Padding(0, 2)
	unselected := lipgloss.NewStyle().Foreground(colMuted).Padding(0, 2)

	noBtn := unselected.Render("No")
	yesBtn := unselected.Render("Yes")
	if m.yes {
		yesBtn = selected.Render("Yes")
	} else {
		noBtn = selected.Render("No")
	}

	b.WriteString(noBtn + "   " + yesBtn)
	b.WriteString("\n\n")
	b.WriteString(footerStyle.Render("←/→ choose  ·  y/n quick  ·  enter confirm  ·  esc cancel"))
	return panelStyle.Render(b.String())
}
