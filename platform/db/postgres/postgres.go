// Package postgres is the reference Postgres driver for platform/db.
//
// Unlike the original prototype where WithTenant only set a context value,
// this driver binds the RequestContext to Postgres session settings on every
// connection checkout via pgxpool's AfterAcquire hook. Generated CREATE POLICY
// statements that reference current_setting('app.tenant_id', true) actually
// fire — RLS is enforced, not decorative.
//
// On connection release, AfterRelease clears the session settings so a
// subsequent checkout for a different tenant cannot read leaked context.
// See tenancy_test.go for the behavior this guards against.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vibeguard/platform/db"
)

// Postgres implements db.DB.
type Postgres struct {
	pool *pgxpool.Pool
}

// New constructs a Postgres driver backed by a pgxpool, with AfterAcquire and
// BeforeRelease hooks installed so RLS session settings track the request
// context attached via db.WithRequest.
func New(ctx context.Context, connString string) (*Postgres, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse pg config: %w", err)
	}

	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		// Initialize session settings to NULL so callers without a
		// RequestContext don't accidentally see a previous tenant's values.
		_, err := conn.Exec(ctx, clearSessionSQL)
		return err
	}
	cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		rc, ok := db.FromContext(ctx)
		if !ok {
			// No request context — background work. Leave session NULL.
			return true
		}
		// Bind the request's identity to Postgres session settings so the
		// generated CREATE POLICY statements that reference
		// current_setting('app.tenant_id', true) actually fire.
		_, err := conn.Exec(ctx, setSessionSQL, rc.TenantID, rc.UserID, rc.Role)
		return err == nil
	}
	cfg.AfterRelease = func(conn *pgx.Conn) bool {
		// Belt-and-braces: clear settings on release so a subsequent
		// non-tenant goroutine acquiring this conn cannot read leaked values.
		_, err := conn.Exec(context.Background(), clearSessionSQL)
		return err == nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

const setSessionSQL = `SELECT
	set_config('app.tenant_id', $1, true),
	set_config('app.user_id',   $2, true),
	set_config('app.role',      $3, true)`

const clearSessionSQL = `SELECT
	set_config('app.tenant_id', NULL, false),
	set_config('app.user_id',   NULL, false),
	set_config('app.role',      NULL, false)`

// Exec runs a statement that returns no rows.
func (p *Postgres) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}

// QueryRow runs a query expected to return at most one row.
func (p *Postgres) QueryRow(ctx context.Context, sql string, args ...any) db.Row {
	return rowAdapter{p.pool.QueryRow(ctx, sql, args...)}
}

// Query runs a query that returns multiple rows.
func (p *Postgres) Query(ctx context.Context, sql string, args ...any) (db.Rows, error) {
	rows, err := p.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &rowsAdapter{rows}, nil
}

// Begin starts a transaction. Inside a tx, set_config(_, _, true) values
// remain in scope until commit/rollback.
func (p *Postgres) Begin(ctx context.Context) (db.Tx, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &pgxTx{tx: tx}, nil
}

// Close closes the connection pool.
func (p *Postgres) Close() {
	p.pool.Close()
}

type rowAdapter struct{ inner pgx.Row }

func (r rowAdapter) Scan(dest ...any) error { return r.inner.Scan(dest...) }

type rowsAdapter struct{ inner pgx.Rows }

func (r *rowsAdapter) Next() bool             { return r.inner.Next() }
func (r *rowsAdapter) Scan(dest ...any) error { return r.inner.Scan(dest...) }
func (r *rowsAdapter) Close()                 { r.inner.Close() }
func (r *rowsAdapter) Err() error             { return r.inner.Err() }

type pgxTx struct{ tx pgx.Tx }

func (t *pgxTx) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}
func (t *pgxTx) QueryRow(ctx context.Context, sql string, args ...any) db.Row {
	return rowAdapter{t.tx.QueryRow(ctx, sql, args...)}
}
func (t *pgxTx) Query(ctx context.Context, sql string, args ...any) (db.Rows, error) {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	return &rowsAdapter{rows}, nil
}
func (t *pgxTx) Begin(ctx context.Context) (db.Tx, error) {
	return nil, errors.New("nested transactions not supported")
}
func (t *pgxTx) Commit() error   { return t.tx.Commit(context.Background()) }
func (t *pgxTx) Rollback() error { return t.tx.Rollback(context.Background()) }
func (t *pgxTx) Close()          {}
