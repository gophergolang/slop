# VibeGuard Platform SDK

**The foundation for secure, event-driven, composable applications.**

## Philosophy

VibeGuard generates **thin** application code. All heavy lifting (database access, events, workflows, security, observability) happens through this well-designed, hand-written Platform SDK.

This gives us:
- Excellent long-term maintainability (humans own the important code)
- True infrastructure flexibility (swap Postgres ↔ CockroachDB, NATS ↔ Kafka by changing config)
- Strong security & compliance enforcement at runtime
- Clean path to the full event-driven + operator-per-app vision

## Packages

### `events` (NATS-powered)
- Clean `Publisher` / `Subscriber` interfaces
- Standard `Event` envelope with `tenant_id`, `type`, `data`
- Ready for exactly-once, dead-letter queues, and schema evolution

### `db`
- Abstract `DB` interface (tenant-aware by default)
- Postgres implementation with automatic RLS context
- Transaction support

### `workflow`
- Lightweight saga runner with automatic compensation
- Designed to grow into full workflow orchestration

## Usage Example (generated code will look like this)

```go
import (
    "github.com/vibeguard/platform/events"
    "github.com/vibeguard/platform/db"
    "github.com/vibeguard/platform/workflow"
)

func (h *TaskHandler) Prioritize(c *gin.Context) {
    saga := workflow.NewSaga(h.logger)
    
    saga.AddStep("load_task", func(ctx) error {
        return h.db.QueryRow(ctx, "SELECT ...").Scan(&task)
    }, nil)
    
    saga.AddStep("call_ai", func(ctx) error {
        result, _ := h.aiClient.Prioritize(task)
        return nil
    }, nil)
    
    saga.AddStep("update_priority", func(ctx) error {
        return h.db.Exec(ctx, "UPDATE tasks SET priority = $1", result.Priority)
    }, func(ctx) error {
        // compensation
        return h.db.Exec(ctx, "UPDATE tasks SET priority = $1", oldPriority)
    })
    
    if err := saga.Run(c.Request.Context()); err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // emit event
    h.events.Publish(c.Request.Context(), "tasks.prioritized", events.Event{
        Type: "TaskPrioritized",
        Data: json.RawMessage(`{...}`),
    })
}
```

## Roadmap

- Phase 1: Core interfaces + NATS + Postgres (current)
- Phase 2: Outbox pattern, exactly-once, dead-letter queues
- Phase 3: Full saga/workflow engine + compensation DSL
- Phase 4: Kubernetes Operator generation + GitOps integration

This SDK is the real moat of VibeGuard. Generated code stays thin and human-owned.