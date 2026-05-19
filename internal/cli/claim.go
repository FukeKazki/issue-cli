package cli

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
)

// Claim transitions a TODO issue to In Progress and records the workflow /
// run-id / start time on `Issue.Run`. It is the non-interactive entry point
// for automation runners (e.g. simple-takt) that need to atomically reserve
// an issue before working on it.
//
// Argument shape:
//
//	issue claim <id> [--workflow NAME] [--run-id ID] [--force] [--format json]
//
// Semantics:
//   - When the issue is already non-TODO and `--force` is absent, the command
//     fails without touching the YAML so concurrent runners surface the
//     conflict instead of clobbering state.
//   - `--force` overrides the guard (any status, including Done) — the use
//     case is a manual re-run or recovery after a crashed runner.
//   - The status transition uses direct assignment (not `AdvanceStatus`),
//     mirroring the explicit user-driven `edit --status` path, because
//     claim is an externally-triggered action and `--force` may need to
//     walk backwards from Reviews/Done.
//   - The previous `Run` block, if any, is replaced (the field reflects the
//     most recent execution, not a history).
func Claim(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue claim <id> [--workflow NAME] [--run-id ID] [--force] [--format json]")
	}
	raw := strings.TrimPrefix(args[0], "#")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid issue id: %q", args[0])
	}

	fs := flag.NewFlagSet("claim", flag.ContinueOnError)
	workflowFlag := fs.String("workflow", "", "workflow name to record on the run metadata")
	runIDFlag := fs.String("run-id", "", "run identifier to record on the run metadata")
	forceFlag := fs.Bool("force", false, "claim even when the issue is not TODO")
	formatFlag := fs.String("format", "", "output format (json); omit for plain text")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}

	if iss.Status != model.StatusTODO && !*forceFlag {
		return fmt.Errorf("issue #%d is already %s%s; pass --force to override",
			id, iss.Status, runHint(iss.Run))
	}

	prevStatus := iss.Status
	iss.Status = model.StatusInProgress
	iss.Run = &model.Run{
		Workflow:  strings.TrimSpace(*workflowFlag),
		ID:        strings.TrimSpace(*runIDFlag),
		StartedAt: time.Now(),
	}
	if err := s.Save(iss); err != nil {
		return err
	}

	if *formatFlag != "" {
		f, err := output.ParseFormat(*formatFlag)
		if err != nil {
			return err
		}
		return output.WriteIssue(os.Stdout, iss, f)
	}
	fmt.Printf("#%d: %s → %s%s\n", id, prevStatus, iss.Status, runHint(iss.Run))
	return nil
}

// runHint formats a parenthesized "(workflow=..., run-id=...)" suffix when
// either field is present. Returns an empty string when neither is set, so
// the surrounding sentence reads naturally.
func runHint(r *model.Run) string {
	if r == nil {
		return ""
	}
	var parts []string
	if r.Workflow != "" {
		parts = append(parts, "workflow="+r.Workflow)
	}
	if r.ID != "" {
		parts = append(parts, "run-id="+r.ID)
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, ", ") + ")"
}
