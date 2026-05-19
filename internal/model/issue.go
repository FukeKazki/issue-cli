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
	ID          int               `yaml:"id" json:"id"`
	Title       string            `yaml:"title" json:"title"`
	Status      Status            `yaml:"status" json:"status"`
	Description string            `yaml:"description" json:"description"`
	References  []string          `yaml:"references" json:"references"`
	Scope       []string          `yaml:"scope" json:"scope"`
	Metadata    map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at" json:"updated_at"`
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
