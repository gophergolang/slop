package db

import "context"

// RequestContext carries the per-request identity vibeguard uses for row-level
// security. Drivers bind these values to Postgres session settings on connection
// checkout (see platform/db/postgres.AfterAcquire).
type RequestContext struct {
	TenantID string
	UserID   string
	Role     string
}

type ctxKey struct{}

// WithRequest attaches a RequestContext to ctx. Generated handler middleware
// installs it after authenticating the caller; drivers read it on every
// connection checkout.
func WithRequest(ctx context.Context, rc RequestContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, rc)
}

// FromContext returns the RequestContext attached to ctx, if any.
//
// A missing RequestContext is legitimate for background work (drainers,
// reconcilers) but generated request handlers must always have one. Use
// RequireRequest in middleware to enforce this.
func FromContext(ctx context.Context) (RequestContext, bool) {
	rc, ok := ctx.Value(ctxKey{}).(RequestContext)
	return rc, ok
}
