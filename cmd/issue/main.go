package main

import (
	"fmt"
	"os"

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
	case "_show":
		err = cli.Show(args)
	case "-h", "--help", "help":
		usage()
		return
	default:
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
  issue list [--all] [--status=STATUS]
  issue create [--title TITLE]

Keys in list:
  Enter   checkout branch issue/<id>
  v       toggle detail preview
  e       edit issue
  c       create new issue
  s       change status (then 1-4 or enter)
  d       delete issue (confirm)
  /       filter
  j/k     move down/up (also ↓/↑)
  q/Esc   quit
`)
}
