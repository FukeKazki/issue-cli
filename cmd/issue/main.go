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
	case "create", "new":
		err = cli.Create(args)
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
  issue                                  on issue/<id>: show that issue; otherwise: open list TUI
  issue <id> | issue #<id>               show issue detail
  issue show <id> [--format markdown|yaml|json]
  issue list [--all] [--status=STATUS] [--format json]
  issue next [--format json]             print the next TODO issue as JSON ({"issue": null} if none)
  issue create [--title TITLE]
  issue edit <id> --status STATUS        update status (case-insensitive; accepts TODO/done/in-progress/review etc.)
  issue metadata <id>                    show free-form metadata attached to an issue
  issue metadata set <id> k=v [k=v ...]  merge key/value pairs into the issue's metadata map
  issue metadata unset <id> k [k ...]    remove keys from the metadata map
  issue metadata clear <id>              drop the entire metadata map

Keys in list:
  Enter   show issue detail (q/Esc to return)
  c       git checkout issue/<id>
  n       create new issue
  e       edit issue
  s       change status (then 1-4 or enter)
  d       delete issue (confirm)
  v       toggle detail preview
  /       filter
  j/k     move down/up (also ↓/↑)
  q/Esc   quit
`)
}
