package main

import (
	"fmt"
	"os"

	"github.com/vibeguard/platform/llm/prompts"
)

func runPrompts(args []string) {
	if len(args) == 0 {
		die("usage: vibeguard prompts <seal|verify> <path>")
	}
	switch args[0] {
	case "seal":
		runPromptsSeal(args[1:])
	case "verify":
		runPromptsVerify(args[1:])
	default:
		die("vibeguard prompts: unknown subcommand %q", args[0])
	}
}

func runPromptsSeal(args []string) {
	if len(args) == 0 {
		die("usage: vibeguard prompts seal <path>")
	}
	path := args[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		die("read %s: %v", path, err)
	}
	out, err := prompts.Seal(raw)
	if err != nil {
		die("seal: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		die("write %s: %v", path, err)
	}
	fmt.Printf("✓ sealed %s\n", path)
}

func runPromptsVerify(args []string) {
	if len(args) == 0 {
		die("usage: vibeguard prompts verify <path>")
	}
	path := args[0]
	raw, err := os.ReadFile(path)
	if err != nil {
		die("read %s: %v", path, err)
	}
	if _, err := prompts.Parse("verify", "0.0.0", raw); err != nil {
		die("verify: %v", err)
	}
	fmt.Printf("✓ %s seal matches\n", path)
}
