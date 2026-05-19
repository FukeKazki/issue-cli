package tui

import (
	"reflect"
	"testing"

	"github.com/FukeKazki/issue-cli/internal/model"
)

func TestFilterRepoFiles(t *testing.T) {
	files := []string{
		"README.md",
		"cmd/issue/main.go",
		"internal/cli/create.go",
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
				m.scopeArea.SetValue("@cmd/issue/main.go")
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
			name: "edit scenario with unchanged values is clean",
			iss: model.Issue{
				Title:       "existing",
				Status:      model.StatusInProgress,
				Description: "body",
				References:  []string{"https://example.com"},
				Scope:       []string{"@cmd/issue/main.go"},
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
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			iss := tc.iss
			fm := newFormModel(&iss, "test")
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
