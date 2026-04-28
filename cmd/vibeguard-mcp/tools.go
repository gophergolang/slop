package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vibeguard/vibeguard/internal/lint"
	"github.com/vibeguard/vibeguard/internal/parser"
	"github.com/vibeguard/vibeguard/internal/render"
	"github.com/vibeguard/vibeguard/internal/render/golang"
	"github.com/vibeguard/vibeguard/internal/render/k8s"
	"github.com/vibeguard/vibeguard/internal/render/openapi"
	"github.com/vibeguard/vibeguard/internal/render/sql"
	"github.com/vibeguard/vibeguard/internal/validate"
)

type toolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func toolDescriptors() []toolDescriptor {
	return []toolDescriptor{
		{
			Name:        "validate_declaration",
			Description: "Validate a vibeguard.yaml declaration. Returns parse errors, semantic rule findings, and warnings.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"yaml"},
				"properties": map[string]any{
					"yaml": map[string]any{"type": "string", "description": "Full text of the declaration"},
				},
			},
		},
		{
			Name:        "generate_project",
			Description: "Render a complete Go service (handlers + repositories + main + migrations + K8s manifests + OpenAPI) from a declaration.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"yaml", "out_dir"},
				"properties": map[string]any{
					"yaml":        map[string]any{"type": "string"},
					"out_dir":     map[string]any{"type": "string", "description": "Output directory (created if needed)"},
					"module_path": map[string]any{"type": "string", "description": "Go module path (default: github.com/example/<app-name>)"},
				},
			},
		},
		{
			Name:        "lint_project",
			Description: "Run master-prompt analyzers (VG001-VG006) on a Go project rooted at root_dir. Returns SARIF findings.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"root_dir"},
				"properties": map[string]any{
					"root_dir": map[string]any{"type": "string", "description": "Root directory of the Go project to lint"},
					"format":   map[string]any{"type": "string", "enum": []string{"text", "json", "sarif"}, "default": "text"},
				},
			},
		},
	}
}

func (s *server) handleToolsCall(req rpcRequest) {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.respondError(req.ID, -32602, "invalid params: "+err.Error())
		return
	}
	switch p.Name {
	case "validate_declaration":
		s.toolValidate(req.ID, p.Arguments)
	case "generate_project":
		s.toolGenerate(req.ID, p.Arguments)
	case "lint_project":
		s.toolLint(req.ID, p.Arguments)
	default:
		s.respondError(req.ID, -32601, "unknown tool: "+p.Name)
	}
}

func (s *server) toolValidate(id json.RawMessage, args json.RawMessage) {
	var a struct {
		YAML string `json:"yaml"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		s.respondError(id, -32602, err.Error())
		return
	}
	app, err := parser.Parse([]byte(a.YAML))
	if err != nil {
		s.respond(id, mcpResult(fmt.Sprintf("PARSE ERROR: %v", err)))
		return
	}
	issues := validate.Run(app)
	if len(issues) == 0 {
		s.respond(id, mcpResult(fmt.Sprintf("✓ declaration is valid (apiVersion=%s, modules=%d)", app.APIVersion, len(app.Modules))))
		return
	}
	var b bytes.Buffer
	errors := 0
	warnings := 0
	for _, iss := range issues {
		mark := "WARN"
		if iss.Severity == validate.SeverityError {
			mark = "ERROR"
			errors++
		} else {
			warnings++
		}
		fmt.Fprintf(&b, "[%s] %s — %s (%s)\n", mark, iss.Rule, iss.Message, iss.Path)
	}
	fmt.Fprintf(&b, "\n%d errors, %d warnings", errors, warnings)
	s.respond(id, mcpResult(b.String()))
}

func (s *server) toolGenerate(id json.RawMessage, args json.RawMessage) {
	var a struct {
		YAML       string `json:"yaml"`
		OutDir     string `json:"out_dir"`
		ModulePath string `json:"module_path"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		s.respondError(id, -32602, err.Error())
		return
	}
	app, err := parser.Parse([]byte(a.YAML))
	if err != nil {
		s.respond(id, mcpResult(fmt.Sprintf("parse: %v", err)))
		return
	}
	if validate.HasErrors(validate.Run(app)) {
		s.respond(id, mcpResult("declaration has errors; refusing to generate. Run validate_declaration to see them."))
		return
	}
	mod := a.ModulePath
	if mod == "" {
		mod = "github.com/example/" + app.Metadata.Name
	}
	if err := os.MkdirAll(a.OutDir, 0o755); err != nil {
		s.respond(id, mcpResult(err.Error()))
		return
	}
	engine := &render.Engine{
		Root: a.OutDir,
		Mode: render.ModeWrite,
		Backends: []render.Backend{
			golang.New(mod),
			sql.New(),
			k8s.New("ghcr.io/example/"+app.Metadata.Name, app.Metadata.Version),
			openapi.New(),
		},
	}
	report, err := engine.Run(app)
	if err != nil {
		s.respond(id, mcpResult(err.Error()))
		return
	}
	var b bytes.Buffer
	fmt.Fprintf(&b, "✓ generated %d files into %s\n\n", len(report.FilesWritten), a.OutDir)
	for _, br := range report.Backends {
		fmt.Fprintf(&b, "  %-8s %d files (%d bytes)\n", br.Name, br.NumFiles, br.NumBytes)
	}
	fmt.Fprintf(&b, "\nNext: cd %s && go mod tidy && go build ./...", a.OutDir)
	s.respond(id, mcpResult(b.String()))
}

func (s *server) toolLint(id json.RawMessage, args json.RawMessage) {
	var a struct {
		RootDir string `json:"root_dir"`
		Format  string `json:"format"`
	}
	if err := json.Unmarshal(args, &a); err != nil {
		s.respondError(id, -32602, err.Error())
		return
	}
	if a.Format == "" {
		a.Format = "text"
	}
	wd, _ := os.Getwd()
	defer os.Chdir(wd)
	if err := os.Chdir(a.RootDir); err != nil {
		s.respond(id, mcpResult(err.Error()))
		return
	}
	var buf bytes.Buffer
	findings, err := lint.Run(lint.Options{
		Patterns: []string{"./..."},
		Format:   lint.Format(a.Format),
		Out:      &buf,
	})
	if err != nil {
		s.respond(id, mcpResult(err.Error()))
		return
	}
	if buf.Len() == 0 {
		fmt.Fprintf(&buf, "%d findings\n", len(findings))
	}
	s.respond(id, mcpResult(buf.String()))
	_ = filepath.Separator
}

func mcpResult(text string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
	}
}
