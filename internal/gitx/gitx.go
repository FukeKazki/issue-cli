package gitx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func CheckoutIssue(id int) error {
	branch := fmt.Sprintf("issue/%d", id)
	if branchExists(branch) {
		return run("git", "checkout", branch)
	}
	return run("git", "checkout", "-b", branch)
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
