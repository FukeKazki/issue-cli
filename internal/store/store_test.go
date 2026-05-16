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
	if out.CreatedAt.IsZero() || out.UpdatedAt.IsZero() {
		t.Fatalf("timestamps should be set")
	}
}

func TestSaveRejectsEmptyTitle(t *testing.T) {
	s := newTestStore(t)
	err := s.Save(&model.Issue{ID: 1, Title: "  ", Status: model.StatusTODO})
	if err == nil {
		t.Fatal("expected error for empty title")
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
