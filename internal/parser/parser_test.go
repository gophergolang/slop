package parser

import (
	"os"
	"testing"

	"github.com/vibeguard/vibeguard/internal/ir"
)

func TestParseSample(t *testing.T) {
	data, err := os.ReadFile("../../fixtures/sample_vibeguard.yaml")
	if err != nil {
		t.Fatalf("read sample: %v", err)
	}
	app, err := Parse(data)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if app.Metadata.Name != "team-task-saas" {
		t.Errorf("metadata.name: got %q, want team-task-saas", app.Metadata.Name)
	}
	if !app.Global.MultiTenancy.Enabled {
		t.Error("multi_tenancy.enabled: got false, want true")
	}
	if got, want := len(app.Modules), 4; got != want {
		t.Errorf("modules: got %d, want %d", got, want)
	}

	// Find the Task entity and check the step DSL parsed.
	var taskEnt *ir.Entity
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			if ent.Name == "Task" {
				taskEnt = ent
			}
		}
	}
	if taskEnt == nil {
		t.Fatal("Task entity missing")
	}
	if taskEnt.PrimaryKey == nil || taskEnt.PrimaryKey.Name != "id" {
		t.Errorf("Task.PrimaryKey: %+v", taskEnt.PrimaryKey)
	}
	if got := len(taskEnt.CRUD.UpdateFields); got != 6 {
		t.Errorf("Task.CRUD.UpdateFields: got %d, want 6", got)
	}
	if !taskEnt.SoftDelete {
		t.Error("Task.SoftDelete: got false, want true")
	}
	if len(taskEnt.API.CustomEndpoints) != 1 {
		t.Fatalf("Task custom endpoints: got %d, want 1", len(taskEnt.API.CustomEndpoints))
	}
	steps := taskEnt.API.CustomEndpoints[0].Logic.Steps
	if got := len(steps); got != 7 {
		t.Errorf("prioritize steps: got %d, want 7", got)
	}
	// Ensure pointer resolution worked for relationships.
	for _, rel := range taskEnt.Relationships {
		if rel.Resolved == nil {
			t.Errorf("relationship to %q did not resolve", rel.To)
		}
	}
}
