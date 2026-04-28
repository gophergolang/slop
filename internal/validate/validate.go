// Package validate is the semantic validator for vibeguard declarations.
//
// JSON-Schema-level validation (required fields, type checks, enum values) is
// the parser's first pass — anything that's structurally invalid never reaches
// the IR. This package is the *second* pass: cross-field rules that the
// schema can't express.
//
// Each rule is a small function over (*ir.Application). New rules are added
// by appending to the registry — there's no plug-in machinery on purpose.
package validate

import (
	"fmt"
	"strings"

	"github.com/vibeguard/vibeguard/internal/ir"
)

// Severity classifies a finding.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Issue is one diagnostic.
type Issue struct {
	Rule     string
	Severity Severity
	Message  string
	Path     string // dotted path within the declaration
}

// Run executes every registered rule against app and returns the union of
// findings. Callers display them in order; errors block code generation,
// warnings do not.
func Run(app *ir.Application) []Issue {
	var out []Issue
	for _, rule := range allRules {
		out = append(out, rule(app)...)
	}
	return out
}

// HasErrors reports whether any issue in iss has Severity == SeverityError.
func HasErrors(iss []Issue) bool {
	for _, i := range iss {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// allRules is the in-order rule registry. Order is stable so diagnostics are
// reproducible across runs.
var allRules = []func(*ir.Application) []Issue{
	ruleAPIVersionDeclared,
	ruleMetadataNamePresent,
	ruleEntityHasPrimaryKey,
	ruleCRUDUpdateFieldsExist,
	ruleRelationshipsResolve,
	ruleRLSReferencesKnownEntity,
	ruleCustomEndpointStepsKnown,
	ruleEventReferencesKnownEntity,
	ruleMultiTenancyHasTenantField,
}

func ruleAPIVersionDeclared(app *ir.Application) []Issue {
	if app.APIVersion == "" {
		return []Issue{{Rule: "VG-VAL-001", Severity: SeverityError, Path: "apiVersion", Message: "apiVersion is required"}}
	}
	if !strings.HasPrefix(app.APIVersion, "vibeguard.dev/") {
		return []Issue{{Rule: "VG-VAL-001", Severity: SeverityWarning, Path: "apiVersion", Message: "apiVersion should be vibeguard.dev/<version>"}}
	}
	return nil
}

func ruleMetadataNamePresent(app *ir.Application) []Issue {
	if app.Metadata.Name == "" {
		return []Issue{{Rule: "VG-VAL-002", Severity: SeverityError, Path: "metadata.name", Message: "metadata.name is required"}}
	}
	return nil
}

func ruleEntityHasPrimaryKey(app *ir.Application) []Issue {
	var out []Issue
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			if ent.PrimaryKey == nil {
				out = append(out, Issue{
					Rule:     "VG-VAL-003",
					Severity: SeverityError,
					Path:     fmt.Sprintf("modules[%s].entities[%s]", mod.Name, ent.Name),
					Message:  "entity has no field marked primary: true",
				})
			}
		}
	}
	return out
}

func ruleCRUDUpdateFieldsExist(app *ir.Application) []Issue {
	var out []Issue
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			present := map[string]bool{}
			for _, f := range ent.Fields {
				present[f.Name] = true
			}
			for _, fname := range ent.CRUD.Update {
				if !present[fname] {
					out = append(out, Issue{
						Rule:     "VG-VAL-004",
						Severity: SeverityError,
						Path:     fmt.Sprintf("modules[%s].entities[%s].crud.update", mod.Name, ent.Name),
						Message:  fmt.Sprintf("update lists field %q which is not declared", fname),
					})
				}
			}
		}
	}
	return out
}

func ruleRelationshipsResolve(app *ir.Application) []Issue {
	var out []Issue
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			for _, rel := range ent.Relationships {
				if rel.Resolved == nil {
					out = append(out, Issue{
						Rule:     "VG-VAL-005",
						Severity: SeverityError,
						Path:     fmt.Sprintf("modules[%s].entities[%s].relationships", mod.Name, ent.Name),
						Message:  fmt.Sprintf("relationship 'to: %s' references an entity that does not exist", rel.To),
					})
				}
			}
		}
	}
	return out
}

func ruleRLSReferencesKnownEntity(app *ir.Application) []Issue {
	known := map[string]bool{}
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			known[ent.Name] = true
		}
	}
	var out []Issue
	for _, mod := range app.Modules {
		for _, p := range mod.Policies.RowLevelSecurity {
			if !known[p.Entity] {
				out = append(out, Issue{
					Rule:     "VG-VAL-006",
					Severity: SeverityError,
					Path:     fmt.Sprintf("modules[%s].policies.row_level_security", mod.Name),
					Message:  fmt.Sprintf("RLS policy references unknown entity %q", p.Entity),
				})
			}
		}
	}
	return out
}

func ruleCustomEndpointStepsKnown(app *ir.Application) []Issue {
	var out []Issue
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			for _, ep := range ent.API.CustomEndpoints {
				if len(ep.Logic.Steps) == 0 {
					out = append(out, Issue{
						Rule:     "VG-VAL-007",
						Severity: SeverityWarning,
						Path:     fmt.Sprintf("modules[%s].entities[%s].api.custom_endpoints[%s]", mod.Name, ent.Name, ep.Path),
						Message:  "custom endpoint declares no logic.steps — handler will be empty",
					})
				}
			}
		}
	}
	return out
}

func ruleEventReferencesKnownEntity(app *ir.Application) []Issue {
	known := map[string]bool{}
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			known[ent.Name] = true
		}
	}
	var out []Issue
	for _, mod := range app.Modules {
		for _, e := range mod.Events {
			if e.Entity != "" && !known[e.Entity] {
				out = append(out, Issue{
					Rule:     "VG-VAL-008",
					Severity: SeverityError,
					Path:     fmt.Sprintf("modules[%s].events", mod.Name),
					Message:  fmt.Sprintf("event %q references unknown entity %q", e.Name, e.Entity),
				})
			}
		}
	}
	return out
}

func ruleMultiTenancyHasTenantField(app *ir.Application) []Issue {
	if !app.Global.MultiTenancy.Enabled {
		return nil
	}
	tenantField := app.Global.MultiTenancy.TenantIDField
	if tenantField == "" {
		tenantField = "tenant_id"
	}
	var out []Issue
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			has := false
			for _, f := range ent.Fields {
				if f.Name == tenantField {
					has = true
					break
				}
			}
			if !has {
				out = append(out, Issue{
					Rule:     "VG-VAL-009",
					Severity: SeverityError,
					Path:     fmt.Sprintf("modules[%s].entities[%s]", mod.Name, ent.Name),
					Message:  fmt.Sprintf("multi_tenancy.enabled but entity has no %q field", tenantField),
				})
			}
		}
	}
	return out
}
