// Package workflow defines the saga interface generated code uses for
// multi-step transactional workflows with compensation.
//
// The reference durable driver lives in workflow/pgsaga (Postgres-backed
// state machine, recovery on restart). The simpler in-process driver in
// workflow/inmem is fine for tests and dev — but production deployments
// should use pgsaga so a process crash mid-workflow doesn't strand state.
package workflow

import "context"

// StepFn executes business logic for one step in a saga.
type StepFn func(ctx context.Context, state State) error

// State is the saga's working memory, persisted between steps by durable
// drivers and held in process memory by inmem.
type State interface {
	Get(key string) (any, bool)
	Set(key string, value any)
}

// Saga is a builder + runner for sequenced compensable steps.
//
// AddStep registers an Execute function and an optional Compensate function.
// Run executes steps in order; on failure, registered compensations run in
// reverse for the steps that already succeeded.
type Saga interface {
	AddStep(name string, execute StepFn, compensate StepFn)
	Run(ctx context.Context, input map[string]any) (InstanceID, error)
}

// InstanceID identifies a saga run for status queries and admin operations.
type InstanceID string
