package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vibeguard/vibeguard/internal/parser"
	"github.com/vibeguard/vibeguard/internal/validate"
)

func runValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	file := fs.String("f", "vibeguard.yaml", "path to declaration")
	_ = fs.Parse(args)

	data, err := os.ReadFile(*file)
	if err != nil {
		die("read %s: %v", *file, err)
	}
	app, err := parser.Parse(data)
	if err != nil {
		die("parse: %v", err)
	}
	issues := validate.Run(app)
	if len(issues) == 0 {
		fmt.Println("✓ declaration is valid")
		return
	}
	errors := 0
	warnings := 0
	for _, iss := range issues {
		mark := "WARN"
		if iss.Severity == validate.SeverityError {
			mark = "ERR "
			errors++
		} else if iss.Severity == validate.SeverityWarning {
			warnings++
		}
		fmt.Printf("[%s] %s  %s\n           %s\n", mark, iss.Rule, iss.Message, iss.Path)
	}
	fmt.Printf("\n%d errors, %d warnings\n", errors, warnings)
	if errors > 0 {
		os.Exit(1)
	}
}

func die(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "vibeguard: "+format+"\n", args...)
	os.Exit(1)
}
