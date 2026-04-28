// Package parser converts on-disk vibeguard.yaml bytes to the typed IR.
//
// The parser is intentionally tolerant: unknown fields are ignored, missing
// optional fields take their zero value. Strict validation is the validator
// package's job. This separation lets us expose `vibeguard ir dump` as a
// debugging aid even on partially-malformed declarations.
package parser

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/vibeguard/vibeguard/internal/ir"
)

// Parse decodes raw YAML bytes into a typed IR Application.
func Parse(data []byte) (*ir.Application, error) {
	var raw rawApplication
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parser: yaml: %w", err)
	}
	app := &ir.Application{
		APIVersion: raw.APIVersion,
		Kind:       raw.Kind,
		Metadata: ir.Metadata{
			Name:        raw.Metadata.Name,
			Version:     raw.Metadata.Version,
			Description: raw.Metadata.Description,
			Compliance:  raw.Metadata.Compliance,
		},
	}
	if raw.Spec.Global != nil {
		app.Global = ir.Global{
			MultiTenancy: ir.MultiTenancy{
				Enabled:       raw.Spec.Global.MultiTenancy.Enabled,
				TenantIDField: raw.Spec.Global.MultiTenancy.TenantIDField,
				Isolation:     raw.Spec.Global.MultiTenancy.Isolation,
			},
		}
	}
	for _, rm := range raw.Spec.Modules {
		mod := decodeModule(rm)
		app.Modules = append(app.Modules, mod)
	}
	resolveAll(app)
	return app, nil
}

// resolveAll fills in pointer fields that depend on cross-module visibility:
// Entity.Module back-pointers, Entity.PrimaryKey, CRUD.UpdateFields, and
// Relationship.Resolved.
func resolveAll(app *ir.Application) {
	byEntityName := map[string]*ir.Entity{}
	for i := range app.Modules {
		mod := &app.Modules[i]
		for _, ent := range mod.Entities {
			ent.Module = mod
			byEntityName[ent.Name] = ent
			for _, f := range ent.Fields {
				if f.Primary {
					ent.PrimaryKey = f
					break
				}
			}
			byField := map[string]*ir.Field{}
			for _, f := range ent.Fields {
				byField[f.Name] = f
			}
			for _, fname := range ent.CRUD.Update {
				if f, ok := byField[fname]; ok {
					ent.CRUD.UpdateFields = append(ent.CRUD.UpdateFields, f)
				}
			}
		}
	}
	for i := range app.Modules {
		for _, ent := range app.Modules[i].Entities {
			for j := range ent.Relationships {
				rel := &ent.Relationships[j]
				rel.Resolved = byEntityName[rel.To]
			}
		}
	}
}

func decodeModule(rm rawModule) ir.Module {
	mod := ir.Module{Name: rm.Name, Type: ir.ModuleType(rm.Type)}
	for _, re := range rm.Entities {
		mod.Entities = append(mod.Entities, decodeEntity(re))
	}
	for _, ri := range rm.Integrations {
		mod.Integrations = append(mod.Integrations, ir.Integration{
			Name:     ri.Name,
			Config:   ri.Config,
			Features: ri.Features,
		})
	}
	for _, rp := range rm.Policies.RowLevelSecurity {
		mod.Policies.RowLevelSecurity = append(mod.Policies.RowLevelSecurity, ir.RLSPolicy{
			Entity:    rp.Entity,
			Condition: rp.Condition,
			ApplyTo:   rp.ApplyTo,
		})
	}
	for _, re := range rm.Events {
		mod.Events = append(mod.Events, ir.Event{
			Name:      re.Name,
			Trigger:   re.Trigger,
			Entity:    re.Entity,
			Condition: re.Condition,
			PublishTo: re.PublishTo,
		})
	}
	return mod
}

func decodeEntity(re rawEntity) *ir.Entity {
	ent := &ir.Entity{
		Name:        re.Name,
		Table:       re.Table,
		Sensitivity: ir.Sensitivity(re.Sensitivity),
		SoftDelete:  re.CRUD.SoftDelete,
		CRUD: ir.CRUD{
			Create:     re.CRUD.Create,
			Read:       re.CRUD.Read,
			List:       re.CRUD.List,
			Update:     re.CRUD.Update.Fields,
			Delete:     re.CRUD.Delete,
			SoftDelete: re.CRUD.SoftDelete,
		},
		API: ir.API{
			BasePath:     re.API.BasePath,
			AuthRequired: re.API.AuthRequired,
			RolesAllowed: re.API.RolesAllowed,
			RateLimit:    re.API.RateLimit,
		},
		BusinessRules: re.BusinessRules,
	}
	for _, rf := range re.Fields {
		f := &ir.Field{
			Name:       rf.Name,
			Type:       ir.FieldType(rf.Type),
			Nullable:   rf.Nullable,
			Unique:     rf.Unique,
			Primary:    rf.Primary,
			EnumValues: rf.Values,
			Validators: rf.Validate,
			DBHints:    ir.DBHints{Index: rf.DB.Index},
		}
		if rf.Default != "" {
			d := rf.Default
			f.Default = &d
		}
		ent.Fields = append(ent.Fields, f)
	}
	for _, rr := range re.Relationships {
		ent.Relationships = append(ent.Relationships, ir.Relationship{
			To:         rr.To,
			Type:       rr.Type,
			ForeignKey: rr.ForeignKey,
		})
	}
	for _, rce := range re.API.CustomEndpoints {
		ce := ir.CustomEndpoint{
			Path:         rce.Path,
			Method:       rce.Method,
			Description:  rce.Description,
			Request:      rce.Request,
			Response:     rce.Response,
			AuthRequired: rce.AuthRequired,
			Logic: ir.Logic{
				Description: rce.Logic.Description,
				Steps:       decodeSteps(rce.Logic.Steps),
			},
		}
		ent.API.CustomEndpoints = append(ent.API.CustomEndpoints, ce)
	}
	return ent
}
