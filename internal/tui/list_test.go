package tui

import "testing"

func TestListColumns(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		showPreview bool
		wantList    int
		wantPreview int
	}{
		{"wide with preview splits", 100, true, 49, 50},
		{"wide without preview is full width", 100, false, 100, 0},
		{"below preview threshold hides preview", 79, true, 79, 0},
		{"narrow without preview never exceeds width", 25, false, 25, 0},
		{"narrow with preview shrinks to width", 25, true, 25, 0},
		{"at preview threshold splits", 80, true, 39, 40},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			listW, previewW := listColumns(tc.width, tc.showPreview)
			if listW != tc.wantList || previewW != tc.wantPreview {
				t.Errorf("listColumns(%d, %v) = (%d, %d), want (%d, %d)",
					tc.width, tc.showPreview, listW, previewW, tc.wantList, tc.wantPreview)
			}
			// The two panels (plus a 1-col gap when both present) must never
			// exceed the terminal width.
			total := listW
			if previewW > 0 {
				total += previewW + 1
			}
			if tc.width > 0 && total > tc.width {
				t.Errorf("panels total %d exceed width %d", total, tc.width)
			}
		})
	}
}
