package model

import (
	"strings"
	"time"
)

type Status string

const (
	StatusTODO       Status = "TODO"
	StatusInProgress Status = "In Progress"
	StatusReviews    Status = "Reviews"
	StatusDone       Status = "Done"
)

func AllStatuses() []Status {
	return []Status{StatusTODO, StatusInProgress, StatusReviews, StatusDone}
}

func OpenStatuses() []Status {
	return []Status{StatusTODO, StatusInProgress, StatusReviews}
}

func ParseStatus(s string) (Status, bool) {
	for _, v := range AllStatuses() {
		if string(v) == s {
			return v, true
		}
	}
	return "", false
}

// ParseStatusFromCLI normalizes a user-supplied CLI argument to a canonical Status.
// Accepts case-insensitive aliases so `DONE`, `in-progress`, `review` etc. work
// without shell-quoting the canonical `In Progress` spelling.
func ParseStatusFromCLI(s string) (Status, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "todo":
		return StatusTODO, true
	case "in progress", "in-progress", "in_progress", "inprogress":
		return StatusInProgress, true
	case "reviews", "review":
		return StatusReviews, true
	case "done":
		return StatusDone, true
	}
	return "", false
}

// StatusRank returns a monotonic rank for a status used to order transitions.
// Unknown statuses return -1.
func StatusRank(s Status) int {
	for i, v := range AllStatuses() {
		if v == s {
			return i
		}
	}
	return -1
}

type Issue struct {
	ID          int       `yaml:"id" json:"id"`
	Title       string    `yaml:"title" json:"title"`
	Status      Status    `yaml:"status" json:"status"`
	Description string    `yaml:"description" json:"description"`
	References  []string  `yaml:"references" json:"references"`
	Scope       []string  `yaml:"scope" json:"scope"`
	Run         *Run      `yaml:"run,omitempty" json:"run,omitempty"`
	CreatedAt   time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at" json:"updated_at"`
}

// RunResult is the terminal outcome of a workflow execution recorded against
// an issue. Canonical values are lowercase to match the YAML on disk; use
// ParseRunResult to normalize CLI input.
type RunResult string

const (
	RunResultSuccess     RunResult = "success"
	RunResultFailure     RunResult = "failure"
	RunResultInterrupted RunResult = "interrupted"
)

// Run holds metadata for the most recent workflow execution against the
// issue. All fields are optional so a hand-managed issue (or an issue that
// has never been claimed) can omit the block entirely — `Issue.Run` is a
// pointer so a nil run is dropped from the YAML output via `omitempty`.
//
// `claim` populates Workflow / ID / StartedAt; `release` populates
// FinishedAt / Result / Error / PRURL. A fresh claim overwrites the previous
// Run (the field reflects the most recent execution, not a history).
type Run struct {
	Workflow   string    `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	ID         string    `yaml:"id,omitempty" json:"id,omitempty"`
	StartedAt  time.Time `yaml:"started_at,omitempty" json:"started_at,omitempty"`
	FinishedAt time.Time `yaml:"finished_at,omitempty" json:"finished_at,omitempty"`
	Result     RunResult `yaml:"result,omitempty" json:"result,omitempty"`
	Error      string    `yaml:"error,omitempty" json:"error,omitempty"`
	PRURL      string    `yaml:"pr_url,omitempty" json:"pr_url,omitempty"`
}

// ParseRunResult normalizes a user-supplied CLI value to a canonical
// RunResult (case-insensitive; whitespace-trimmed). Returns false for empty
// or unknown values so callers can surface a friendly error.
func ParseRunResult(s string) (RunResult, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "success":
		return RunResultSuccess, true
	case "failure":
		return RunResultFailure, true
	case "interrupted":
		return RunResultInterrupted, true
	}
	return "", false
}

func (i *Issue) IsOpen() bool {
	return i.Status != StatusDone
}

// AdvanceStatus moves Status forward to target only if target outranks the
// current status. Returns true when the field changed.
func (i *Issue) AdvanceStatus(target Status) bool {
	if StatusRank(target) <= StatusRank(i.Status) {
		return false
	}
	i.Status = target
	return true
}
