package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Takt.Workflow != "" {
		t.Errorf("expected empty workflow, got %q", cfg.Takt.Workflow)
	}
	if cfg.Takt.WorktreeDir != "" {
		t.Errorf("expected empty worktree-dir, got %q", cfg.Takt.WorktreeDir)
	}
}

func TestLoadFullConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte("takt:\n  workflow: my-workflow\n  worktree-dir: /tmp/wt\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Takt.Workflow != "my-workflow" {
		t.Errorf("workflow = %q, want %q", cfg.Takt.Workflow, "my-workflow")
	}
	if cfg.Takt.WorktreeDir != "/tmp/wt" {
		t.Errorf("worktree-dir = %q, want %q", cfg.Takt.WorktreeDir, "/tmp/wt")
	}
}

func TestLoadPartialConfig(t *testing.T) {
	dir := t.TempDir()
	content := []byte("takt:\n  workflow: only-wf\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Takt.Workflow != "only-wf" {
		t.Errorf("workflow = %q, want %q", cfg.Takt.Workflow, "only-wf")
	}
	if cfg.Takt.WorktreeDir != "" {
		t.Errorf("worktree-dir = %q, want empty", cfg.Takt.WorktreeDir)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	content := []byte(":\n  invalid: [unclosed\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(dir)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
