package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/output"
	"github.com/FukeKazki/issue-cli/internal/store"
)

// Metadata dispatches `issue metadata` subcommands. Free-form key/value
// attributes are stored on Issue.Metadata so external tools — automation
// runners or humans — can attach context to an issue without the CLI
// hard-coding a schema.
//
// Subcommands:
//
//	issue metadata <id>                              # show (implicit)
//	issue metadata show <id> [--format ...]
//	issue metadata set <id> key=value [key=value ...] [--format json]
//	issue metadata unset <id> key [key ...] [--format json]
//	issue metadata clear <id> [--format json]
//
// `set` merges into the existing map (overwriting same-key entries). `unset`
// removes named keys; the map is dropped entirely when it becomes empty so
// `metadata:` does not linger in the YAML.
func Metadata(args []string) error {
	if len(args) < 1 {
		return metadataUsage()
	}
	switch args[0] {
	case "set":
		return metadataSet(args[1:])
	case "unset":
		return metadataUnset(args[1:])
	case "clear":
		return metadataClear(args[1:])
	case "show":
		return metadataShow(args[1:])
	default:
		return metadataShow(args)
	}
}

func metadataUsage() error {
	return fmt.Errorf("usage: issue metadata <id>  |  set <id> key=value...  |  unset <id> key...  |  clear <id>")
}

func parseIssueID(arg string) (int, error) {
	raw := strings.TrimPrefix(arg, "#")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid issue id: %q", arg)
	}
	return id, nil
}

// splitPositionalAndFlags partitions `rest` at the first `-`-prefixed token.
// All positional pairs / keys must come before any flag.
func splitPositionalAndFlags(rest []string) (positional, flags []string) {
	for i, a := range rest {
		if strings.HasPrefix(a, "-") {
			return rest[:i], rest[i:]
		}
	}
	return rest, nil
}

func metadataShow(args []string) error {
	if len(args) < 1 {
		return metadataUsage()
	}
	id, err := parseIssueID(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("metadata show", flag.ContinueOnError)
	formatFlag := fs.String("format", "", "output format (json|yaml|markdown); omit for plain text")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}

	if *formatFlag != "" {
		f, err := output.ParseFormat(*formatFlag)
		if err != nil {
			return err
		}
		return output.WriteIssue(os.Stdout, iss, f)
	}

	if len(iss.Metadata) == 0 {
		fmt.Printf("#%d: (no metadata)\n", id)
		return nil
	}
	keys := make([]string, 0, len(iss.Metadata))
	for k := range iss.Metadata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("%s=%s\n", k, iss.Metadata[k])
	}
	return nil
}

func metadataSet(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: issue metadata set <id> key=value [key=value ...] [--format json]")
	}
	id, err := parseIssueID(args[0])
	if err != nil {
		return err
	}
	pairs, flags := splitPositionalAndFlags(args[1:])
	if len(pairs) == 0 {
		return fmt.Errorf("metadata set: at least one key=value pair is required")
	}
	kv := make(map[string]string, len(pairs))
	for _, p := range pairs {
		eq := strings.IndexByte(p, '=')
		if eq <= 0 {
			return fmt.Errorf("metadata set: expected key=value, got %q", p)
		}
		key := strings.TrimSpace(p[:eq])
		if key == "" {
			return fmt.Errorf("metadata set: empty key in %q", p)
		}
		kv[key] = p[eq+1:]
	}

	fs := flag.NewFlagSet("metadata set", flag.ContinueOnError)
	formatFlag := fs.String("format", "", "output format (json|yaml|markdown); omit for plain text")
	if err := fs.Parse(flags); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	if iss.Metadata == nil {
		iss.Metadata = make(map[string]string, len(kv))
	}
	for k, v := range kv {
		iss.Metadata[k] = v
	}
	if err := s.Save(iss); err != nil {
		return err
	}

	if *formatFlag != "" {
		f, err := output.ParseFormat(*formatFlag)
		if err != nil {
			return err
		}
		return output.WriteIssue(os.Stdout, iss, f)
	}
	fmt.Printf("#%d: metadata set (%d keys)\n", id, len(kv))
	return nil
}

func metadataUnset(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: issue metadata unset <id> key [key ...] [--format json]")
	}
	id, err := parseIssueID(args[0])
	if err != nil {
		return err
	}
	keys, flags := splitPositionalAndFlags(args[1:])
	if len(keys) == 0 {
		return fmt.Errorf("metadata unset: at least one key is required")
	}

	fs := flag.NewFlagSet("metadata unset", flag.ContinueOnError)
	formatFlag := fs.String("format", "", "output format (json|yaml|markdown); omit for plain text")
	if err := fs.Parse(flags); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}

	removed := 0
	for _, k := range keys {
		if _, ok := iss.Metadata[k]; ok {
			delete(iss.Metadata, k)
			removed++
		}
	}
	if len(iss.Metadata) == 0 {
		iss.Metadata = nil
	}
	if err := s.Save(iss); err != nil {
		return err
	}

	if *formatFlag != "" {
		f, err := output.ParseFormat(*formatFlag)
		if err != nil {
			return err
		}
		return output.WriteIssue(os.Stdout, iss, f)
	}
	fmt.Printf("#%d: metadata unset (%d removed)\n", id, removed)
	return nil
}

func metadataClear(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: issue metadata clear <id> [--format json]")
	}
	id, err := parseIssueID(args[0])
	if err != nil {
		return err
	}
	fs := flag.NewFlagSet("metadata clear", flag.ContinueOnError)
	formatFlag := fs.String("format", "", "output format (json|yaml|markdown); omit for plain text")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	s, err := store.New()
	if err != nil {
		return err
	}
	iss, err := s.Load(id)
	if err != nil {
		return fmt.Errorf("load issue #%d: %v", id, err)
	}
	iss.Metadata = nil
	if err := s.Save(iss); err != nil {
		return err
	}

	if *formatFlag != "" {
		f, err := output.ParseFormat(*formatFlag)
		if err != nil {
			return err
		}
		return output.WriteIssue(os.Stdout, iss, f)
	}
	fmt.Printf("#%d: metadata cleared\n", id)
	return nil
}
