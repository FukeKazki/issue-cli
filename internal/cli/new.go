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

// newFlags holds the raw, unparsed values of `issue-cli new`'s flags. Parsing
// and validation happen in applyNewFlags so the flag→Issue mapping is testable
// without touching the store or the TUI.
type newFlags struct {
	title       string
	typeFlag    string
	description string
	scope       string
	references  string
	blockedBy   string
	parent      string
}

func New(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	var f newFlags
	fs.StringVar(&f.title, "title", "", "title (skips TUI when set)")
	fs.StringVar(&f.typeFlag, "type", "", "issue type (Bug|Feature|Enhancement|Docs|Refactor; case-insensitive)")
	fs.StringVar(&f.description, "description", "", "description")
	fs.StringVar(&f.scope, "scope", "", "comma-separated scope paths")
	fs.StringVar(&f.references, "references", "", "comma-separated references")
	fs.StringVar(&f.blockedBy, "blocked-by", "", "comma-separated issue IDs that block this issue")
	fs.StringVar(&f.parent, "parent", "", "parent issue ID")
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

	if f.title != "" {
		if err := applyNewFlags(iss, f); err != nil {
			return err
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

// applyNewFlags populates iss from the parsed CLI flags. The caller guarantees
// f.title is non-empty (it is the gate into the non-interactive path); a
// whitespace-only title is still rejected. Only flags with non-empty values
// touch their corresponding field, so omitted flags leave the zero value.
func applyNewFlags(iss *model.Issue, f newFlags) error {
	iss.Title = strings.TrimSpace(f.title)
	if iss.Title == "" {
		return errors.New("--title must not be empty")
	}
	if f.typeFlag != "" {
		t, ok := model.ParseTypeFromCLI(f.typeFlag)
		if !ok {
			return fmt.Errorf("unknown type: %q (expected Bug, Feature, Enhancement, Docs, Refactor)", f.typeFlag)
		}
		iss.Type = t
	}
	if f.description != "" {
		iss.Description = f.description
	}
	if f.scope != "" {
		iss.Scope = splitCSV(f.scope)
	}
	if f.references != "" {
		iss.References = splitCSV(f.references)
	}
	if f.blockedBy != "" {
		ids, err := parseIntCSV(f.blockedBy)
		if err != nil {
			return fmt.Errorf("invalid --blocked-by: %w", err)
		}
		iss.BlockedBy = ids
	}
	if f.parent != "" {
		pid, err := parseIDArg(f.parent)
		if err != nil {
			return fmt.Errorf("invalid --parent: %w", err)
		}
		iss.Parent = &pid
	}
	return nil
}

// parseIDArg parses a single issue ID, tolerating a leading '#'. Mirrors the
// per-element parsing in parseIntCSV but for a lone value.
func parseIDArg(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("not an integer: %q", s)
	}
	if n <= 0 {
		return 0, fmt.Errorf("issue ID must be positive: %d", n)
	}
	return n, nil
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
