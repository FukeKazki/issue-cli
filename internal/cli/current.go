package cli

import (
	"fmt"

	"github.com/FukeKazki/issue-cli/internal/gitx"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

// Current prints the issue detail for the issue branch the user is currently on
// (e.g. `issue/2` → issue #2). Returns an error when not on an issue branch.
func Current(args []string) error {
	id, err := gitx.CurrentIssueID()
	if err != nil {
		return err
	}
	if id == 0 {
		br, _ := gitx.CurrentBranch()
		if br == "" {
			return fmt.Errorf("not on an issue branch (detached HEAD)")
		}
		return fmt.Errorf("not on an issue branch (current: %s)", br)
	}
	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	fmt.Print(tui.RenderDetail(iss))
	return nil
}
