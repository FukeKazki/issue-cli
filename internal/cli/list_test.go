package cli

import (
	"path/filepath"
	"testing"

	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	return &store.Store{Dir: filepath.Join(t.TempDir(), ".issues")}
}

func TestAdvanceOnCheckout(t *testing.T) {
	cases := []struct {
		name   string
		from   model.Status
		want   model.Status
	}{
		{"TODO advances", model.StatusTODO, model.StatusInProgress},
		{"In Progress unchanged", model.StatusInProgress, model.StatusInProgress},
		{"Reviews unchanged (forward-only)", model.StatusReviews, model.StatusReviews},
		{"Done unchanged", model.StatusDone, model.StatusDone},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestStore(t)
			iss := &model.Issue{ID: 1, Title: "t", Status: c.from}
			if err := s.Save(iss); err != nil {
				t.Fatal(err)
			}
			if err := advanceOnCheckout(s, 1); err != nil {
				t.Fatalf("advanceOnCheckout: %v", err)
			}
			got, err := s.Load(1)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != c.want {
				t.Errorf("status = %q, want %q", got.Status, c.want)
			}
		})
	}
}
