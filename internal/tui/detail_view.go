package tui

import (
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	tea "github.com/charmbracelet/bubbletea"
)

type detailModel struct {
	iss      *model.Issue
	parent   *model.Issue
	children []model.Issue
	width    int
	height   int
}

func RunDetailView(iss *model.Issue, parent *model.Issue, children []model.Issue) error {
	p := tea.NewProgram(detailModel{iss: iss, parent: parent, children: children}, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m detailModel) Init() tea.Cmd { return nil }

func (m detailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "enter", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m detailModel) View() string {
	body := strings.TrimRight(RenderDetail(m.iss, m.parent, m.children), "\n")
	return panelStyle.Render(body) + "\n" + footerStyle.Render("q/esc/enter back")
}
