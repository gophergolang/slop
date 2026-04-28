// Package db defines the abstract database interfaces all vibeguard modules use.
//
// Concrete drivers live in subpackages (postgres/, cockroach/, ...). Generated code
// imports only this package; the driver is wired at process bootstrap from the
// declaration's spec.platform.db.driver field.
package db

import "context"

// DB is the abstract interface generated code calls. Drivers implement it.
//
// Every Exec/Query/QueryRow respects the RequestContext attached via WithRequest,
// which the driver propagates into Postgres session settings (app.tenant_id,
// app.user_id, app.role) so RLS policies fire automatically. See platform/db/postgres
// for the reference implementation.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) error
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	Begin(ctx context.Context) (Tx, error)
	Close()
}

// Row is a single-row result.
type Row interface {
	Scan(dest ...any) error
}

// Rows is a multi-row result.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

// Tx is a database transaction. It is itself a DB so callers can compose.
type Tx interface {
	DB
	Commit() error
	Rollback() error
}
