package golang

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vibeguard/vibeguard/internal/ir"
	"github.com/vibeguard/vibeguard/internal/render"
)

// minimal IR with one node-backed endpoint, just enough for emitNodes.
func sampleAppWithNode() *ir.Application {
	mod := ir.Module{Name: "tasks", Type: ir.ModuleBusiness}
	ent := &ir.Entity{
		Name:  "Task",
		Table: "tasks",
		Fields: []*ir.Field{
			{Name: "id", Type: ir.FieldUUID, Primary: true},
		},
		API: ir.API{
			BasePath: "/api/v1/tasks",
			CustomEndpoints: []ir.CustomEndpoint{
				{
					Path:         "/api/v1/tasks/:id/prioritize",
					Method:       "POST",
					Node:         "tasks.Prioritize",
					Request:      "PrioritizeRequest",
					Response:     "PrioritizeResponse",
					AuthRequired: true,
					RolesAllowed: []string{"admin", "member"},
				},
			},
		},
	}
	mod.Entities = []*ir.Entity{ent}
	ent.Module = &mod
	ent.PrimaryKey = ent.Fields[0]
	app := &ir.Application{
		APIVersion: "vibeguard.dev/v1",
		Kind:       "Application",
		Metadata:   ir.Metadata{Name: "demo", Version: "1.0.0"},
		Global:     ir.Global{MultiTenancy: ir.MultiTenancy{Enabled: true, TenantIDField: "tenant_id"}},
		Modules:    []ir.Module{mod},
	}
	app.Modules[0].Entities[0].Module = &app.Modules[0]
	return app
}

func TestEmitNodesProducesExpectedFiles(t *testing.T) {
	app := sampleAppWithNode()
	b := New("github.com/example/demo")
	mod := app.Modules[0]
	ent := mod.Entities[0]
	files, err := b.emitNodes(app, mod, ent, "tasks", "internal/tasks")
	if err != nil {
		t.Fatalf("emitNodes: %v", err)
	}
	want := map[string]bool{
		"internal/tasks/nodes/nodes.go":          true,  // KeepIfExists
		"internal/tasks/nodes/task_prioritize.go": true, // KeepIfExists
		"internal/tasks/task_node_handler.go":    false, // overwritten
	}
	got := map[string]bool{}
	for _, f := range files {
		got[f.Path] = f.KeepIfExists
	}
	for path, keep := range want {
		v, ok := got[path]
		if !ok {
			t.Errorf("missing file %s in output", path)
			continue
		}
		if v != keep {
			t.Errorf("%s: KeepIfExists got %v want %v", path, v, keep)
		}
	}
}

func TestNodeWrapperContainsAuthAndRoleChecks(t *testing.T) {
	app := sampleAppWithNode()
	b := New("github.com/example/demo")
	mod := app.Modules[0]
	ent := mod.Entities[0]
	files, err := b.emitNodes(app, mod, ent, "tasks", "internal/tasks")
	if err != nil {
		t.Fatalf("emitNodes: %v", err)
	}
	var wrapper string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "task_node_handler.go") {
			wrapper = string(f.Content)
		}
	}
	if wrapper == "" {
		t.Fatal("wrapper not emitted")
	}
	for _, expect := range []string{
		`http.StatusUnauthorized`,
		`nodeRoleAllowed`,
		`h.nodes.Prioritize`,
		`RegisterTaskNodeRoutes`,
	} {
		if !strings.Contains(wrapper, expect) {
			t.Errorf("wrapper missing %q\n--- wrapper ---\n%s", expect, wrapper)
		}
	}
}

// TestRunPreservesStubAcrossRegenerate exercises the Engine end-to-end:
// first generate scaffolds the stub; second generate must NOT overwrite it.
func TestRunPreservesStubAcrossRegenerate(t *testing.T) {
	app := sampleAppWithNode()
	b := New("github.com/example/demo")
	dir := t.TempDir()

	first := &render.Engine{Root: dir, Mode: render.ModeWrite, Backends: []render.Backend{b}}
	if _, err := first.Run(app); err != nil {
		t.Fatalf("first run: %v", err)
	}
	stubPath := filepath.Join(dir, "internal/tasks/nodes/task_prioritize.go")
	original, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatalf("read stub: %v", err)
	}

	dev := append(original, []byte("\n// developer wrote this\n")...)
	if err := os.WriteFile(stubPath, dev, 0o644); err != nil {
		t.Fatal(err)
	}

	second := &render.Engine{Root: dir, Mode: render.ModeWrite, Backends: []render.Backend{b}}
	report, err := second.Run(app)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}

	after, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "developer wrote this") {
		t.Errorf("stub overwritten on regenerate; report.Skipped=%v", report.Skipped)
	}
	if !contains(report.Skipped, "internal/tasks/nodes/task_prioritize.go") {
		t.Errorf("expected stub in Skipped list; got %v", report.Skipped)
	}
	wrapperPath := filepath.Join(dir, "internal/tasks/task_node_handler.go")
	if _, err := os.Stat(wrapperPath); err != nil {
		t.Errorf("wrapper not written: %v", err)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
