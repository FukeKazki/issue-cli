package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

func Show(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue _show <id>")
	}
	raw := strings.TrimPrefix(args[0], "#")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("invalid id %q: %v", args[0], err)
	}
	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	fmt.Print(tui.RenderDetail(iss))
	return nil
}
