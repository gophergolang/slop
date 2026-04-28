// Package featureflag is a minimal feature-flag interface compatible with
// OpenFeature semantics (returning a default on resolution failure).
//
// Reference adapter (jsonfile) lives at featureflag/jsonfile. The
// declaration's `feature_flags` block compiles to constants of these flag
// names so handlers reference flags by typed identifier, not by string.
package featureflag

import "context"

// Provider resolves a flag value with a fallback default.
type Provider interface {
	Bool(ctx context.Context, key string, dflt bool) bool
	String(ctx context.Context, key string, dflt string) string
}
