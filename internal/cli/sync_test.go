package cli

import (
	"testing"
	"time"

	"github.com/FukeKazki/issue-cli/internal/gh"
	"github.com/FukeKazki/issue-cli/internal/model"
)

func withStub(t *testing.T, stub func(branch string) (gh.PRStatus, error)) {
	t.Helper()
	orig := prStatusForBranch
	prStatusForBranch = stub
	t.Cleanup(func() { prStatusForBranch = orig })
}

func TestSyncIssueAdvancesOnOpenPR(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(&model.Issue{ID: 1, Title: "t", Status: model.StatusInProgress}); err != nil {
		t.Fatal(err)
	}
	withStub(t, func(branch string) (gh.PRStatus, error) {
		if branch != "issue/1" {
			t.Errorf("branch = %q, want issue/1", branch)
		}
		return gh.PRStatus{Found: true, Number: 9, State: "OPEN"}, nil
	})
	if err := syncIssue(s, 1); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load(1)
	if got.Status != model.StatusReviews {
		t.Errorf("status = %q, want Reviews", got.Status)
	}
}

func TestSyncIssueAdvancesOnMergedPR(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(&model.Issue{ID: 2, Title: "t", Status: model.StatusReviews}); err != nil {
		t.Fatal(err)
	}
	withStub(t, func(branch string) (gh.PRStatus, error) {
		return gh.PRStatus{Found: true, Number: 9, State: "MERGED", MergedAt: time.Now()}, nil
	})
	if err := syncIssue(s, 2); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load(2)
	if got.Status != model.StatusDone {
		t.Errorf("status = %q, want Done", got.Status)
	}
}

func TestSyncIssueNoPRLeavesStatus(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(&model.Issue{ID: 3, Title: "t", Status: model.StatusTODO}); err != nil {
		t.Fatal(err)
	}
	withStub(t, func(branch string) (gh.PRStatus, error) {
		return gh.PRStatus{Found: false}, nil
	})
	if err := syncIssue(s, 3); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load(3)
	if got.Status != model.StatusTODO {
		t.Errorf("status = %q, want TODO", got.Status)
	}
}

func TestSyncIssueForwardOnly(t *testing.T) {
	// Already Done — even if PR is somehow OPEN, must not regress.
	s := newTestStore(t)
	if err := s.Save(&model.Issue{ID: 4, Title: "t", Status: model.StatusDone}); err != nil {
		t.Fatal(err)
	}
	withStub(t, func(branch string) (gh.PRStatus, error) {
		return gh.PRStatus{Found: true, Number: 9, State: "OPEN"}, nil
	})
	if err := syncIssue(s, 4); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load(4)
	if got.Status != model.StatusDone {
		t.Errorf("status = %q, want Done (no regression)", got.Status)
	}
}
