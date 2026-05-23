package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

func New(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	title := fs.String("title", "", "title (skips TUI when set)")
	typeFlag := fs.String("type", "", "issue type (Bug|Feature|Enhancement|Docs|Refactor; case-insensitive)")
	description := fs.String("description", "", "description")
	scope := fs.String("scope", "", "comma-separated scope paths")
	references := fs.String("references", "", "comma-separated references")
	blockedBy := fs.String("blocked-by", "", "comma-separated issue IDs that block this issue")
	if err := fs.Parse(args); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	id, err := s.NextID()
	if err != nil {
		return err
	}
	iss := &model.Issue{ID: id, Status: model.StatusTODO}

	if *title != "" {
		iss.Title = strings.TrimSpace(*title)
		if iss.Title == "" {
			return errors.New("--title must not be empty")
		}
		if *typeFlag != "" {
			t, ok := model.ParseTypeFromCLI(*typeFlag)
			if !ok {
				return fmt.Errorf("unknown type: %q (expected Bug, Feature, Enhancement, Docs, Refactor)", *typeFlag)
			}
			iss.Type = t
		}
		if *description != "" {
			iss.Description = *description
		}
		if *scope != "" {
			iss.Scope = splitCSV(*scope)
		}
		if *references != "" {
			iss.References = splitCSV(*references)
		}
		if *blockedBy != "" {
			ids, err := parseIntCSV(*blockedBy)
			if err != nil {
				return fmt.Errorf("invalid --blocked-by: %w", err)
			}
			iss.BlockedBy = ids
		}
	} else {
		cands, err := loadIssueCandidates(s, iss.ID)
		if err != nil {
			return err
		}
		if err := tui.RunForm(iss, "New Issue", true, cands); err != nil {
			if errors.Is(err, tui.ErrCanceled) {
				fmt.Fprintln(os.Stderr, "canceled")
				return nil
			}
			return err
		}
		if strings.TrimSpace(iss.Title) == "" {
			return errors.New("title is required")
		}
	}

	if err := s.Save(iss); err != nil {
		return err
	}
	fmt.Printf("created #%d: %s\n", iss.ID, s.Path(iss.ID))
	return nil
}

// splitCSV splits a comma-separated string into trimmed, non-empty elements.
func splitCSV(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

// parseIntCSV splits a comma-separated string into positive integers.
func parseIntCSV(s string) ([]int, error) {
	var out []int
	for _, v := range strings.Split(s, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		v = strings.TrimPrefix(v, "#")
		n, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("not an integer: %q", v)
		}
		if n <= 0 {
			return nil, fmt.Errorf("issue ID must be positive: %d", n)
		}
		out = append(out, n)
	}
	return out, nil
}
