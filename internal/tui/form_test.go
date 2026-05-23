package tui

import (
	"reflect"
	"testing"

	"github.com/FukeKazki/issue-cli/internal/model"
)

func TestFilterRepoFiles(t *testing.T) {
	files := []string{
		"README.md",
		"cmd/issue-cli/main.go",
		"internal/cli/new.go",
		"internal/cli/list.go",
		"internal/store/store.go",
		"internal/tui/form.go",
		"internal/tui/list.go",
	}

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "empty returns all (up to limit)",
			query: "",
			want:  files,
		},
		{
			name:  "exact prefix match comes first",
			query: "internal/tui",
			want:  []string{"internal/tui/form.go", "internal/tui/list.go"},
		},
		{
			name:  "post-slash prefix ranks above plain substring",
			query: "list.go",
			want:  []string{"internal/cli/list.go", "internal/tui/list.go"},
		},
		{
			name:  "case insensitive",
			query: "README",
			want:  []string{"README.md"},
		},
		{
			name:  "no match",
			query: "nothing",
			want:  nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterRepoFiles(files, tc.query)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("filterRepoFiles(%q) = %v, want %v", tc.query, got, tc.want)
			}
		})
	}
}

func TestWrappedLineCount(t *testing.T) {
	tests := []struct {
		name  string
		value string
		width int
		want  int
	}{
		{"empty value renders one row", "", 40, 1},
		{"single short line", "hello", 40, 1},
		{"wraps at width boundary", "abcdefghij", 5, 2},
		{"wraps with remainder", "abcdefghijk", 5, 3},
		{"newlines counted as separate lines", "a\nb\nc", 40, 3},
		{"blank line counts as one row", "a\n\nb", 40, 3},
		{"long line plus short line", "abcdefghij\nshort", 5, 3},
		{"width<=0 falls back to one row", "anything", 0, 1},
		{"east-asian width counts as 2", "あいう", 4, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := wrappedLineCount(tc.value, tc.width); got != tc.want {
				t.Errorf("wrappedLineCount(%q, %d) = %d, want %d", tc.value, tc.width, got, tc.want)
			}
		})
	}
}

func TestIsDirty(t *testing.T) {
	tests := []struct {
		name   string
		iss    model.Issue
		mutate func(m *formModel)
		want   bool
	}{
		{
			name: "create initial state is clean",
			iss:  model.Issue{Status: model.StatusTODO},
			want: false,
		},
		{
			name: "title typed is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.titleInput.SetValue("hello")
			},
			want: true,
		},
		{
			name: "title whitespace only is clean",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.titleInput.SetValue("   ")
			},
			want: false,
		},
		{
			name: "description typed is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.descArea.SetValue("body")
			},
			want: true,
		},
		{
			name: "description trailing newline only is clean",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.descArea.SetValue("\n")
			},
			want: false,
		},
		{
			name: "references typed is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.refsArea.SetValue("https://example.com")
			},
			want: true,
		},
		{
			name: "scope typed is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.scopeArea.SetValue("@cmd/issue-cli/main.go")
			},
			want: true,
		},
		{
			name: "status changed from initial is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.statusIdx = (m.statusIdx + 1) % len(m.statuses)
			},
			want: true,
		},
		{
			name: "type set from (none) is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.typeIdx = (m.typeIdx + 1) % len(m.types)
			},
			want: true,
		},
		{
			name: "type unchanged from initial is clean",
			iss: model.Issue{
				Status: model.StatusTODO,
				Type:   model.TypeFeature,
			},
			want: false,
		},
		{
			name: "type changed from initial is dirty",
			iss: model.Issue{
				Status: model.StatusTODO,
				Type:   model.TypeFeature,
			},
			mutate: func(m *formModel) {
				m.typeIdx = (m.typeIdx + 1) % len(m.types)
			},
			want: true,
		},
		{
			name: "edit scenario with unchanged values is clean",
			iss: model.Issue{
				Title:       "existing",
				Status:      model.StatusInProgress,
				Description: "body",
				References:  []string{"https://example.com"},
				Scope:       []string{"@cmd/issue-cli/main.go"},
			},
			want: false,
		},
		{
			name: "edit scenario with mutated title is dirty",
			iss: model.Issue{
				Title:  "existing",
				Status: model.StatusInProgress,
			},
			mutate: func(m *formModel) {
				m.titleInput.SetValue("updated")
			},
			want: true,
		},
		{
			name: "blocked_by typed is dirty",
			iss:  model.Issue{Status: model.StatusTODO},
			mutate: func(m *formModel) {
				m.blockedByArea.SetValue("2")
			},
			want: true,
		},
		{
			name: "edit scenario with unchanged blocked_by is clean",
			iss: model.Issue{
				Title:     "existing",
				Status:    model.StatusInProgress,
				BlockedBy: []int{2, 3},
			},
			want: false,
		},
		{
			name: "edit scenario with mutated blocked_by is dirty",
			iss: model.Issue{
				Title:     "existing",
				Status:    model.StatusInProgress,
				BlockedBy: []int{2, 3},
			},
			mutate: func(m *formModel) {
				m.blockedByArea.SetValue("2")
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iss := tc.iss
			fm := newFormModel(&iss, "test", nil)
			if tc.mutate != nil {
				tc.mutate(&fm)
			}
			if got := fm.isDirty(); got != tc.want {
				t.Errorf("isDirty() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStringSliceEqual(t *testing.T) {
	tests := []struct {
		name string
		a, b []string
		want bool
	}{
		{"both nil", nil, nil, true},
		{"nil vs empty", nil, []string{}, true},
		{"same elements", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different lengths", []string{"a"}, []string{"a", "b"}, false},
		{"different elements", []string{"a"}, []string{"b"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stringSliceEqual(tc.a, tc.b); got != tc.want {
				t.Errorf("stringSliceEqual(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestParseIssueIDs(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []int
		wantErr bool
	}{
		{"empty returns nil", "", nil, false},
		{"whitespace-only returns nil", "  \n\n  ", nil, false},
		{"plain decimals", "1\n2\n3", []int{1, 2, 3}, false},
		{"hash prefix tolerated", "#1\n#2", []int{1, 2}, false},
		{"mixed forms with blank lines", "\n1\n#2\n  3  \n", []int{1, 2, 3}, false},
		{"non-numeric rejected", "abc", nil, true},
		{"zero rejected", "0", nil, true},
		{"negative rejected", "-1", nil, true},
		{"partial-failure rejects the whole value", "1\nabc", nil, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseIssueIDs(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseIssueIDs(%q) err = %v, wantErr = %v", tc.raw, err, tc.wantErr)
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseIssueIDs(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestJoinIssueIDs(t *testing.T) {
	tests := []struct {
		name string
		in   []int
		want string
	}{
		{"nil returns empty", nil, ""},
		{"empty returns empty", []int{}, ""},
		{"single", []int{7}, "7"},
		{"multiple newline-joined", []int{1, 2, 3}, "1\n2\n3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := joinIssueIDs(tc.in); got != tc.want {
				t.Errorf("joinIssueIDs(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestFilterIssueCandidates(t *testing.T) {
	cands := []IssueCandidate{
		{ID: 1, Title: "implement feature X"},
		{ID: 2, Title: "fix bug Y"},
		{ID: 10, Title: "add docs"},
		{ID: 11, Title: "polish UI"},
		{ID: 12, Title: "relate to feature"},
	}
	t.Run("empty query returns all except excluded", func(t *testing.T) {
		got := filterIssueCandidates(cands, "", 2)
		want := []IssueCandidate{
			{ID: 1, Title: "implement feature X"},
			{ID: 10, Title: "add docs"},
			{ID: 11, Title: "polish UI"},
			{ID: 12, Title: "relate to feature"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filterIssueCandidates(empty) = %v, want %v", got, want)
		}
	})
	t.Run("id prefix ranks above title", func(t *testing.T) {
		got := filterIssueCandidates(cands, "1", 0)
		// "1" prefixes ids 1, 10, 11, 12 — title match for "implement feature X"
		// and "fix bug Y" is irrelevant (no "1" substring). Ordering: by id.
		want := []IssueCandidate{
			{ID: 1, Title: "implement feature X"},
			{ID: 10, Title: "add docs"},
			{ID: 11, Title: "polish UI"},
			{ID: 12, Title: "relate to feature"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filterIssueCandidates(\"1\") = %v, want %v", got, want)
		}
	})
	t.Run("title substring case-insensitive", func(t *testing.T) {
		got := filterIssueCandidates(cands, "FEATURE", 0)
		want := []IssueCandidate{
			{ID: 1, Title: "implement feature X"},
			{ID: 12, Title: "relate to feature"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filterIssueCandidates(\"FEATURE\") = %v, want %v", got, want)
		}
	})
	t.Run("hash prefix tolerated", func(t *testing.T) {
		got := filterIssueCandidates(cands, "#2", 0)
		want := []IssueCandidate{{ID: 2, Title: "fix bug Y"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filterIssueCandidates(\"#2\") = %v, want %v", got, want)
		}
	})
	t.Run("excludeID drops self even on prefix match", func(t *testing.T) {
		got := filterIssueCandidates(cands, "1", 1)
		// ID 1 is the self — must not appear even though it prefixes "1".
		want := []IssueCandidate{
			{ID: 10, Title: "add docs"},
			{ID: 11, Title: "polish UI"},
			{ID: 12, Title: "relate to feature"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("filterIssueCandidates exclude self = %v, want %v", got, want)
		}
	})
	t.Run("no match returns nil", func(t *testing.T) {
		got := filterIssueCandidates(cands, "nothing-here", 0)
		if got != nil {
			t.Errorf("filterIssueCandidates(no match) = %v, want nil", got)
		}
	})
}

func TestWindowAround(t *testing.T) {
	tests := []struct {
		name              string
		idx, total, size  int
		wantStart, wantEnd int
	}{
		{"fits within size", 0, 5, 8, 0, 5},
		{"window at start", 1, 20, 8, 0, 8},
		{"window centered", 10, 20, 8, 6, 14},
		{"window clamped to end", 19, 20, 8, 12, 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, e := windowAround(tc.idx, tc.total, tc.size)
			if s != tc.wantStart || e != tc.wantEnd {
				t.Errorf("windowAround(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tc.idx, tc.total, tc.size, s, e, tc.wantStart, tc.wantEnd)
			}
		})
	}
}
