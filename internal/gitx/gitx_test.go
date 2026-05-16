package gitx

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCheckoutIssueCreatesAndReuses(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(name string, args ...string) {
		t.Helper()
		c := exec.Command(name, args...)
		c.Stdout, c.Stderr = os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			t.Fatalf("%s %v: %v", name, args, err)
		}
	}
	run("git", "init", "-q")
	run("git", "config", "user.email", "t@local")
	run("git", "config", "user.name", "t")
	run("git", "commit", "--allow-empty", "-q", "-m", "init")

	if err := CheckoutIssue(42); err != nil {
		t.Fatalf("first checkout: %v", err)
	}
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "issue/42" {
		t.Fatalf("branch = %q, want issue/42", got)
	}

	// switch back, then re-checkout (must take the "exists" branch).
	run("git", "checkout", "-q", "-")
	if err := CheckoutIssue(42); err != nil {
		t.Fatalf("re-checkout: %v", err)
	}
	out, _ = exec.Command("git", "branch", "--show-current").Output()
	if got := strings.TrimSpace(string(out)); got != "issue/42" {
		t.Fatalf("after re-checkout: branch = %q, want issue/42", got)
	}
}
