package ir

import "strings"

// EntityRootPath returns the chain of entities from the topmost ancestor down
// to ent (inclusive). When ent has no parents, the slice contains only ent.
// When the parent graph is a DAG with multiple parents, the first declared
// parent at each level is followed (single chain). Cycle-safe: validators
// reject cycles before any backend calls this.
func EntityRootPath(ent *Entity) []*Entity {
	if ent == nil {
		return nil
	}
	chain := []*Entity{ent}
	seen := map[string]bool{ent.Name: true}
	cur := ent
	for len(cur.ParentRefs) > 0 {
		p := cur.ParentRefs[0]
		if p == nil || seen[p.Name] {
			break
		}
		seen[p.Name] = true
		chain = append([]*Entity{p}, chain...)
		cur = p
	}
	return chain
}

// EffectiveBasePath returns the URL prefix for ent's REST collection,
// honoring an explicit ent.API.BasePath when set; otherwise deriving a
// nested path from the parent chain (e.g. /api/v1/teams/:team_id/tasks for
// Team -> Task). The returned path never has a trailing slash.
func EffectiveBasePath(ent *Entity) string {
	if ent == nil {
		return ""
	}
	if ent.API.BasePath != "" {
		return strings.TrimRight(ent.API.BasePath, "/")
	}
	chain := EntityRootPath(ent)
	parts := []string{"api", "v1"}
	for i, e := range chain {
		parts = append(parts, pluralLower(e.Name))
		if i < len(chain)-1 {
			parts = append(parts, ":"+snakeLower(e.Name)+"_id")
		}
	}
	return "/" + strings.Join(parts, "/")
}

// EffectiveItemPath returns the URL for a single item under EffectiveBasePath
// (i.e. /api/v1/teams/:team_id/tasks/:id).
func EffectiveItemPath(ent *Entity) string {
	return EffectiveBasePath(ent) + "/:id"
}

func pluralLower(name string) string {
	low := snakeLower(name)
	switch {
	case strings.HasSuffix(low, "y"):
		return low[:len(low)-1] + "ies"
	case strings.HasSuffix(low, "s"), strings.HasSuffix(low, "x"), strings.HasSuffix(low, "z"):
		return low + "es"
	default:
		return low + "s"
	}
}

func snakeLower(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
