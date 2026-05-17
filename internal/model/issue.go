package model

import "time"

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
	ID          int       `yaml:"id"`
	Title       string    `yaml:"title"`
	Status      Status    `yaml:"status"`
	Description string    `yaml:"description"`
	References  []string  `yaml:"references"`
	Scope       []string  `yaml:"scope"`
	CreatedAt   time.Time `yaml:"created_at"`
	UpdatedAt   time.Time `yaml:"updated_at"`
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
