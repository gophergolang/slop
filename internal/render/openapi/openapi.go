// Package openapi is the OpenAPI 3.1 backend.
//
// Status: emits a small but valid spec listing the routes the declaration
// enabled. Schema components for entities are scaffolded — full
// CreateRequest / UpdateRequest variants land in the follow-up branch.
package openapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vibeguard/vibeguard/internal/ir"
	"github.com/vibeguard/vibeguard/internal/render"
)

// Backend implements render.Backend for an OpenAPI 3.1 spec.
type Backend struct{}

// New constructs an OpenAPI backend.
func New() *Backend { return &Backend{} }

// Name reports "openapi".
func (Backend) Name() string { return "openapi" }

// Plan emits openapi.json.
func (Backend) Plan(app *ir.Application) (render.FileSet, error) {
	spec := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       app.Metadata.Name,
			"version":     app.Metadata.Version,
			"description": app.Metadata.Description,
		},
		"paths":      pathsFor(app),
		"components": map[string]any{"schemas": schemasFor(app)},
	}
	out, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal openapi: %w", err)
	}
	return render.FileSet{
		{Path: "openapi.json", Mode: 0o644, Content: out},
	}, nil
}

func pathsFor(app *ir.Application) map[string]any {
	out := map[string]any{}
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			base := ent.API.BasePath
			if base == "" {
				continue
			}
			collection := map[string]any{}
			if ent.CRUD.List {
				collection["get"] = op(ent, "list", "200", "List "+ent.Name)
			}
			if ent.CRUD.Create {
				collection["post"] = op(ent, "create", "201", "Create a "+ent.Name)
			}
			if len(collection) > 0 {
				out[base] = collection
			}
			item := map[string]any{}
			if ent.CRUD.Read {
				item["get"] = op(ent, "get", "200", "Get a "+ent.Name)
			}
			if len(ent.CRUD.UpdateFields) > 0 {
				item["patch"] = op(ent, "update", "200", "Update a "+ent.Name)
			}
			if ent.CRUD.Delete {
				item["delete"] = op(ent, "delete", "204", "Delete a "+ent.Name)
			}
			if len(item) > 0 {
				out[strings.TrimRight(base, "/")+"/{id}"] = item
			}
		}
	}
	return out
}

func op(ent *ir.Entity, kind, status, summary string) map[string]any {
	return map[string]any{
		"summary":   summary,
		"operationId": ent.Name + "_" + kind,
		"tags":      []string{ent.Module.Name},
		"responses": map[string]any{status: map[string]any{"description": "ok"}},
	}
}

func schemasFor(app *ir.Application) map[string]any {
	out := map[string]any{}
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			out[ent.Name] = map[string]any{
				"type":       "object",
				"properties": props(ent),
			}
		}
	}
	return out
}

func props(ent *ir.Entity) map[string]any {
	out := map[string]any{}
	for _, f := range ent.Fields {
		out[f.Name] = map[string]any{"type": jsonType(f)}
	}
	return out
}

func jsonType(f *ir.Field) string {
	switch f.Type {
	case ir.FieldString, ir.FieldText, ir.FieldUUID, ir.FieldEnum, ir.FieldTimestamp, ir.FieldDecimal:
		return "string"
	case ir.FieldInt, ir.FieldBigInt:
		return "integer"
	case ir.FieldBool:
		return "boolean"
	case ir.FieldJSON:
		return "object"
	}
	return "string"
}
