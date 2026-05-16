package gitx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func CheckoutIssue(id int) error {
	branch := fmt.Sprintf("issue/%d", id)
	if branchExists(branch) {
		return run("git", "checkout", branch)
	}
	return run("git", "checkout", "-b", branch)
}

// CurrentBranch returns the current Git branch name, or an empty string
// if HEAD is detached or the repo cannot be queried.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "symbolic-ref", "--short", "-q", "HEAD").Output()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentIssueID returns the id parsed from the current branch when it
// matches `issue/<id>`. Returns 0 if the branch does not match.
func CurrentIssueID() (int, error) {
	br, err := CurrentBranch()
	if err != nil || br == "" {
		return 0, err
	}
	rest, ok := strings.CutPrefix(br, "issue/")
	if !ok {
		return 0, nil
	}
	id, err := strconv.Atoi(rest)
	if err != nil {
		return 0, nil
	}
	return id, nil
}

func branchExists(branch string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Stdout, cmd.Stderr = nil, nil
	return cmd.Run() == nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("%s: %s", name, msg)
		}
		return err
	}
	return nil
}
