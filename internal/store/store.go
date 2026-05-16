package store

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/FukeKazki/issue-cli/internal/model"
	"gopkg.in/yaml.v3"
)

const DirName = ".issues"

type Store struct {
	Dir string
}

func New() (*Store, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	return &Store{Dir: filepath.Join(root, DirName)}, nil
}

func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	cwd, cerr := os.Getwd()
	if cerr != nil {
		return "", cerr
	}
	return cwd, nil
}

func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.Dir, 0o755)
}

func (s *Store) Path(id int) string {
	return filepath.Join(s.Dir, strconv.Itoa(id)+".yaml")
}

func (s *Store) Load(id int) (*model.Issue, error) {
	return readIssue(s.Path(id))
}

func (s *Store) LoadAll() ([]model.Issue, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var issues []model.Issue
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		p := filepath.Join(s.Dir, e.Name())
		iss, err := readIssue(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", p, err)
			continue
		}
		issues = append(issues, *iss)
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].ID < issues[j].ID })
	return issues, nil
}

func readIssue(path string) (*model.Issue, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var iss model.Issue
	if err := yaml.Unmarshal(b, &iss); err != nil {
		return nil, err
	}
	return &iss, nil
}

func (s *Store) NextID() (int, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 1, nil
		}
		return 0, err
	}
	max := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".yaml")
		n, err := strconv.Atoi(name)
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

func (s *Store) Delete(id int) error {
	if err := os.Remove(s.Path(id)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("issue #%d not found", id)
		}
		return err
	}
	return nil
}

func (s *Store) Save(iss *model.Issue) error {
	if err := validate(iss); err != nil {
		return err
	}
	if err := s.EnsureDir(); err != nil {
		return err
	}
	now := time.Now()
	if iss.CreatedAt.IsZero() {
		iss.CreatedAt = now
	}
	iss.UpdatedAt = now

	b, err := yaml.Marshal(iss)
	if err != nil {
		return err
	}
	path := s.Path(iss.ID)
	tmp, err := os.CreateTemp(s.Dir, ".tmp-*.yaml")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

func validate(iss *model.Issue) error {
	if iss.ID <= 0 {
		return fmt.Errorf("issue id must be positive (got %d)", iss.ID)
	}
	if strings.TrimSpace(iss.Title) == "" {
		return fmt.Errorf("title must not be empty")
	}
	if _, ok := model.ParseStatus(string(iss.Status)); !ok {
		return fmt.Errorf("invalid status %q", iss.Status)
	}
	return nil
}
