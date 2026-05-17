package gh

import (
	"testing"
	"time"
)

func TestParsePRListEmpty(t *testing.T) {
	ps, err := parsePRList([]byte(`[]`))
	if err != nil {
		t.Fatal(err)
	}
	if ps.Found {
		t.Errorf("Found = true, want false on empty list")
	}
}

func TestParsePRListOpen(t *testing.T) {
	ps, err := parsePRList([]byte(`[{"number":42,"state":"OPEN","mergedAt":null}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !ps.Found || ps.Number != 42 || ps.State != "OPEN" || !ps.MergedAt.IsZero() {
		t.Errorf("unexpected status: %+v", ps)
	}
}

func TestParsePRListMerged(t *testing.T) {
	ps, err := parsePRList([]byte(`[{"number":7,"state":"MERGED","mergedAt":"2026-05-01T12:00:00Z"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if !ps.Found || ps.State != "MERGED" || ps.MergedAt.IsZero() {
		t.Errorf("unexpected status: %+v", ps)
	}
	if got := ps.MergedAt.UTC(); got.Year() != 2026 || got.Month() != time.May {
		t.Errorf("MergedAt = %v, want 2026-05-*", got)
	}
}

func TestPRStatusForBranchUsesRunSeam(t *testing.T) {
	orig := run
	t.Cleanup(func() { run = orig })

	var gotArgs []string
	run = func(name string, args ...string) ([]byte, error) {
		if name != "gh" {
			t.Fatalf("name = %q, want gh", name)
		}
		gotArgs = args
		return []byte(`[{"number":1,"state":"OPEN","mergedAt":null}]`), nil
	}
	ps, err := PRStatusForBranch("issue/9")
	if err != nil {
		t.Fatal(err)
	}
	if !ps.Found || ps.Number != 1 {
		t.Errorf("status = %+v", ps)
	}
	if len(gotArgs) < 3 || gotArgs[0] != "pr" || gotArgs[1] != "list" {
		t.Errorf("args = %v, want gh pr list ...", gotArgs)
	}
	var sawHead bool
	for i, a := range gotArgs {
		if a == "--head" && i+1 < len(gotArgs) && gotArgs[i+1] == "issue/9" {
			sawHead = true
		}
	}
	if !sawHead {
		t.Errorf("missing --head issue/9 in args: %v", gotArgs)
	}
}
