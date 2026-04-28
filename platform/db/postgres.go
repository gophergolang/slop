package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the abstract interface all VibeGuard modules use.
type DB interface {
	Exec(ctx context.Context, sql string, args ...any) error
	QueryRow(ctx context.Context, sql string, args ...any) Row
	Query(ctx context.Context, sql string, args ...any) (Rows, error)
	Begin(ctx context.Context) (Tx, error)
	Close()
}

// Row is a single row result.
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

// Tx is a transaction.
type Tx interface {
	DB
	Commit() error
	Rollback() error
}

// Postgres implements DB using pgx.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres creates a tenant-aware Postgres client.
func NewPostgres(ctx context.Context, connString string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}
	return &Postgres{pool: pool}, nil
}

// WithTenant sets the tenant context for RLS (recommended to call this in middleware).
func (p *Postgres) WithTenant(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantKey{}, tenantID)
}

type tenantKey struct{}

// Exec executes a query.
func (p *Postgres) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}

// QueryRow executes a query that returns a single row.
func (p *Postgres) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

// Query executes a query that returns multiple rows.
func (p *Postgres) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

// Begin starts a transaction.
func (p *Postgres) Begin(ctx context.Context) (Tx, error) {
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

// pgxTx wraps a pgx.Tx to satisfy the Tx interface.
type pgxTx struct {
	tx pgx.Tx
}

func (t *pgxTx) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := t.tx.Exec(ctx, sql, args...)
	return err
}

func (t *pgxTx) QueryRow(ctx context.Context, sql string, args ...any) Row {
	return t.tx.QueryRow(ctx, sql, args...)
}

func (t *pgxTx) Query(ctx context.Context, sql string, args ...any) (Rows, error) {
	return t.tx.Query(ctx, sql, args...)
}

func (t *pgxTx) Begin(ctx context.Context) (Tx, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (t *pgxTx) Commit() error   { return t.tx.Commit(context.Background()) }
func (t *pgxTx) Rollback() error { return t.tx.Rollback(context.Background()) }
func (t *pgxTx) Close()          {}