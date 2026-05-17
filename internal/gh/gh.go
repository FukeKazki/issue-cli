// Package gh is a thin wrapper around the `gh` CLI for read-only PR state.
package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// PRStatus describes the PR associated with a branch.
type PRStatus struct {
	Found    bool
	Number   int
	State    string // "OPEN" | "CLOSED" | "MERGED"
	MergedAt time.Time
}

// run is the seam tests swap to avoid shelling out to a real `gh` binary.
var run = func(name string, args ...string) ([]byte, error) {
	if _, err := exec.LookPath(name); err != nil {
		return nil, fmt.Errorf("%s not found in PATH: install GitHub CLI (https://cli.github.com)", name)
	}
	return exec.Command(name, args...).Output()
}

// PRStatusForBranch returns the newest PR whose head matches branch.
// Found is false when no PR exists.
func PRStatusForBranch(branch string) (PRStatus, error) {
	out, err := run("gh", "pr", "list",
		"--head", branch,
		"--state", "all",
		"--json", "number,state,mergedAt",
		"--limit", "1",
	)
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(ee.Stderr))
			if msg != "" {
				return PRStatus{}, fmt.Errorf("gh pr list: %s", msg)
			}
		}
		return PRStatus{}, fmt.Errorf("gh pr list: %w", err)
	}
	return parsePRList(out)
}

func parsePRList(raw []byte) (PRStatus, error) {
	var rows []struct {
		Number   int        `json:"number"`
		State    string     `json:"state"`
		MergedAt *time.Time `json:"mergedAt"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return PRStatus{}, fmt.Errorf("gh pr list: parse json: %w", err)
	}
	if len(rows) == 0 {
		return PRStatus{Found: false}, nil
	}
	r := rows[0]
	ps := PRStatus{Found: true, Number: r.Number, State: r.State}
	if r.MergedAt != nil {
		ps.MergedAt = *r.MergedAt
	}
	return ps, nil
}
