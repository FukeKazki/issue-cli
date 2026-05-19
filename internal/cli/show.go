package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

// Show prints a single issue. Without `--format` it emits the same
// lipgloss-styled plain text used by the TUI preview; with `--format` it
// renders the issue as JSON / YAML / Markdown for piping into other tools.
//
// Returns an error (causing a non-zero exit in main) when the id is missing,
// malformed, or unknown — required by the `issue show` acceptance criterion.
//
// Argument shape: `issue show <id> [--format ...]`. The id must be the first
// positional argument so the same code path serves `issue <id>` and
// `issue #<id>` shortcuts; flags after the id are parsed against args[1:],
// matching the convention used by `issue edit`.
func Show(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue show <id> [--format markdown|yaml|json]")
	}
	raw := strings.TrimPrefix(args[0], "#")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("invalid id %q: %v", args[0], err)
	}

	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	formatFlag := fs.String("format", "", "output format (markdown|yaml|json); omit for plain text")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	if *formatFlag == "" {
		fmt.Print(tui.RenderDetail(iss))
		return nil
	}
	f, err := output.ParseFormat(*formatFlag)
	if err != nil {
		return err
	}
	return output.WriteIssue(os.Stdout, iss, f)
}
