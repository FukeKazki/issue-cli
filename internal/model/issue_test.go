package model

import "testing"

func TestStatusRankOrdering(t *testing.T) {
	want := []Status{StatusTODO, StatusInProgress, StatusReviews, StatusDone}
	for i, s := range want {
		if got := StatusRank(s); got != i {
			t.Errorf("StatusRank(%q) = %d, want %d", s, got, i)
		}
	}
	if got := StatusRank(Status("Bogus")); got != -1 {
		t.Errorf("unknown status rank = %d, want -1", got)
	}
}

func TestAllTypesOrdering(t *testing.T) {
	want := []Type{TypeBug, TypeFeature, TypeEnhancement, TypeDocs, TypeRefactor}
	got := AllTypes()
	if len(got) != len(want) {
		t.Fatalf("AllTypes() len = %d, want %d", len(got), len(want))
	}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("AllTypes()[%d] = %q, want %q", i, got[i], v)
		}
	}
}

func TestParseType(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		want   Type
		wantOK bool
	}{
		{"canonical Bug", "Bug", TypeBug, true},
		{"canonical Feature", "Feature", TypeFeature, true},
		{"canonical Enhancement", "Enhancement", TypeEnhancement, true},
		{"canonical Docs", "Docs", TypeDocs, true},
		{"canonical Refactor", "Refactor", TypeRefactor, true},
		{"empty rejected", "", "", false},
		{"lowercase rejected", "bug", "", false},
		{"uppercase rejected", "BUG", "", false},
		{"unknown rejected", "Bogus", "", false},
		{"whitespace rejected", " Bug", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := ParseType(c.in)
			if ok != c.wantOK {
				t.Fatalf("ParseType(%q) ok = %v, want %v", c.in, ok, c.wantOK)
			}
			if got != c.want {
				t.Errorf("ParseType(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestAdvanceStatusForwardOnly(t *testing.T) {
	cases := []struct {
		name      string
		from, to  Status
		want      Status
		advanced  bool
	}{
		{"TODO to In Progress", StatusTODO, StatusInProgress, StatusInProgress, true},
		{"TODO to Done", StatusTODO, StatusDone, StatusDone, true},
		{"In Progress to TODO (no-op)", StatusInProgress, StatusTODO, StatusInProgress, false},
		{"Reviews to In Progress (no-op)", StatusReviews, StatusInProgress, StatusReviews, false},
		{"Done to Reviews (no-op)", StatusDone, StatusReviews, StatusDone, false},
		{"In Progress to In Progress (no-op)", StatusInProgress, StatusInProgress, StatusInProgress, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			iss := &Issue{Status: c.from}
			got := iss.AdvanceStatus(c.to)
			if got != c.advanced {
				t.Errorf("advanced = %v, want %v", got, c.advanced)
			}
			if iss.Status != c.want {
				t.Errorf("status = %q, want %q", iss.Status, c.want)
			}
		})
	}
}
