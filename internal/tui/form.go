package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/FukeKazki/issue-cli/internal/model"
)

func RunForm(iss *model.Issue, title string) error {
	refsText := strings.Join(iss.References, "\n")
	scopeText := strings.Join(iss.Scope, "\n")
	statusStr := string(iss.Status)
	if statusStr == "" {
		statusStr = string(model.StatusTODO)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&iss.Title).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return errors.New("title is required")
					}
					return nil
				}),
			huh.NewSelect[string]().
				Title("Status").
				Options(
					huh.NewOption(string(model.StatusTODO), string(model.StatusTODO)),
					huh.NewOption(string(model.StatusInProgress), string(model.StatusInProgress)),
					huh.NewOption(string(model.StatusReviews), string(model.StatusReviews)),
					huh.NewOption(string(model.StatusDone), string(model.StatusDone)),
				).
				Value(&statusStr),
			huh.NewText().
				Title("References (one per line)").
				Description("URLs or notes; blank lines ignored").
				Value(&refsText),
			huh.NewText().
				Title("Scope (one path per line)").
				Description("@ is auto-prepended if missing").
				Value(&scopeText),
		),
	).WithTheme(huh.ThemeBase()).WithShowHelp(true)

	if title != "" {
		form = form.WithTheme(huh.ThemeBase())
	}

	if err := form.Run(); err != nil {
		return err
	}

	iss.Title = strings.TrimSpace(iss.Title)
	if st, ok := model.ParseStatus(statusStr); ok {
		iss.Status = st
	}
	iss.References = splitLines(refsText)
	iss.Scope = normalizeScope(splitLines(scopeText))
	return nil
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func normalizeScope(items []string) []string {
	out := make([]string, 0, len(items))
	for _, s := range items {
		if !strings.HasPrefix(s, "@") {
			s = "@" + s
		}
		out = append(out, s)
	}
	return out
}
