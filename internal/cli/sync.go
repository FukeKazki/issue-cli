package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/gh"
	"github.com/FukeKazki/issue-cli/internal/gitx"
	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
)

// prStatusForBranch is a seam tests swap to stub out the gh dependency.
var prStatusForBranch = gh.PRStatusForBranch

// Sync reconciles an issue's status with its GitHub PR state. With no args it
// targets the current `issue/<id>` branch; an explicit id (or `#id`) overrides.
func Sync(args []string) error {
	id, err := resolveSyncID(args)
	if err != nil {
		return err
	}
	s, err := store.New()
	if err != nil {
		return err
	}
	return syncIssue(s, id)
}

func syncIssue(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}

	branch := fmt.Sprintf("issue/%d", id)
	pr, err := prStatusForBranch(branch)
	if err != nil {
		return err
	}
	if !pr.Found {
		fmt.Printf("#%d: no PR found for %s (status %s, unchanged)\n", id, branch, iss.Status)
		return nil
	}

	target := iss.Status
	switch {
	case pr.State == "MERGED" || !pr.MergedAt.IsZero():
		target = model.StatusDone
	case pr.State == "OPEN":
		target = model.StatusReviews
	}

	from := iss.Status
	if !iss.AdvanceStatus(target) {
		fmt.Printf("#%d: status %s already up to date (PR #%d %s)\n", id, iss.Status, pr.Number, pr.State)
		return nil
	}
	if err := s.Save(iss); err != nil {
		return err
	}
	fmt.Printf("#%d: %s → %s (PR #%d %s)\n", id, from, iss.Status, pr.Number, pr.State)
	return nil
}

func resolveSyncID(args []string) (int, error) {
	if len(args) > 0 {
		raw := strings.TrimPrefix(args[0], "#")
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			return 0, fmt.Errorf("invalid issue id: %q", args[0])
		}
		return id, nil
	}
	id, err := gitx.CurrentIssueID()
	if err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, fmt.Errorf("not on an issue/<id> branch; pass an id explicitly")
	}
	return id, nil
}
