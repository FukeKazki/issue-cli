package cli

import (
	"fmt"

	"github.com/FukeKazki/issue-cli/internal/gitx"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

// Default is the no-arg entry point. On an `issue/<id>` branch it prints the
// matching issue's detail; otherwise it opens the list TUI.
func Default() error {
	id, err := gitx.CurrentIssueID()
	if err != nil {
		return err
	}
	if id == 0 {
		return List(nil)
	}
	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	parent, children, err := resolveRelatives(iss, s)
	if err != nil {
		return err
	}
	fmt.Print(tui.RenderDetail(iss, parent, children))
	return nil
}
