package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/vibeguard/vibeguard/internal/parser"
	"github.com/vibeguard/vibeguard/internal/render"
	"github.com/vibeguard/vibeguard/internal/render/golang"
	"github.com/vibeguard/vibeguard/internal/render/k8s"
	"github.com/vibeguard/vibeguard/internal/render/nextjs"
	"github.com/vibeguard/vibeguard/internal/render/openapi"
	"github.com/vibeguard/vibeguard/internal/render/sql"
	"github.com/vibeguard/vibeguard/internal/validate"
)

func runGenerate(args []string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	file := fs.String("f", "vibeguard.yaml", "path to declaration")
	out := fs.String("o", "out", "output directory")
	module := fs.String("module", "", "Go module path (default: github.com/example/<app-name>)")
	image := fs.String("image", "", "container image (default: ghcr.io/example/<app-name>)")
	tag := fs.String("tag", "", "container image tag (default: app version)")
	dryRun := fs.Bool("dry-run", false, "list files that would be written")
	diff := fs.Bool("diff", false, "diff against existing files")
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
	if validate.HasErrors(issues) {
		fmt.Fprintln(os.Stderr, "vibeguard: declaration has errors; fix them before generating")
		for _, iss := range issues {
			if iss.Severity == validate.SeverityError {
				fmt.Fprintf(os.Stderr, "  [ERR] %s  %s  (%s)\n", iss.Rule, iss.Message, iss.Path)
			}
		}
		os.Exit(1)
	}

	mod := *module
	if mod == "" {
		mod = "github.com/example/" + app.Metadata.Name
	}
	img := *image
	if img == "" {
		img = "ghcr.io/example/" + app.Metadata.Name
	}
	imgTag := *tag
	if imgTag == "" {
		imgTag = app.Metadata.Version
		if imgTag == "" {
			imgTag = "dev"
		}
	}

	mode := render.ModeWrite
	switch {
	case *dryRun:
		mode = render.ModeDryRun
	case *diff:
		mode = render.ModeDiff
	}

	engine := &render.Engine{
		Root: *out,
		Mode: mode,
		Backends: []render.Backend{
			golang.New(mod),
			sql.New(),
			k8s.New(img, imgTag),
			openapi.New(),
			nextjs.New(),
		},
	}
	report, err := engine.Run(app)
	if err != nil {
		die("render: %v", err)
	}

	if mode == render.ModeWrite {
		fmt.Printf("✓ generated %d files into %s/\n", len(report.FilesWritten), *out)
		for _, b := range report.Backends {
			fmt.Printf("    %-8s %d files (%d bytes)\n", b.Name, b.NumFiles, b.NumBytes)
		}
		fmt.Printf("\nNext: cd %s && go mod tidy && go build ./...\n", *out)
	}
}
