package cli

import (
	"testing"

	"github.com/FukeKazki/issue-cli/internal/model"
)

func TestApplyNewFlags_AllFields(t *testing.T) {
	iss := &model.Issue{ID: 5, Status: model.StatusTODO}
	f := newFlags{
		title:       "  Add parent flag  ",
		typeFlag:    "feature",
		description: "let new accept a parent",
		scope:       "internal/cli, cmd/issue-cli ",
		references:  "#33,docs",
		blockedBy:   "1, #2",
		parent:      "#3",
	}
	if err := applyNewFlags(iss, f); err != nil {
		t.Fatalf("applyNewFlags: %v", err)
	}

	if iss.Title != "Add parent flag" {
		t.Errorf("Title = %q, want trimmed %q", iss.Title, "Add parent flag")
	}
	if iss.Type != model.TypeFeature {
		t.Errorf("Type = %q, want %q", iss.Type, model.TypeFeature)
	}
	if iss.Description != "let new accept a parent" {
		t.Errorf("Description = %q", iss.Description)
	}
	if got, want := iss.Scope, []string{"internal/cli", "cmd/issue-cli"}; !equalStrs(got, want) {
		t.Errorf("Scope = %v, want %v", got, want)
	}
	if got, want := iss.References, []string{"#33", "docs"}; !equalStrs(got, want) {
		t.Errorf("References = %v, want %v", got, want)
	}
	if got, want := iss.BlockedBy, []int{1, 2}; !equalInts(got, want) {
		t.Errorf("BlockedBy = %v, want %v", got, want)
	}
	if iss.Parent == nil || *iss.Parent != 3 {
		t.Errorf("Parent = %v, want 3", iss.Parent)
	}
}

func TestApplyNewFlags_OmittedFieldsStayZero(t *testing.T) {
	iss := &model.Issue{ID: 1, Status: model.StatusTODO}
	if err := applyNewFlags(iss, newFlags{title: "just a title"}); err != nil {
		t.Fatalf("applyNewFlags: %v", err)
	}
	if iss.Type != "" {
		t.Errorf("Type = %q, want empty", iss.Type)
	}
	if iss.Parent != nil {
		t.Errorf("Parent = %v, want nil", iss.Parent)
	}
	if iss.BlockedBy != nil || iss.Scope != nil || iss.References != nil {
		t.Errorf("expected nil slices, got blocked=%v scope=%v refs=%v", iss.BlockedBy, iss.Scope, iss.References)
	}
}

func TestApplyNewFlags_Errors(t *testing.T) {
	cases := []struct {
		name string
		f    newFlags
	}{
		{"whitespace title", newFlags{title: "   "}},
		{"unknown type", newFlags{title: "t", typeFlag: "bogus"}},
		{"non-integer parent", newFlags{title: "t", parent: "abc"}},
		{"non-positive parent", newFlags{title: "t", parent: "0"}},
		{"bad blocked-by", newFlags{title: "t", blockedBy: "1,x"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			iss := &model.Issue{ID: 9, Status: model.StatusTODO}
			if err := applyNewFlags(iss, c.f); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestParseIDArg(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"3", 3, false},
		{"#42", 42, false},
		{"  7 ", 7, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseIDArg(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseIDArg(%q): expected error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseIDArg(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseIDArg(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func equalStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
