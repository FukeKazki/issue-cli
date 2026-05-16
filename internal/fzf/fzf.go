package fzf

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type Result struct {
	Key  string
	Line string
}

func Available() error {
	if _, err := exec.LookPath("fzf"); err != nil {
		return fmt.Errorf("fzf not found in PATH — install via 'brew install fzf' or your package manager")
	}
	return nil
}

func Run(lines []string, opts []string) (Result, error) {
	if err := Available(); err != nil {
		return Result{}, err
	}
	cmd := exec.Command("fzf", opts...)
	cmd.Stdin = strings.NewReader(strings.Join(lines, "\n"))
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			// 130 = user interrupted (Ctrl-C / Esc when no preview-hide bind),
			// 1   = no match.
			code := ee.ExitCode()
			if code == 130 || code == 1 {
				return Result{}, nil
			}
		}
		return Result{}, err
	}

	parts := strings.SplitN(strings.TrimRight(out.String(), "\n"), "\n", 2)
	r := Result{}
	if len(parts) >= 1 {
		r.Key = parts[0]
	}
	if len(parts) >= 2 {
		r.Line = parts[1]
	}
	// When --expect is set, fzf always prints the key on the first line (empty
	// if Enter), and selection on subsequent lines. If user pressed Enter
	// without --expect catching it, parts[0] is the selection itself; we
	// normalize this case in the caller.
	return r, nil
}
