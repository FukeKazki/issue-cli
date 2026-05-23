package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/FukeKazki/issue-cli/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		if err := cli.Default(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}
	sub := os.Args[1]
	args := os.Args[2:]

	var err error
	switch sub {
	case "list", "ls":
		err = cli.List(args)
	case "new":
		err = cli.New(args)
	case "show":
		err = cli.Show(args)
	case "next":
		err = cli.Next(args)
	case "edit":
		err = cli.Edit(args)
	case "metadata", "meta":
		err = cli.Metadata(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
		if _, e := strconv.Atoi(strings.TrimPrefix(sub, "#")); e == nil {
			err = cli.Show(os.Args[1:])
			break
		}
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", sub)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `Usage:
  issue-cli                                  on issue/<id>: show that issue; otherwise: open list TUI
  issue-cli <id> | issue-cli #<id>           show issue detail
  issue-cli show <id> [--format markdown|yaml|json]
  issue-cli list [--all] [--status=STATUS] [--format json]
  issue-cli next [--format json]             print the next TODO issue as JSON ({"issue": null} if none)
  issue-cli new [--title TITLE]
  issue-cli edit <id> --status STATUS        update status (case-insensitive; accepts TODO/done/in-progress/review etc.)
  issue-cli metadata <id>                    show free-form metadata attached to an issue
  issue-cli metadata set <id> k=v [k=v ...]  merge key/value pairs into the issue's metadata map
  issue-cli metadata unset <id> k [k ...]    remove keys from the metadata map
  issue-cli metadata clear <id>              drop the entire metadata map

Keys in list:
  Enter   show issue detail (q/Esc to return)
  c       git checkout issue/<id>
  n       new issue
  e       edit issue
  s       change status (then 1-4 or enter)
  d       delete issue (confirm)
  v       toggle detail preview
  /       filter
  j/k     move down/up (also ↓/↑)
  q/Esc   quit
`)
}
