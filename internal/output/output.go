// Package output renders Issue values to machine-readable formats (JSON, YAML,
// Markdown) for non-interactive subcommands. The CLI layer reaches into this
// package only when `--format` is supplied; human-facing TUI rendering stays in
// internal/tui.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/FukeKazki/issue-cli/internal/model"
	"gopkg.in/yaml.v3"
)

// Format is the value of `--format` on `show` / `list` / `next`.
type Format string

const (
	FormatJSON     Format = "json"
	FormatYAML     Format = "yaml"
	FormatMarkdown Format = "markdown"
)

// ParseFormat returns the canonical Format for a CLI value. Empty string is
// rejected; callers decide whether to default before calling.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	case "markdown", "md":
		return FormatMarkdown, nil
	}
	return "", fmt.Errorf("unknown format %q (expected json, yaml, markdown)", s)
}

// WriteIssue renders a single issue in the requested format. Used by
// `issue show <id> --format ...`.
func WriteIssue(w io.Writer, iss *model.Issue, f Format) error {
	switch f {
	case FormatJSON:
		return writeJSON(w, iss)
	case FormatYAML:
		return writeYAML(w, iss)
	case FormatMarkdown:
		return writeMarkdown(w, iss)
	}
	return fmt.Errorf("unsupported format %q", f)
}

// WriteIssues renders a slice in the requested format. Currently JSON only —
// `issue list --format` is documented as JSON-only. An empty slice still
// produces a valid JSON array (`[]`).
func WriteIssues(w io.Writer, issues []model.Issue, f Format) error {
	if f != FormatJSON {
		return fmt.Errorf("list does not support format %q (only json)", f)
	}
	if issues == nil {
		issues = []model.Issue{}
	}
	return writeJSON(w, issues)
}

// WriteNextIssue renders the `issue next` envelope. `nil` produces
// `{"issue": null}` so downstream pipes always receive valid JSON.
func WriteNextIssue(w io.Writer, iss *model.Issue, f Format) error {
	if f != FormatJSON {
		return fmt.Errorf("next does not support format %q (only json)", f)
	}
	return writeJSON(w, struct {
		Issue *model.Issue `json:"issue"`
	}{Issue: iss})
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeYAML(w io.Writer, v any) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return err
	}
	return enc.Close()
}

func writeMarkdown(w io.Writer, iss *model.Issue) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# #%d  %s\n\n", iss.ID, iss.Title)
	fmt.Fprintf(&b, "**Status:** %s\n\n", iss.Status)

	fmt.Fprintln(&b, "## Description")
	if strings.TrimSpace(iss.Description) == "" {
		fmt.Fprintln(&b, "(none)")
	} else {
		fmt.Fprintln(&b, strings.TrimRight(iss.Description, "\n"))
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## References")
	if len(iss.References) == 0 {
		fmt.Fprintln(&b, "(none)")
	} else {
		for _, r := range iss.References {
			fmt.Fprintf(&b, "- %s\n", r)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintln(&b, "## Scope")
	if len(iss.Scope) == 0 {
		fmt.Fprintln(&b, "(none)")
	} else {
		for _, s := range iss.Scope {
			fmt.Fprintf(&b, "- %s\n", s)
		}
	}
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "Created: %s\n", fmtTime(iss.CreatedAt))
	fmt.Fprintf(&b, "Updated: %s\n", fmtTime(iss.UpdatedAt))

	_, err := io.WriteString(w, b.String())
	return err
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05 -0700")
}
