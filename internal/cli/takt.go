package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/FukeKazki/issue-cli/internal/config"
	"github.com/FukeKazki/issue-cli/internal/model"
	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
)

func Takt(args []string) error {
	fs := flag.NewFlagSet("takt", flag.ContinueOnError)
	workflow := fs.String("workflow", "", "simple-takt workflow name (or set takt.workflow in .issues/config.yaml)")
	limit := fs.Int("limit", 1, "maximum number of issues to run")
	untilEmpty := fs.Bool("until-empty", false, "keep selecting TODO issues until none remain")
	issueFlag := fs.Int("issue", 0, "run one specific issue instead of selecting from TODO")
	taskFormat := fs.String("task-format", "markdown", "issue show format: markdown|json|yaml")
	dryRun := fs.Bool("dry-run", false, "print issues that would run without mutating metadata")
	continueOnError := fs.Bool("continue-on-error", false, "record failed runs and continue with next issue")
	useWorktree := fs.Bool("worktree", false, "run each issue in an isolated git worktree")
	worktreeDir := fs.String("worktree-dir", "", "base directory for worktrees (or set takt.worktree-dir in .issues/config.yaml)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	limitSet := false
	workflowSet := false
	worktreeDirSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "limit":
			limitSet = true
		case "workflow":
			workflowSet = true
		case "worktree-dir":
			worktreeDirSet = true
		}
	})

	if *limit < 1 {
		return fmt.Errorf("--limit must be a positive integer")
	}
	if *untilEmpty && *issueFlag != 0 {
		return fmt.Errorf("--until-empty cannot be combined with --issue")
	}
	if _, err := output.ParseFormat(*taskFormat); err != nil {
		return fmt.Errorf("--task-format must be one of: markdown, json, yaml")
	}

	repoRoot := taktRepoRoot()
	cfg, err := config.Load(filepath.Join(repoRoot, store.DirName))
	if err != nil {
		return err
	}
	if !workflowSet && cfg.Takt.Workflow != "" {
		*workflow = cfg.Takt.Workflow
	}
	if !worktreeDirSet && cfg.Takt.WorktreeDir != "" {
		*worktreeDir = cfg.Takt.WorktreeDir
	}
	if *workflow == "" {
		return fmt.Errorf("--workflow is required (set via flag or takt.workflow in .issues/config.yaml)")
	}

	const defaultWorktreeDir = "../issue-worktrees"
	if *worktreeDir == "" {
		*worktreeDir = defaultWorktreeDir
	}

	taktBin := os.Getenv("SIMPLE_TAKT_BIN")
	if taktBin == "" {
		taktBin = "simple-takt"
	}
	if !*dryRun {
		if _, err := exec.LookPath(taktBin); err != nil {
			return fmt.Errorf("required command not found: %s", taktBin)
		}
	}

	logDir := os.Getenv("ISSUE_TAKT_LOG_DIR")
	if logDir == "" {
		logDir = ".takt/issue-runner"
	}
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(repoRoot, logDir)
	}

	wtDir := *worktreeDir
	if !filepath.IsAbs(wtDir) {
		wtDir = filepath.Join(repoRoot, wtDir)
	}

	maxRuns := *limit
	if *untilEmpty && !limitSet {
		maxRuns = 0
	}

	if !*dryRun {
		if err := os.MkdirAll(logDir, 0o755); err != nil {
			return err
		}
	}

	s, err := store.New()
	if err != nil {
		return err
	}

	issueBin := os.Getenv("ISSUE_BIN")
	if issueBin == "" {
		issueBin = "issue-cli"
	}

	r := &taktRunner{
		store:       s,
		workflow:    *workflow,
		taskFormat:  *taskFormat,
		taktBin:     taktBin,
		issueBin:    issueBin,
		repoRoot:    repoRoot,
		logDir:      logDir,
		worktreeDir: wtDir,
		useWorktree: *useWorktree,
		dryRun:      *dryRun,
		issueID:     *issueFlag,
	}

	runCount := 0
outerLoop:
	for {
		remaining := 0
		if maxRuns > 0 {
			remaining = maxRuns - runCount
			if remaining <= 0 {
				break
			}
		}

		selectLimit := remaining
		if *untilEmpty && !*dryRun {
			selectLimit = 1
		}

		ids, err := r.selectIssueIDs(selectLimit)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			if runCount == 0 {
				fmt.Println("no runnable TODO issues")
			}
			break
		}

		for _, id := range ids {
			runCount++
			if err := r.runIssue(id); err != nil {
				if !*continueOnError {
					break outerLoop
				}
				fmt.Fprintln(os.Stderr, "continuing after failure because --continue-on-error is set")
			}
		}

		if !*untilEmpty || *dryRun {
			break
		}
	}

	r.printSummary()
	if r.failedCount > 0 {
		return fmt.Errorf("completed with %d failed issue(s)", r.failedCount)
	}
	return nil
}

type taktRunner struct {
	store       *store.Store
	workflow    string
	taskFormat  string
	taktBin     string
	issueBin    string
	repoRoot    string
	logDir      string
	worktreeDir string
	useWorktree bool
	dryRun      bool
	issueID     int

	attemptedIDs []int
	failedCount  int
	succeedCount int
	summaryRows  []taktSummaryRow
}

type taktSummaryRow struct {
	status       string
	id           int
	exitCode     int
	logFile      string
	worktreePath string
}

func (r *taktRunner) selectIssueIDs(limit int) ([]int, error) {
	if r.issueID != 0 {
		for _, aid := range r.attemptedIDs {
			if aid == r.issueID {
				return nil, nil
			}
		}
		return []int{r.issueID}, nil
	}

	issues, err := r.store.LoadAll()
	if err != nil {
		return nil, err
	}

	var ids []int
	for _, iss := range issues {
		if iss.Status != model.StatusTODO {
			continue
		}
		if ws := iss.Metadata["workflow-status"]; ws == "running" || ws == "queued" {
			continue
		}
		attempted := false
		for _, aid := range r.attemptedIDs {
			if aid == iss.ID {
				attempted = true
				break
			}
		}
		if attempted {
			continue
		}
		ids = append(ids, iss.ID)
	}

	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

func (r *taktRunner) setMetadata(id int, kv map[string]string) error {
	iss, err := r.store.Load(id)
	if err != nil {
		return err
	}
	if iss.Metadata == nil {
		iss.Metadata = make(map[string]string, len(kv))
	}
	for k, v := range kv {
		iss.Metadata[k] = v
	}
	return r.store.Save(iss)
}

func (r *taktRunner) runIssue(id int) error {
	r.attemptedIDs = append(r.attemptedIDs, id)

	now := time.Now()
	runID := fmt.Sprintf("%s-issue-%d", now.Format("20060102-150405"), id)
	startedAt := now.Format("2006-01-02T15:04:05-0700")
	logFile := filepath.Join(r.logDir, runID+".log")
	runDir := r.repoRoot
	worktreePath := ""

	if r.dryRun {
		iss, err := r.store.Load(id)
		if err != nil {
			return fmt.Errorf("load issue #%d: %v", id, err)
		}
		if r.useWorktree {
			fmt.Printf("would run #%d: %s (workflow=%s, task-format=%s, worktree=%s)\n",
				id, iss.Title, r.workflow, r.taskFormat,
				filepath.Join(r.worktreeDir, fmt.Sprintf("issue-%d", id)))
		} else {
			fmt.Printf("would run #%d: %s (workflow=%s, task-format=%s)\n",
				id, iss.Title, r.workflow, r.taskFormat)
		}
		return nil
	}

	if r.useWorktree {
		path, err := r.ensureWorktree(id)
		if err != nil {
			finishedAt := time.Now().Format("2006-01-02T15:04:05-0700")
			_ = r.setMetadata(id, map[string]string{
				"workflow":        r.workflow,
				"workflow-status": "failed",
				"run-id":          runID,
				"started-at":      startedAt,
				"finished-at":     finishedAt,
				"exit-code":       "1",
				"log-file":        logFile,
			})
			fmt.Fprintf(os.Stderr, "issue #%d: worktree setup failed: %v\n", id, err)
			r.failedCount++
			r.summaryRows = append(r.summaryRows, taktSummaryRow{"failed", id, 1, logFile, ""})
			return err
		}
		worktreePath = path
		runDir = path
	}

	fmt.Printf("running issue #%d with workflow %s (run-id=%s)\n", id, r.workflow, runID)

	md := map[string]string{
		"workflow":        r.workflow,
		"workflow-status": "running",
		"run-id":          runID,
		"started-at":      startedAt,
		"log-file":        logFile,
	}
	if worktreePath != "" {
		md["worktree-path"] = worktreePath
	}
	if err := r.setMetadata(id, md); err != nil {
		return err
	}

	iss, err := r.store.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	f, _ := output.ParseFormat(r.taskFormat)

	tmpFile, err := os.CreateTemp("", "issue-takt-*.txt")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer os.Remove(tmpName)

	if err := output.WriteIssue(tmpFile, iss, f); err != nil {
		tmpFile.Close()
		return fmt.Errorf("render issue #%d: %v", id, err)
	}
	tmpFile.Close()

	taskInput, err := os.Open(tmpName)
	if err != nil {
		return err
	}
	defer taskInput.Close()

	logF, err := os.Create(logFile)
	if err != nil {
		return fmt.Errorf("create log file: %v", err)
	}

	cmd := exec.Command(r.taktBin, "-w", r.workflow)
	cmd.Dir = runDir
	cmd.Stdin = taskInput
	cmd.Stdout = io.MultiWriter(os.Stdout, logF)
	cmd.Stderr = io.MultiWriter(os.Stderr, logF)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("ISSUE_ID=%d", id),
		"ISSUE_RUN_ID="+runID,
		"ISSUE_WORKFLOW="+r.workflow,
		"ISSUE_BIN="+r.issueBin,
		"ISSUE_REPO_ROOT="+r.repoRoot,
		"ISSUE_RUN_DIR="+runDir,
		"ISSUE_WORKTREE_PATH="+worktreePath,
		"ISSUE_TASK_FORMAT="+r.taskFormat,
		"ISSUE_LOG_FILE="+logFile,
	)

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}
	logF.Close()

	finishedAt := time.Now().Format("2006-01-02T15:04:05-0700")

	if exitCode == 0 {
		_ = r.setMetadata(id, map[string]string{
			"workflow-status": "success",
			"finished-at":    finishedAt,
			"exit-code":      "0",
			"log-file":       logFile,
		})
		fmt.Printf("issue #%d: workflow succeeded\n", id)
		r.succeedCount++
		r.summaryRows = append(r.summaryRows, taktSummaryRow{"success", id, 0, logFile, worktreePath})
		return nil
	}

	_ = r.setMetadata(id, map[string]string{
		"workflow-status": "failed",
		"finished-at":    finishedAt,
		"exit-code":      strconv.Itoa(exitCode),
		"log-file":       logFile,
	})
	fmt.Fprintf(os.Stderr, "issue #%d: workflow failed (exit-code=%d)\n", id, exitCode)
	r.failedCount++
	r.summaryRows = append(r.summaryRows, taktSummaryRow{"failed", id, exitCode, logFile, worktreePath})
	return fmt.Errorf("issue #%d: workflow failed (exit-code=%d)", id, exitCode)
}

func (r *taktRunner) ensureWorktree(id int) (string, error) {
	branch := fmt.Sprintf("issue/%d", id)
	path := filepath.Join(r.worktreeDir, fmt.Sprintf("issue-%d", id))

	if info, err := os.Stat(path); err == nil && info.IsDir() {
		cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("worktree path exists but is not a git worktree: %s", path)
		}
		out, err := exec.Command("git", "-C", path, "branch", "--show-current").Output()
		if err != nil {
			return "", fmt.Errorf("could not determine branch in worktree: %s", path)
		}
		cur := strings.TrimSpace(string(out))
		if cur != branch {
			return "", fmt.Errorf("worktree path is on %s, expected %s: %s", cur, branch, path)
		}
	} else {
		if err := os.MkdirAll(r.worktreeDir, 0o755); err != nil {
			return "", err
		}
		if taktBranchExists(branch) {
			cmd := exec.Command("git", "worktree", "add", path, branch)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("git worktree add: %v", err)
			}
		} else {
			cmd := exec.Command("git", "worktree", "add", "-b", branch, path)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return "", fmt.Errorf("git worktree add -b: %v", err)
			}
		}
	}

	out, err := exec.Command("git", "-C", path, "status", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("git status in worktree: %v", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		return "", fmt.Errorf("worktree is dirty; refusing to run: %s", path)
	}

	r.syncTaktConfig(path)
	return path, nil
}

func (r *taktRunner) syncTaktConfig(targetDir string) {
	srcDir := filepath.Join(r.repoRoot, ".takt")
	if _, err := os.Stat(srcDir); err != nil {
		return
	}
	dstDir := filepath.Join(targetDir, ".takt")
	os.MkdirAll(dstDir, 0o755)

	for _, entry := range []string{"config.yaml", "workflows", "facets"} {
		src := filepath.Join(srcDir, entry)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		dst := filepath.Join(dstDir, entry)
		os.RemoveAll(dst)
		exec.Command("cp", "-R", src, dst).Run()
	}
}

func (r *taktRunner) printSummary() {
	if len(r.summaryRows) == 0 {
		return
	}
	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("- succeeded: %d\n", r.succeedCount)
	fmt.Printf("- failed: %d\n", r.failedCount)
	fmt.Println()
	fmt.Println("Details:")
	for _, row := range r.summaryRows {
		if row.exitCode == 0 {
			fmt.Printf("- #%d: %s\n", row.id, row.status)
		} else {
			fmt.Printf("- #%d: %s (exit-code=%d)\n", row.id, row.status, row.exitCode)
		}
		fmt.Printf("  log: %s\n", row.logFile)
		if row.worktreePath != "" {
			fmt.Printf("  worktree: %s\n", row.worktreePath)
		}
	}
}

func taktRepoRoot() string {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		cwd, _ := os.Getwd()
		return cwd
	}
	return strings.TrimSpace(string(out))
}

func taktBranchExists(branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}
