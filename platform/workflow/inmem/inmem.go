// Package inmem is the in-process saga driver. It runs synchronously in the
// caller's goroutine and holds state in memory.
//
// Use only for tests, dev, and within-request workflows that do not need
// crash recovery. Production multi-step transactions should use pgsaga.
package inmem

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/vibeguard/platform/workflow"
)

type step struct {
	name       string
	execute    workflow.StepFn
	compensate workflow.StepFn
}

// Saga is an in-process implementation of workflow.Saga.
type Saga struct {
	steps  []step
	logger *zap.Logger
}

// New creates a new in-process saga.
func New(logger *zap.Logger) *Saga {
	return &Saga{logger: logger}
}

// AddStep registers a step.
func (s *Saga) AddStep(name string, execute workflow.StepFn, compensate workflow.StepFn) {
	s.steps = append(s.steps, step{name: name, execute: execute, compensate: compensate})
}

// Run executes all steps synchronously. On failure, registered compensations
// run in reverse for the steps that already succeeded. Returns a synthetic
// instance id so callers can correlate logs.
func (s *Saga) Run(ctx context.Context, input map[string]any) (workflow.InstanceID, error) {
	id := workflow.InstanceID(randomID())
	st := &mapState{m: map[string]any{}}
	for k, v := range input {
		st.Set(k, v)
	}
	var executed []int
	for i, st_ := range s.steps {
		s.logger.Info("execute step",
			zap.String("saga", string(id)),
			zap.String("step", st_.name))
		if err := st_.execute(ctx, st); err != nil {
			s.logger.Error("step failed; compensating",
				zap.String("saga", string(id)),
				zap.String("step", st_.name),
				zap.Error(err))
			s.compensate(ctx, st, executed)
			return id, fmt.Errorf("saga %s failed at %s: %w", id, st_.name, err)
		}
		executed = append(executed, i)
	}
	return id, nil
}

func (s *Saga) compensate(ctx context.Context, state workflow.State, executed []int) {
	for i := len(executed) - 1; i >= 0; i-- {
		st := s.steps[executed[i]]
		if st.compensate == nil {
			continue
		}
		s.logger.Info("compensate", zap.String("step", st.name))
		if err := st.compensate(ctx, state); err != nil {
			s.logger.Error("compensation failed",
				zap.String("step", st.name),
				zap.Error(err))
		}
	}
}

type mapState struct {
	mu sync.Mutex
	m  map[string]any
}

func (s *mapState) Get(key string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.m[key]
	return v, ok
}

func (s *mapState) Set(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = value
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
