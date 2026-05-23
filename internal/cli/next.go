package cli

import (
	"flag"
	"os"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
)

// Next returns the next actionable TODO issue for automation consumers.
//
// It is deterministic: store.LoadAll already sorts by ID ascending, so the
// lowest-numbered TODO wins. When no TODO exists the command still exits 0
// and emits `{"issue": null}`, keeping downstream pipes (e.g. simple-takt)
// happy with valid JSON.
func Next(args []string) error {
	fs := flag.NewFlagSet("next", flag.ContinueOnError)
	formatFlag := fs.String("format", "json", "output format (json)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	f, err := output.ParseFormat(*formatFlag)
	if err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	issues, err := s.LoadAll()
	if err != nil {
		return err
	}
	// Build a lookup so we can check blocker statuses in O(1).
	statusByID := make(map[int]model.Status, len(issues))
	for _, iss := range issues {
		statusByID[iss.ID] = iss.Status
	}

	var next *model.Issue
	for i := range issues {
		if issues[i].Status != model.StatusTODO {
			continue
		}
		// Skip if any blocker is not Done (or unknown / missing).
		blocked := false
		for _, bid := range issues[i].BlockedBy {
			st, ok := statusByID[bid]
			if !ok || st != model.StatusDone {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		next = &issues[i]
		break
	}
	return output.WriteNextIssue(os.Stdout, next, f)
}
