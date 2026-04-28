// Command vibeguard is the multi-subcommand CLI: validate, generate, lint,
// ir dump, prompts seal, version.
//
// Each subcommand is one file in this package. main.go just wires them.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "validate":
		runValidate(os.Args[2:])
	case "generate":
		runGenerate(os.Args[2:])
	case "lint":
		runLint(os.Args[2:])
	case "ir":
		runIR(os.Args[2:])
	case "prompts":
		runPrompts(os.Args[2:])
	case "version":
		runVersion(os.Args[2:])
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "vibeguard: unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `vibeguard — declaration-driven SDK for vibe coding & infrastructure

USAGE:
    vibeguard <subcommand> [flags]

SUBCOMMANDS:
    validate    Validate a vibeguard.yaml against schema + semantic rules
    ir dump     Print the typed IR parsed from a vibeguard.yaml
    generate    Generate a project (Go, SQL, K8s, OpenAPI) from a declaration
    lint        Run master-prompt analyzers on a Go project (text/json/sarif)
    prompts     Manage sealed LLM prompts (seal, verify)
    version     Print version

DOCS:
    docs/QUICKSTART.md, docs/ARCHITECTURE.md, docs/ROADMAP.md
`)
}
