package validate

import (
	"strings"
	"testing"

	"github.com/vibeguard/vibeguard/internal/parser"
)

func TestSampleIsClean(t *testing.T) {
	data, err := readFixture("sample_vibeguard.yaml")
	if err != nil {
		t.Fatal(err)
	}
	app, err := parser.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	issues := Run(app)
	if HasErrors(issues) {
		t.Errorf("sample declaration has errors: %+v", issues)
	}
}

func TestCRUDUpdateFieldsExistRule(t *testing.T) {
	yaml := `apiVersion: vibeguard.dev/v1
kind: Application
metadata: { name: bad, version: "1.0" }
spec:
  modules:
    - name: thing
      type: business
      entities:
        - name: Thing
          table: things
          fields:
            - { name: id, type: uuid, primary: true }
            - { name: name, type: string }
          crud:
            create: true
            read: true
            update: [name, missing_field]
`
	app, err := parser.Parse([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	issues := Run(app)
	found := false
	for _, iss := range issues {
		if iss.Rule == "VG-VAL-004" && strings.Contains(iss.Message, "missing_field") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VG-VAL-004 finding for missing_field; got %+v", issues)
	}
}

func TestRelationshipResolutionRule(t *testing.T) {
	yaml := `apiVersion: vibeguard.dev/v1
kind: Application
metadata: { name: bad, version: "1.0" }
spec:
  modules:
    - name: thing
      type: business
      entities:
        - name: Thing
          table: things
          fields:
            - { name: id, type: uuid, primary: true }
          relationships:
            - { to: Nonexistent, type: belongs_to, foreign_key: x }
`
	app, _ := parser.Parse([]byte(yaml))
	issues := Run(app)
	found := false
	for _, iss := range issues {
		if iss.Rule == "VG-VAL-005" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VG-VAL-005 finding for unresolved relationship; got %+v", issues)
	}
}

func TestParentsExistRule(t *testing.T) {
	yaml := `apiVersion: vibeguard.dev/v1
kind: Application
metadata: { name: bad, version: "1.0" }
spec:
  modules:
    - name: thing
      type: business
      entities:
        - name: Thing
          table: things
          parents: [Ghost]
          fields:
            - { name: id, type: uuid, primary: true }
          crud: { create: true, read: true }
`
	app, _ := parser.Parse([]byte(yaml))
	issues := Run(app)
	found := false
	for _, iss := range issues {
		if iss.Rule == "VG-VAL-010" && strings.Contains(iss.Message, "Ghost") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VG-VAL-010 for missing parent; got %+v", issues)
	}
}

func TestNoParentCycleRule(t *testing.T) {
	yaml := `apiVersion: vibeguard.dev/v1
kind: Application
metadata: { name: bad, version: "1.0" }
spec:
  modules:
    - name: thing
      type: business
      entities:
        - name: A
          table: a
          parents: [B]
          fields: [{ name: id, type: uuid, primary: true }]
          crud: { read: true }
        - name: B
          table: b
          parents: [A]
          fields: [{ name: id, type: uuid, primary: true }]
          crud: { read: true }
`
	app, _ := parser.Parse([]byte(yaml))
	issues := Run(app)
	found := false
	for _, iss := range issues {
		if iss.Rule == "VG-VAL-011" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VG-VAL-011 for parent cycle; got %+v", issues)
	}
}

func TestTenantRootConsistentRule(t *testing.T) {
	yaml := `apiVersion: vibeguard.dev/v1
kind: Application
metadata: { name: bad, version: "1.0" }
spec:
  global:
    multi_tenancy: { enabled: true, tenant_id_field: tenant_id, isolation: row }
  modules:
    - name: thing
      type: business
      entities:
        - name: GlobalCatalog
          table: global_catalog
          fields:
            - { name: id, type: uuid, primary: true }
          crud: { read: true }
        - name: TenantThing
          table: tenant_things
          parents: [GlobalCatalog]
          fields:
            - { name: id, type: uuid, primary: true }
            - { name: tenant_id, type: uuid }
          crud: { read: true }
`
	app, _ := parser.Parse([]byte(yaml))
	issues := Run(app)
	found := false
	for _, iss := range issues {
		if iss.Rule == "VG-VAL-012" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VG-VAL-012 for tenant boundary crossing; got %+v", issues)
	}
}

func readFixture(name string) ([]byte, error) {
	for _, candidate := range []string{
		"../../fixtures/" + name,
		"../fixtures/" + name,
	} {
		if data, err := readFile(candidate); err == nil {
			return data, nil
		}
	}
	return nil, ErrFixtureNotFound
}
