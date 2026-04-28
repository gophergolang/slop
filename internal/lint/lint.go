// Package lint runs the master-prompt-derived static analyzers against a
// vibeguard project and emits findings in human, JSON, or SARIF form.
//
// The analyzers themselves live in internal/rules/code (one file per rule).
// This package is the harness: it builds a multichecker, walks the package
// graph, collects diagnostics, and formats them.
package lint

import (
	"encoding/json"
	"fmt"
	"go/types"
	"io"
	"sort"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/vibeguard/vibeguard/internal/rules/code"
)

// Format selects the output shape.
type Format string

const (
	FormatText  Format = "text"
	FormatJSON  Format = "json"
	FormatSARIF Format = "sarif"
)

// Options configures Run.
type Options struct {
	Patterns []string // package patterns, e.g. "./..."
	Format   Format
	Out      io.Writer
}

// Finding is one diagnostic.
type Finding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Col      int    `json:"col"`
	Message  string `json:"message"`
}

// Run loads packages, runs every analyzer, formats findings.
func Run(opts Options) ([]Finding, error) {
	if opts.Format == "" {
		opts.Format = FormatText
	}
	if opts.Out == nil {
		opts.Out = nopWriter{}
	}
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedDeps | packages.NeedImports,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, opts.Patterns...)
	if err != nil {
		return nil, fmt.Errorf("load packages: %w", err)
	}

	analyzers := code.AllAnalyzers()
	var findings []Finding
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			// Surface load errors as findings so partial projects still get
			// useful diagnostics for the parts that *do* load.
			for _, e := range pkg.Errors {
				findings = append(findings, Finding{
					RuleID:   "VG-LOAD",
					Severity: "warning",
					File:     "(load)",
					Message:  e.Error(),
				})
			}
			continue
		}
		for _, a := range analyzers {
			collected := []Finding{}
			pass := buildPass(a, pkg, func(d analysis.Diagnostic) {
				pos := pkg.Fset.Position(d.Pos)
				collected = append(collected, Finding{
					RuleID:   a.Name,
					Severity: "warning",
					File:     pos.Filename,
					Line:     pos.Line,
					Col:      pos.Column,
					Message:  d.Message,
				})
			})
			result, err := a.Run(pass)
			if err != nil {
				findings = append(findings, Finding{
					RuleID:   a.Name,
					Severity: "warning",
					Message:  fmt.Sprintf("analyzer %s failed: %v", a.Name, err),
				})
				continue
			}
			_ = result
			findings = append(findings, collected...)
		}
	}

	sort.Slice(findings, func(i, j int) bool {
		if findings[i].File != findings[j].File {
			return findings[i].File < findings[j].File
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	if err := write(opts.Out, opts.Format, findings); err != nil {
		return findings, err
	}
	return findings, nil
}

func write(w io.Writer, f Format, findings []Finding) error {
	switch f {
	case FormatText:
		for _, fi := range findings {
			fmt.Fprintf(w, "%s:%d:%d %s [%s] %s\n", fi.File, fi.Line, fi.Col, fi.Severity, fi.RuleID, fi.Message)
		}
	case FormatJSON:
		return json.NewEncoder(w).Encode(findings)
	case FormatSARIF:
		return writeSARIF(w, findings)
	}
	return nil
}

func buildPass(a *analysis.Analyzer, pkg *packages.Package, report func(analysis.Diagnostic)) *analysis.Pass {
	return &analysis.Pass{
		Analyzer:         a,
		Fset:             pkg.Fset,
		Files:            pkg.Syntax,
		Pkg:              pkg.Types,
		TypesInfo:        pkg.TypesInfo,
		ResultOf:         map[*analysis.Analyzer]any{},
		Report:           report,
		ImportObjectFact: func(obj types.Object, fact analysis.Fact) bool { return false },
	}
}

type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }
