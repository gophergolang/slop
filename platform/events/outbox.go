package events

import (
	"context"

	"github.com/vibeguard/platform/db"
)

// Outbox is the transactional outbox primitive. EnqueueTx writes the event to
// the outbox table inside the caller's transaction. A separate drainer process
// reads the outbox and publishes to the underlying bus, providing at-least-once
// semantics with strong durability.
//
// Generated handlers always call EnqueueTx (never Publish directly) so that
// business writes and event publication share one atomic boundary.
type Outbox interface {
	EnqueueTx(ctx context.Context, tx db.Tx, subject string, event Event) error
}

// OutboxSchemaSQL is the DDL the operator (or `vibeguard generate`) installs
// before the application boots.
const OutboxSchemaSQL = `CREATE TABLE IF NOT EXISTS outbox (
	id              uuid PRIMARY KEY,
	tenant_id       uuid,
	type            text NOT NULL,
	subject         text NOT NULL,
	payload         jsonb NOT NULL,
	headers         jsonb,
	created_at      timestamptz NOT NULL DEFAULT now(),
	published_at    timestamptz,
	attempts        int NOT NULL DEFAULT 0,
	next_attempt_at timestamptz NOT NULL DEFAULT now(),
	status          text NOT NULL DEFAULT 'pending'
);
CREATE INDEX IF NOT EXISTS outbox_pending_idx
	ON outbox (next_attempt_at)
	WHERE status = 'pending';`
