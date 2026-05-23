package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
)

// Edit updates fields of an existing issue from the CLI. Currently supports
// `--status` (case-insensitive, accepts canonical names plus common aliases).
// The status change is an explicit user action so it allows any direction
// (forward or backward), mirroring the TUI `s` key path in changeStatus.
func Edit(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue-cli edit <id> --status STATUS")
	}
	raw := strings.TrimPrefix(args[0], "#")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid issue id: %q", args[0])
	}

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	statusFlag := fs.String("status", "", "new status (TODO|In Progress|Reviews|Done; case-insensitive, aliases allowed)")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*statusFlag) == "" {
		return fmt.Errorf("--status is required")
	}
	target, ok := model.ParseStatusFromCLI(*statusFlag)
	if !ok {
		return fmt.Errorf("unknown status: %q (expected TODO, In Progress, Reviews, Done)", *statusFlag)
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	if iss.Status == target {
		fmt.Printf("#%d: status %s already up to date\n", id, iss.Status)
		return nil
	}
	from := iss.Status
	iss.Status = target
	if err := s.Save(iss); err != nil {
		return err
	}
	fmt.Printf("#%d: %s → %s\n", id, from, iss.Status)
	return nil
}
