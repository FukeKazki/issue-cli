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

// Type classifies what kind of work an Issue represents. It is independent of
// Status (which tracks lifecycle) — Type is the orthogonal "what is this
// change about" axis. Empty string is a valid value and means "no type set",
// which keeps existing on-disk issues (created before Type was introduced)
// backwards-compatible.
type Type string

const (
	TypeBug         Type = "Bug"
	TypeFeature     Type = "Feature"
	TypeEnhancement Type = "Enhancement"
	TypeDocs        Type = "Docs"
	TypeRefactor    Type = "Refactor"
)

// AllTypes returns the canonical Type values in display order. Order matches
// the form picker and the issue body's enumeration.
func AllTypes() []Type {
	return []Type{TypeBug, TypeFeature, TypeEnhancement, TypeDocs, TypeRefactor}
}

// ParseType is the canonical-strict string→Type parser. It accepts only the
// exact canonical spellings ("Bug", "Feature", "Docs", "Refactor"); empty
// strings and any other input return ok=false. Used by store.validate to keep
// YAML on disk in canonical form. Empty Type is allowed at the validate layer
// (Type is optional) — callers decide whether empty is acceptable.
func ParseType(s string) (Type, bool) {
	for _, v := range AllTypes() {
		if string(v) == s {
			return v, true
		}
	}
	return "", false
}

// ParseTypeFromCLI normalizes a user-supplied CLI argument to a canonical Type.
// Accepts case-insensitive input so `bug`, `BUG`, `Bug` all resolve to TypeBug.
func ParseTypeFromCLI(s string) (Type, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bug":
		return TypeBug, true
	case "feature":
		return TypeFeature, true
	case "enhancement":
		return TypeEnhancement, true
	case "docs":
		return TypeDocs, true
	case "refactor":
		return TypeRefactor, true
	}
	return "", false
}

type Issue struct {
	ID          int               `yaml:"id" json:"id"`
	Title       string            `yaml:"title" json:"title"`
	Status      Status            `yaml:"status" json:"status"`
	Type        Type              `yaml:"type,omitempty" json:"type,omitempty"`
	Description string            `yaml:"description" json:"description"`
	References  []string          `yaml:"references" json:"references"`
	Scope       []string          `yaml:"scope" json:"scope"`
	BlockedBy   []int             `yaml:"blocked_by" json:"blocked_by"`
	Parent      *int              `yaml:"parent,omitempty" json:"parent,omitempty"`
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
