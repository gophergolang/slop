package workflow

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Step represents one step in a saga.
type Step struct {
	Name      string
	Execute   func(ctx context.Context) error
	Compensate func(ctx context.Context) error // optional
}

// Saga runs a series of steps with compensation on failure.
type Saga struct {
	steps  []Step
	logger *zap.Logger
}

// NewSaga creates a new saga.
func NewSaga(logger *zap.Logger) *Saga {
	return &Saga{logger: logger}
}

// AddStep adds a step to the saga.
func (s *Saga) AddStep(name string, execute func(ctx context.Context) error, compensate func(ctx context.Context) error) {
	s.steps = append(s.steps, Step{Name: name, Execute: execute, Compensate: compensate})
}

// Run executes all steps. On failure it runs compensations in reverse.
func (s *Saga) Run(ctx context.Context) error {
	var executed []int

	for i, step := range s.steps {
		s.logger.Info("executing saga step", zap.String("step", step.Name))
		if err := step.Execute(ctx); err != nil {
			s.logger.Error("saga step failed, compensating", zap.String("step", step.Name), zap.Error(err))
			s.compensate(ctx, executed)
			return fmt.Errorf("saga failed at step %s: %w", step.Name, err)
		}
		executed = append(executed, i)
	}
	return nil
}

func (s *Saga) compensate(ctx context.Context, executed []int) {
	for i := len(executed) - 1; i >= 0; i-- {
		idx := executed[i]
		step := s.steps[idx]
		if step.Compensate != nil {
			s.logger.Info("running compensation", zap.String("step", step.Name))
			if err := step.Compensate(ctx); err != nil {
				s.logger.Error("compensation failed", zap.String("step", step.Name), zap.Error(err))
			}
		}
	}
}