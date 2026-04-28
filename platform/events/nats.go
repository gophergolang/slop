package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// Publisher defines the interface for publishing events.
type Publisher interface {
	Publish(ctx context.Context, subject string, event Event) error
	Request(ctx context.Context, subject string, event Event, timeout time.Duration) (*Event, error)
}

// Subscriber defines the interface for consuming events.
type Subscriber interface {
	Subscribe(subject string, handler func(ctx context.Context, event Event) error) error
	QueueSubscribe(subject, queue string, handler func(ctx context.Context, event Event) error) error
}

// Event is the standard envelope used across VibeGuard.
type Event struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	TenantID  string          `json:"tenant_id,omitempty"`
	Source    string          `json:"source,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Client wraps NATS with VibeGuard conventions.
type Client struct {
	conn   *nats.Conn
	logger *zap.Logger
}

// NewClient creates a new NATS-backed event client.
func NewClient(url string, logger *zap.Logger) (*Client, error) {
	conn, err := nats.Connect(url,
		nats.Name("vibeguard-platform"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	return &Client{conn: conn, logger: logger}, nil
}

// Publish sends an event to a NATS subject.
func (c *Client) Publish(ctx context.Context, subject string, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return c.conn.Publish(subject, data)
}

// Request sends a request and waits for a reply (useful for RPC-style events).
func (c *Client) Request(ctx context.Context, subject string, event Event, timeout time.Duration) (*Event, error) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, _ := json.Marshal(event)

	msg, err := c.conn.RequestWithContext(ctx, subject, data, timeout)
	if err != nil {
		return nil, err
	}

	var reply Event
	if err := json.Unmarshal(msg.Data, &reply); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reply: %w", err)
	}
	return &reply, nil
}

// Subscribe subscribes to a subject.
func (c *Client) Subscribe(subject string, handler func(ctx context.Context, event Event) error) error {
	_, err := c.conn.Subscribe(subject, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.logger.Error("failed to unmarshal event", zap.Error(err))
			return
		}
		if err := handler(context.Background(), event); err != nil {
			c.logger.Error("handler failed", zap.Error(err), zap.String("subject", subject))
		}
	})
	return err
}

// QueueSubscribe subscribes to a subject with a queue group (load balancing).
func (c *Client) QueueSubscribe(subject, queue string, handler func(ctx context.Context, event Event) error) error {
	_, err := c.conn.QueueSubscribe(subject, queue, func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			c.logger.Error("failed to unmarshal event", zap.Error(err))
			return
		}
		if err := handler(context.Background(), event); err != nil {
			c.logger.Error("handler failed", zap.Error(err), zap.String("subject", subject))
		}
	})
	return err
}

// Close closes the NATS connection.
func (c *Client) Close() {
	c.conn.Close()
}