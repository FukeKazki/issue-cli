package cli

import (
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
	"github.com/FukeKazki/issue-cli/internal/tui"
)

func Create(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	title := fs.String("title", "", "title (skips TUI when set)")
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
	} else {
		if err := tui.RunForm(iss, "Create Issue"); err != nil {
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
