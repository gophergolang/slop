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
