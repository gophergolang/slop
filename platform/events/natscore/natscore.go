// Package natscore is the legacy NATS Core driver. It provides at-most-once
// delivery and is intentionally kept only for compatibility with consumers that
// haven't migrated to the JetStream driver.
//
// Deprecated: New deployments should use platform/events/jetstream which
// provides at-least-once delivery, durable consumers, and dedup windows.
package natscore

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/vibeguard/platform/events"
)

// Client wraps NATS Core with vibeguard conventions.
type Client struct {
	conn   *nats.Conn
	logger *zap.Logger
}

// New creates a new NATS-Core-backed event client.
func New(url string, logger *zap.Logger) (*Client, error) {
	conn, err := nats.Connect(url,
		nats.Name("vibeguard-platform"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	return &Client{conn: conn, logger: logger}, nil
}

// Publish sends an event to a NATS subject (at-most-once).
func (c *Client) Publish(_ context.Context, subject string, event events.Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	return c.conn.Publish(subject, data)
}

// Request sends a request and waits for a reply. The timeout is enforced via
// a context deadline; pass context.WithTimeout from the caller.
func (c *Client) Request(ctx context.Context, subject string, event events.Event, timeout time.Duration) (*events.Event, error) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	data, _ := json.Marshal(event)
	msg, err := c.conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, err
	}
	var reply events.Event
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return nil, fmt.Errorf("unmarshal reply: %w", err)
	}
	return &reply, nil
}

// Subscribe subscribes to a subject. At-most-once delivery; no replay.
func (c *Client) Subscribe(subject string, handler events.Handler) error {
	_, err := c.conn.Subscribe(subject, c.adapt(subject, handler))
	return err
}

// QueueSubscribe subscribes with a queue group (load balancing).
func (c *Client) QueueSubscribe(subject, queue string, handler events.Handler) error {
	_, err := c.conn.QueueSubscribe(subject, queue, c.adapt(subject, handler))
	return err
}

// Close closes the NATS connection.
func (c *Client) Close() { c.conn.Close() }

func (c *Client) adapt(subject string, handler events.Handler) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var event events.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.logger.Error("unmarshal event", zap.Error(err), zap.String("subject", subject))
			return
		}
		if err := handler(context.Background(), event); err != nil {
			c.logger.Error("handler failed", zap.Error(err), zap.String("subject", subject))
		}
	}
}
