package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/gitx"
	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

func List(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	all := fs.Bool("all", false, "include Done issues")
	statusFilter := fs.String("status", "", "filter by status (TODO|In Progress|Reviews|Done)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
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
		case tui.ListActionCheckout:
			if res.IssueID == 0 {
				return nil
			}
			return gitx.CheckoutIssue(res.IssueID)
		case tui.ListActionEdit:
			if err := editIssue(s, res.IssueID); err != nil {
				fmt.Fprintln(os.Stderr, "edit failed:", err)
			}
		case tui.ListActionCreate:
			if id, err := createFromList(s); err != nil {
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

func editIssue(s *store.Store, id int) error {
	iss, err := s.Load(id)
	if err != nil {
		return err
	}
	if err := tui.RunForm(iss, "Edit Issue"); err != nil {
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

func createFromList(s *store.Store) (int, error) {
	id, err := s.NextID()
	if err != nil {
		return 0, err
	}
	iss := &model.Issue{ID: id, Status: model.StatusTODO}
	if err := tui.RunForm(iss, "Create Issue"); err != nil {
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
