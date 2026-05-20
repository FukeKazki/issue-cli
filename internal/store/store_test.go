package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FukeKazki/issue-cli/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	return &Store{Dir: filepath.Join(dir, ".issues")}
}

func TestNextIDEmpty(t *testing.T) {
	s := newTestStore(t)
	id, err := s.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("want 1, got %d", id)
	}
}

func TestNextIDWithGap(t *testing.T) {
	s := newTestStore(t)
	if err := s.EnsureDir(); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"1.yaml", "3.yaml", "7.yaml"} {
		if err := os.WriteFile(filepath.Join(s.Dir, n), []byte("id: 1\ntitle: x\nstatus: TODO\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	id, err := s.NextID()
	if err != nil {
		t.Fatal(err)
	}
	if id != 8 {
		t.Fatalf("want 8, got %d", id)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	s := newTestStore(t)
	in := &model.Issue{
		ID:         1,
		Title:      "Test",
		Status:     model.StatusTODO,
		References: []string{"https://example.com", "design.md"},
		Scope:      []string{"@apps/web/x.tsx"},
		BlockedBy:  []int{2, 3},
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != in.Title || out.Status != in.Status || len(out.References) != 2 || len(out.Scope) != 1 {
		t.Fatalf("round-trip mismatch: %+v", out)
	}
	if len(out.BlockedBy) != 2 || out.BlockedBy[0] != 2 || out.BlockedBy[1] != 3 {
		t.Fatalf("blocked_by round-trip mismatch: %+v", out.BlockedBy)
	}
	if out.CreatedAt.IsZero() || out.UpdatedAt.IsZero() {
		t.Fatalf("timestamps should be set")
	}
}

func TestSaveAndLoadRoundTripWithType(t *testing.T) {
	s := newTestStore(t)
	in := &model.Issue{
		ID:     1,
		Title:  "Typed",
		Status: model.StatusTODO,
		Type:   model.TypeFeature,
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != model.TypeFeature {
		t.Fatalf("type round-trip mismatch: got %q, want %q", out.Type, model.TypeFeature)
	}
}

func TestSaveAndLoadRoundTripEmptyType(t *testing.T) {
	s := newTestStore(t)
	in := &model.Issue{
		ID:     1,
		Title:  "Untyped",
		Status: model.StatusTODO,
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	out, err := s.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if out.Type != "" {
		t.Fatalf("empty type should round-trip as empty, got %q", out.Type)
	}
}

func TestSaveRejectsInvalidType(t *testing.T) {
	s := newTestStore(t)
	err := s.Save(&model.Issue{
		ID:     1,
		Title:  "Bad type",
		Status: model.StatusTODO,
		Type:   model.Type("Bogus"),
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestSaveRejectsSelfBlock(t *testing.T) {
	s := newTestStore(t)
	err := s.Save(&model.Issue{
		ID:        1,
		Title:     "Test",
		Status:    model.StatusTODO,
		BlockedBy: []int{1},
	})
	if err == nil {
		t.Fatal("expected error for self-reference in blocked_by")
	}
}

func TestSaveRejectsNonPositiveBlockedBy(t *testing.T) {
	cases := []struct {
		name string
		ids  []int
	}{
		{"zero", []int{0}},
		{"negative", []int{-1}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newTestStore(t)
			err := s.Save(&model.Issue{
				ID:        1,
				Title:     "Test",
				Status:    model.StatusTODO,
				BlockedBy: c.ids,
			})
			if err == nil {
				t.Fatalf("expected error for blocked_by=%v", c.ids)
			}
		})
	}
}

func TestSaveRejectsEmptyTitle(t *testing.T) {
	s := newTestStore(t)
	err := s.Save(&model.Issue{ID: 1, Title: "  ", Status: model.StatusTODO})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	if err := s.Save(&model.Issue{ID: 1, Title: "t", Status: model.StatusTODO}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(1); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Path(1)); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, got err=%v", err)
	}
	if err := s.Delete(1); err == nil {
		t.Fatal("expected error when deleting missing issue")
	}
}

func TestLoadAllSorted(t *testing.T) {
	s := newTestStore(t)
	for _, id := range []int{3, 1, 2} {
		if err := s.Save(&model.Issue{ID: id, Title: "t", Status: model.StatusTODO}); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].ID != 1 || list[2].ID != 3 {
		t.Fatalf("not sorted: %+v", list)
	}
}
