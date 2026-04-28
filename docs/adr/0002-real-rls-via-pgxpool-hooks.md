# ADR 0002 — Real RLS enforcement via pgxpool checkout hooks

## Status

Accepted (branch 4-7).

## Context

The original `platform/db/postgres.go` had a method `WithTenant(ctx, tenantID)` that did this:

```go
func (p *Postgres) WithTenant(ctx context.Context, tenantID string) context.Context {
    return context.WithValue(ctx, tenantKey{}, tenantID)
}
```

It stored the tenant id in a Go context value and never told Postgres anything. Meanwhile, the declaration's `policies.row_level_security` blocks emitted `CREATE POLICY` statements that referenced `current_setting('app.tenant_id', true)` — a Postgres session setting that nothing was setting. So:

- RLS policies were valid SQL
- They were applied to tables
- They never matched (because `current_setting` returned empty)
- Every query saw every tenant's rows

This was the second-biggest credibility hole after the half-built generator. A multi-tenant SaaS that doesn't actually enforce tenant isolation is not multi-tenant.

## Decision

Use `pgxpool.Config.BeforeAcquire` and `AfterRelease` to bind the request's `RequestContext` to Postgres session settings on every connection checkout, and clear them on release.

```go
cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
    rc, ok := db.FromContext(ctx)
    if !ok { return true }
    _, err := conn.Exec(ctx,
        "SELECT set_config('app.tenant_id',$1,true), "+
        "       set_config('app.user_id',$2,true), "+
        "       set_config('app.role',$3,true)",
        rc.TenantID, rc.UserID, rc.Role)
    return err == nil
}
cfg.AfterRelease = func(conn *pgx.Conn) bool {
    _, _ = conn.Exec(context.Background(),
        "SELECT set_config('app.tenant_id',NULL,false), ...")
    return true
}
```

`set_config(_, _, true)` is the SET-LOCAL-equivalent that respects transactions. Returning `false` from `BeforeAcquire` destroys the connection — appropriate behavior if we can't bind tenant context.

The generator emits the `RequestContext` middleware that runs after auth and installs the `RequestContext` on `c.Request.Context()`. Generated repositories then call `db.DB.Exec/Query/QueryRow` like normal — the binding is invisible and automatic.

## Consequences

**Good:**

- `CREATE POLICY ... USING (tenant_id::text = current_setting('app.tenant_id', true))` actually fires.
- Generated code doesn't need to think about RLS — it's enforced by the driver.
- Tests can construct two `RequestContext`s and prove that connection A cannot see connection B's rows (the killer regression test in `tenancy_test.go`).
- The same mechanism gives `app.user_id` and `app.role` to RLS conditions for finer-grained policies.

**Tradeoffs:**

- Every connection checkout pays a single round-trip for `set_config`. Negligible vs. query latency, but real.
- `AfterRelease` clears settings unconditionally. A future optimization could skip the clear if the next checkout is for the same tenant — not worth the complexity right now.
- Background goroutines without a `RequestContext` get `NULL` session settings, which means RLS will reject all rows. Background work that legitimately needs to read across tenants must use a `db.Background()` checkout that bypasses these hooks. (Not yet implemented — flagged for branch `4-7-followup`.)

## Alternatives considered

- **Pass tenant explicitly to every query.** The existing `WHERE tenant_id = $1` pattern. Works but: every repository method has to remember to do it; an injection vulnerability bypasses it; RLS is the belt + suspenders, not a substitute.
- **Use `SET LOCAL` inside an explicit `BEGIN`.** Forces every read into a transaction. `set_config(_,_,true)` is the cleaner idiom that pgx recommends.
- **Per-tenant connection pools.** Would scale poorly past ~100 tenants and complicates pool warmup. The session-setting approach scales linearly with request rate, not tenant count.

## Verification

- Tenancy leak test (lands in `4-7-followup`): two contexts, two tenants, two checkouts; assert read-A returns empty for tenant B's tenant_id.
- The `vibeguard generate` output's `cmd/server/main.go` installs the middleware that calls `db.WithRequest`. A handler test using `httptest` confirms the binding propagates.

## See also

- [`docs/ARCHITECTURE.md`](../ARCHITECTURE.md#real-rls-enforcement-the-key-fix)
- `platform/db/postgres/postgres.go` — the implementation
