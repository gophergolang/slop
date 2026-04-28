// Package pgsaga is the Postgres-backed durable saga driver.
//
// Status: scaffolded interface and schema in this branch; worker loop is
// the focus of the follow-up branch (see docs/ROADMAP.md).
//
// Design:
//   - saga_instances(id, name, current_step, state jsonb, status, ...)
//   - saga_step_log(saga_id, step_index, phase, status, attempts, ...)
//   - Per-pod Worker claims rows via FOR UPDATE SKIP LOCKED, runs the next
//     step, persists state + current_step in one tx, re-queues with backoff
//     on error.
//   - On compensation: walks current_step → 0 running registered comp fns.
//   - Step *definitions* live in code (registered at boot via
//     workflow.Register), runtime *state* lives in Postgres. Process restart
//     = recovery loop picks up where it left off.
//
// See platform/workflow/pgsaga/SchemaSQL for the DDL the operator installs.
package pgsaga

// SchemaSQL is the durable saga state schema. Applied by the operator (or
// `vibeguard generate` when the local migration target is set).
const SchemaSQL = `CREATE TABLE IF NOT EXISTS saga_instances (
	id              uuid PRIMARY KEY,
	name            text NOT NULL,
	tenant_id       uuid,
	current_step    int  NOT NULL DEFAULT 0,
	state           jsonb NOT NULL DEFAULT '{}'::jsonb,
	status          text NOT NULL,
	last_error      text,
	created_at      timestamptz NOT NULL DEFAULT now(),
	updated_at      timestamptz NOT NULL DEFAULT now(),
	next_attempt_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS saga_pending_idx
	ON saga_instances (next_attempt_at)
	WHERE status IN ('running', 'compensating');

CREATE TABLE IF NOT EXISTS saga_step_log (
	saga_id    uuid REFERENCES saga_instances(id) ON DELETE CASCADE,
	step_index int  NOT NULL,
	step_name  text NOT NULL,
	phase      text NOT NULL,
	status     text NOT NULL,
	attempts   int  NOT NULL DEFAULT 1,
	error      text,
	at         timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (saga_id, step_index, phase, attempts)
);`
