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

// Release records the terminal result of a workflow execution on the
// issue's Run metadata. It is the counterpart to `Claim` and is intended
// for automation runners to call when an execution finishes (successfully,
// with a failure, or interrupted).
//
// Argument shape:
//
//	issue release <id> --result success|failure|interrupted \
//	                   [--error MSG] [--pr-url URL] [--format json]
//
// Semantics:
//   - `--result` is required and must parse via `model.ParseRunResult`.
//   - Status is intentionally NOT changed here — status transitions remain
//     the responsibility of `issue edit --status`. Release only stamps
//     finish-time / result / error / pr_url on the Run block.
//   - If the issue has no Run yet (e.g. a manual recording without a prior
//     claim), an empty Run is allocated and only FinishedAt / Result /
//     Error / PRURL are populated.
//   - Title, description, references, and scope are untouched — the
//     full Issue is round-tripped via `store.Save`.
func Release(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue release <id> --result success|failure|interrupted [--error MSG] [--pr-url URL] [--format json]")
	}
	raw := strings.TrimPrefix(args[0], "#")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid issue id: %q", args[0])
	}

	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	resultFlag := fs.String("result", "", "execution result (success|failure|interrupted)")
	errorFlag := fs.String("error", "", "short error summary to attach when result is failure/interrupted")
	prURLFlag := fs.String("pr-url", "", "pull-request URL produced by the run")
	formatFlag := fs.String("format", "", "output format (json); omit for plain text")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*resultFlag) == "" {
		return fmt.Errorf("--result is required (success|failure|interrupted)")
	}
	result, ok := model.ParseRunResult(*resultFlag)
	if !ok {
		return fmt.Errorf("unknown result: %q (expected success, failure, interrupted)", *resultFlag)
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}

	if iss.Run == nil {
		iss.Run = &model.Run{}
	}
	iss.Run.FinishedAt = time.Now()
	iss.Run.Result = result
	if *errorFlag != "" {
		iss.Run.Error = *errorFlag
	}
	if *prURLFlag != "" {
		iss.Run.PRURL = *prURLFlag
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
	fmt.Printf("#%d: released (result=%s%s)\n", id, iss.Run.Result, runIDHint(iss.Run))
	return nil
}

// runIDHint formats ", run-id=<id>" when the Run.ID is present. It exists
// for the release plain-text output where the workflow name is already
// implicit from context; only the run-id is worth surfacing.
func runIDHint(r *model.Run) string {
	if r == nil || r.ID == "" {
		return ""
	}
	return ", run-id=" + r.ID
}
