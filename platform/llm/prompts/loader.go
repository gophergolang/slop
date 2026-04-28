// Package prompts is the sealed-prompt loader.
//
// Prompts live on disk as <name>/<semver>.md with YAML frontmatter:
//
//	---
//	model_default: claude-haiku-4-5
//	inputs:
//	  type: object
//	  required: [task]
//	output_schema:
//	  type: object
//	  required: [priority, reason]
//	sha256: 4c9f5e...
//	---
//	You are an expert agile coach. Prioritize this task (1-10).
//	...
//
// At load time the loader hashes the body (everything after the closing
// `---`) and refuses if the manifest sha256 does not match. This guarantees
// prompts cannot be silently mutated without an explicit version bump in the
// declaration that references them.
//
// The CLI command `vibeguard prompts seal <path>` rewrites the sha256 field
// to match the current body — that's the human gesture that says "I meant
// to change this."
package prompts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the parsed YAML header of a sealed prompt.
type Frontmatter struct {
	ModelDefault string         `yaml:"model_default"`
	Inputs       map[string]any `yaml:"inputs"`
	OutputSchema map[string]any `yaml:"output_schema"`
	SHA256       string         `yaml:"sha256"`
}

// Prompt is a loaded, sealed, ready-to-render prompt.
type Prompt struct {
	Name        string
	Version     string
	Frontmatter Frontmatter
	Body        string
}

// Parse parses + verifies a single prompt file's contents. ErrSealMismatch
// is returned if the body's sha256 differs from the frontmatter declaration.
func Parse(name, version string, raw []byte) (*Prompt, error) {
	fm, body, err := splitFrontmatter(raw)
	if err != nil {
		return nil, fmt.Errorf("prompts: %s/%s: %w", name, version, err)
	}
	if fm.SHA256 == "" {
		return nil, fmt.Errorf("prompts: %s/%s: frontmatter is missing sha256 (run `vibeguard prompts seal`)", name, version)
	}
	got := sha256Hex([]byte(body))
	if got != fm.SHA256 {
		return nil, fmt.Errorf("%w: %s/%s declared %s, got %s", ErrSealMismatch, name, version, fm.SHA256, got)
	}
	return &Prompt{
		Name:        name,
		Version:     version,
		Frontmatter: fm,
		Body:        body,
	}, nil
}

// Seal computes the sha256 of body and rewrites raw with that value in the
// frontmatter, returning the updated bytes. Used by `vibeguard prompts seal`.
func Seal(raw []byte) ([]byte, error) {
	fm, body, err := splitFrontmatter(raw)
	if err != nil {
		return nil, err
	}
	fm.SHA256 = sha256Hex([]byte(body))
	return assemble(fm, body)
}

// ErrSealMismatch indicates a prompt body has changed without a corresponding
// version bump.
var ErrSealMismatch = errors.New("prompts: seal mismatch")

func splitFrontmatter(raw []byte) (Frontmatter, string, error) {
	const sep = "---"
	s := string(raw)
	if !strings.HasPrefix(s, sep) {
		return Frontmatter{}, "", errors.New("missing leading --- frontmatter")
	}
	rest := s[len(sep):]
	rest = strings.TrimLeft(rest, "\r\n")
	idx := strings.Index(rest, "\n"+sep)
	if idx < 0 {
		return Frontmatter{}, "", errors.New("missing closing --- frontmatter")
	}
	header := rest[:idx]
	body := rest[idx+len("\n"+sep):]
	body = strings.TrimLeft(body, "\r\n")
	var fm Frontmatter
	if err := yaml.Unmarshal([]byte(header), &fm); err != nil {
		return Frontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return fm, body, nil
}

func assemble(fm Frontmatter, body string) ([]byte, error) {
	header, err := yaml.Marshal(fm)
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.Write(header)
	b.WriteString("---\n")
	b.WriteString(body)
	return []byte(b.String()), nil
}

func sha256Hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
