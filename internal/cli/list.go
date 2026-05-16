package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/fzf"
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

	if err := fzf.Available(); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	for {
		issues, err := s.LoadAll()
		if err != nil {
			return err
		}
		issues = filterIssues(issues, *all, *statusFilter)
		if len(issues) == 0 {
			fmt.Fprintln(os.Stderr, "no issues to show")
			return nil
		}

		lines := make([]string, 0, len(issues))
		for _, iss := range issues {
			lines = append(lines, fmt.Sprintf("#%d\t[%s]\t%s", iss.ID, iss.Status, iss.Title))
		}

		opts := []string{
			"--ansi",
			"--reverse",
			"--no-mouse",
			"--delimiter=\t",
			"--header=Enter: checkout / v: detail / e: edit / c: create / s: status / Esc: close preview / Ctrl-C: quit",
			"--preview", self + " _show {1}",
			"--preview-window=right:60%:hidden",
			"--bind=v:show-preview",
			"--bind=esc:hide-preview",
			"--expect=enter,e,c,s",
		}

		res, err := fzf.Run(lines, opts)
		if err != nil {
			return err
		}
		// fzf returned without a key press (Ctrl-C / empty): exit.
		if res.Key == "" && res.Line == "" {
			return nil
		}

		id := parseID(res.Line)

		switch res.Key {
		case "enter":
			if id == 0 {
				return nil
			}
			return gitx.CheckoutIssue(id)
		case "e":
			if id == 0 {
				continue
			}
			if err := editIssue(s, id); err != nil {
				fmt.Fprintln(os.Stderr, "edit failed:", err)
			}
		case "c":
			if err := createFromList(s); err != nil {
				fmt.Fprintln(os.Stderr, "create failed:", err)
			}
		case "s":
			if id == 0 {
				continue
			}
			if err := changeStatus(s, id); err != nil {
				fmt.Fprintln(os.Stderr, "status change failed:", err)
			}
		default:
			return nil
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

func parseID(line string) int {
	if line == "" {
		return 0
	}
	first := strings.SplitN(line, "\t", 2)[0]
	raw := strings.TrimPrefix(first, "#")
	id, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return id
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
	statuses := model.AllStatuses()
	lines := make([]string, 0, len(statuses))
	for i, st := range statuses {
		marker := " "
		if iss.Status == st {
			marker = "*"
		}
		lines = append(lines, fmt.Sprintf("%d %s %s", i+1, marker, st))
	}
	opts := []string{
		"--reverse",
		"--no-mouse",
		fmt.Sprintf("--header=Status for #%d (current marked *) — 1:TODO  2:In Progress  3:Reviews  4:Done  Esc:cancel", id),
		"--expect=1,2,3,4,enter",
	}
	res, err := fzf.Run(lines, opts)
	if err != nil {
		return err
	}
	var newStatus model.Status
	switch res.Key {
	case "1":
		newStatus = model.StatusTODO
	case "2":
		newStatus = model.StatusInProgress
	case "3":
		newStatus = model.StatusReviews
	case "4":
		newStatus = model.StatusDone
	case "enter":
		if res.Line == "" {
			return nil
		}
		idx, err := strconv.Atoi(strings.SplitN(res.Line, " ", 2)[0])
		if err != nil || idx < 1 || idx > len(statuses) {
			return nil
		}
		newStatus = statuses[idx-1]
	default:
		return nil
	}
	if iss.Status == newStatus {
		return nil
	}
	iss.Status = newStatus
	return s.Save(iss)
}

func createFromList(s *store.Store) error {
	id, err := s.NextID()
	if err != nil {
		return err
	}
	iss := &model.Issue{ID: id, Status: model.StatusTODO}
	if err := tui.RunForm(iss, "Create Issue"); err != nil {
		if errors.Is(err, tui.ErrCanceled) {
			return nil
		}
		return err
	}
	if strings.TrimSpace(iss.Title) == "" {
		return fmt.Errorf("title is required")
	}
	return s.Save(iss)
}
