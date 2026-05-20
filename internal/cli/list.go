package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/gitx"
	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

func List(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	all := fs.Bool("all", false, "include Done issues")
	statusFilter := fs.String("status", "", "filter by status (TODO|In Progress|Reviews|Done)")
	formatFlag := fs.String("format", "", "non-interactive output format (json); omit to open the TUI")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}

	if *formatFlag != "" {
		return runListFormatted(s, *all, *statusFilter, *formatFlag)
	}

	lastID := 0
	for {
		issues, err := s.LoadAll()
		if err != nil {
			return err
		}
		issues = filterIssues(issues, *all, *statusFilter)

		header := "Issues"
		if *statusFilter != "" {
			header = fmt.Sprintf("Issues — status=%s", *statusFilter)
		} else if *all {
			header = "Issues (all)"
		} else {
			header = "Issues (open)"
		}

		res, err := tui.RunList(issues, header, lastID)
		if err != nil {
			return err
		}
		lastID = res.IssueID

		switch res.Action {
		case tui.ListActionQuit:
			return nil
		case tui.ListActionShow:
			if err := showIssue(s, res.IssueID); err != nil {
				fmt.Fprintln(os.Stderr, "show failed:", err)
			}
		case tui.ListActionCheckout:
			if res.IssueID == 0 {
				return nil
			}
			if err := gitx.CheckoutIssue(res.IssueID); err != nil {
				return err
			}
			return advanceOnCheckout(s, res.IssueID)
		case tui.ListActionEdit:
			if err := editIssue(s, res.IssueID); err != nil {
				fmt.Fprintln(os.Stderr, "edit failed:", err)
			}
		case tui.ListActionNew:
			if id, err := newFromList(s); err != nil {
				fmt.Fprintln(os.Stderr, "create failed:", err)
			} else if id > 0 {
				lastID = id
			}
		case tui.ListActionStatus:
			if err := changeStatus(s, res.IssueID); err != nil {
				fmt.Fprintln(os.Stderr, "status change failed:", err)
			}
		case tui.ListActionDelete:
			deleted, err := deleteIssue(s, res.IssueID)
			if err != nil {
				fmt.Fprintln(os.Stderr, "delete failed:", err)
			} else if deleted {
				lastID = 0
			}
		}
	}
}

// runListFormatted is the non-interactive path taken when `--format` is set.
// It applies the same filters as the TUI path and renders the resulting
// slice via internal/output, then returns without ever opening the TUI loop.
func runListFormatted(s *store.Store, all bool, statusFilter, formatRaw string) error {
	f, err := output.ParseFormat(formatRaw)
	if err != nil {
		return err
	}
	issues, err := s.LoadAll()
	if err != nil {
		return err
	}
	issues = filterIssues(issues, all, statusFilter)
	return output.WriteIssues(os.Stdout, issues, f)
}

func filterIssues(in []model.Issue, all bool, statusFilter string) []model.Issue {
	out := make([]model.Issue, 0, len(in))
	for _, iss := range in {
		if statusFilter != "" {
			if string(iss.Status) != statusFilter {
				continue
			}
		} else if !all && !iss.IsOpen() {
			continue
		}
		out = append(out, iss)
	}
	return out
}

func showIssue(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	return tui.RunDetailView(iss)
}

func editIssue(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	cands, err := loadIssueCandidates(s, iss.ID)
	if err != nil {
		return err
	}
	if err := tui.RunForm(iss, "Edit Issue", false, cands); err != nil {
		if errors.Is(err, tui.ErrCanceled) {
			return nil
		}
		return err
	}
	return s.Save(iss)
}

func changeStatus(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	res, err := tui.RunStatusPicker(fmt.Sprintf("Status for #%d", id), iss.Status)
	if err != nil {
		return err
	}
	if !res.Selected || res.Status == iss.Status {
		return nil
	}
	iss.Status = res.Status
	return s.Save(iss)
}

func deleteIssue(s *store.Store, id int) (bool, error) {
	iss, err := s.Load(id)
	if err != nil {
		return false, err
	}
	ok, err := tui.RunConfirm(fmt.Sprintf("Delete #%d %q?", id, iss.Title))
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := s.Delete(id); err != nil {
		return false, err
	}
	return true, nil
}

// advanceOnCheckout bumps a freshly checked-out issue to In Progress if it's
// still TODO. Already-advanced issues are left alone (forward-only).
func advanceOnCheckout(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	from := iss.Status
	if !iss.AdvanceStatus(model.StatusInProgress) {
		return nil
	}
	if err := s.Save(iss); err != nil {
		return err
	}
	fmt.Printf("#%d: %s → %s\n", id, from, iss.Status)
	return nil
}

func newFromList(s *store.Store) (int, error) {
	id, err := s.NextID()
	if err != nil {
		return 0, err
	}
	iss := &model.Issue{ID: id, Status: model.StatusTODO}
	cands, err := loadIssueCandidates(s, iss.ID)
	if err != nil {
		return 0, err
	}
	if err := tui.RunForm(iss, "New Issue", true, cands); err != nil {
		if errors.Is(err, tui.ErrCanceled) {
			return 0, nil
		}
		return 0, err
	}
	if strings.TrimSpace(iss.Title) == "" {
		return 0, fmt.Errorf("title is required")
	}
	if err := s.Save(iss); err != nil {
		return 0, err
	}
	return id, nil
}

// loadIssueCandidates reads all issues from the store and returns the
// (id, title) pairs needed for the form's blocked_by completion popup.
// `excludeID` filters out the current issue so a user cannot pick themselves.
func loadIssueCandidates(s *store.Store, excludeID int) ([]tui.IssueCandidate, error) {
	issues, err := s.LoadAll()
	if err != nil {
		return nil, err
	}
	out := make([]tui.IssueCandidate, 0, len(issues))
	for _, iss := range issues {
		if iss.ID == excludeID {
			continue
		}
		out = append(out, tui.IssueCandidate{ID: iss.ID, Title: iss.Title})
	}
	return out, nil
}
