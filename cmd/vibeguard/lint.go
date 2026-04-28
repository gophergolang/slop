package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vibeguard/vibeguard/internal/lint"
)

func runLint(args []string) {
	fs := flag.NewFlagSet("lint", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text | json | sarif")
	out := fs.String("out", "", "write output to file (default: stdout)")
	_ = fs.Parse(args)

	patterns := fs.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	var w *os.File = os.Stdout
	if *out != "" {
		f, err := os.Create(*out)
		if err != nil {
			die("create %s: %v", *out, err)
		}
		defer f.Close()
		w = f
	}

	findings, err := lint.Run(lint.Options{
		Patterns: patterns,
		Format:   lint.Format(*format),
		Out:      w,
	})
	if err != nil {
		die("lint: %v", err)
	}
	if *format == "text" {
		fmt.Printf("\n%d findings\n", len(findings))
	}
	for _, f := range findings {
		if f.Severity == "error" {
			os.Exit(1)
		}
	}
}
