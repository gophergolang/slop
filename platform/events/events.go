// Package events defines the abstract event-bus interfaces all vibeguard modules use.
//
// Concrete drivers live in subpackages (jetstream/, natscore/, kafka/, ...).
// Generated code imports only this package; the driver is wired at process
// bootstrap from the declaration's spec.platform.events.driver field.
//
// The Event envelope is the universal shape. Drivers must preserve all fields
// exactly when round-tripping through their wire format. Event.ID is also the
// idempotency key used by JetStream's deduplication window and by consumer-side
// processed_events tables.
package events

import (
	"context"
	"encoding/json"
	"time"
)

// Event is the standard envelope. TenantID is required for every event in a
// multi-tenant declaration; consumers MUST filter on it before acting.
type Event struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	TenantID  string            `json:"tenant_id,omitempty"`
	Source    string            `json:"source,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
	Data      json.RawMessage   `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Publisher publishes events to the bus.
type Publisher interface {
	Publish(ctx context.Context, subject string, event Event) error
	Request(ctx context.Context, subject string, event Event, timeout time.Duration) (*Event, error)
}

// Handler is invoked once per delivered event. Returning a non-nil error causes
// the driver to NACK / redeliver per its at-least-once contract.
type Handler func(ctx context.Context, event Event) error

// Subscriber consumes events from the bus. QueueSubscribe load-balances across
// instances sharing the same queue name (competing consumers).
type Subscriber interface {
	Subscribe(subject string, handler Handler) error
	QueueSubscribe(subject, queue string, handler Handler) error
}

// Client combines publisher + subscriber, which is what generated code needs.
type Client interface {
	Publisher
	Subscriber
	Close()
}
