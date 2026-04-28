// Package jetstream is the durable event-bus driver vibeguard uses by default.
//
// JetStream provides:
//   - At-least-once delivery (consumers ACK explicitly)
//   - Durable subscriptions that survive process restarts
//   - Producer-side dedup via Nats-Msg-Id (default 2-minute window)
//   - Configurable retention (WorkQueue / Limits / Interest)
//
// The transactional outbox pattern (see platform/events.Outbox + the drainer
// in this package) layers on top to give effectively-once semantics when
// combined with consumer-side dedup tables.
package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/vibeguard/platform/events"
)

// Client implements events.Client backed by NATS JetStream.
type Client struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	logger *zap.Logger
}

// New connects to NATS and initializes a JetStream context.
func New(url string, logger *zap.Logger) (*Client, error) {
	conn, err := nats.Connect(url,
		nats.Name("vibeguard-platform"),
		nats.MaxReconnects(10),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("connect nats: %w", err)
	}
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("init jetstream: %w", err)
	}
	return &Client{conn: conn, js: js, logger: logger}, nil
}

// Publish sends an event with Nats-Msg-Id set to event.ID for dedup.
func (c *Client) Publish(ctx context.Context, subject string, event events.Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	msg := &nats.Msg{Subject: subject, Data: data, Header: nats.Header{}}
	if event.ID != "" {
		msg.Header.Set(jetstream.MsgIDHeader, event.ID)
	}
	_, err = c.js.PublishMsg(ctx, msg)
	return err
}

// Request is implemented over NATS Core because JetStream is a publish-only
// surface; request/reply is rarely the right tool for durable workflows.
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

// Subscribe creates a push consumer with explicit ACK semantics.
//
// The vibeguard operator (or the bootstrap script) must have created the
// stream that covers `subject` before this is called. This method does not
// declare streams — that's a control-plane concern, not a data-plane one.
func (c *Client) Subscribe(subject string, handler events.Handler) error {
	return c.subscribe(subject, "", handler)
}

// QueueSubscribe creates a pull consumer named `queue` for load balancing.
func (c *Client) QueueSubscribe(subject, queue string, handler events.Handler) error {
	return c.subscribe(subject, queue, handler)
}

func (c *Client) subscribe(subject, queue string, handler events.Handler) error {
	ctx := context.Background()
	stream, err := c.js.StreamNameBySubject(ctx, subject)
	if err != nil {
		return fmt.Errorf("locate stream for %s: %w", subject, err)
	}
	durable := queue
	if durable == "" {
		durable = "vibeguard-" + sanitize(subject)
	}
	cons, err := c.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Durable:       durable,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    10,
		FilterSubject: subject,
	})
	if err != nil {
		return fmt.Errorf("create consumer: %w", err)
	}
	_, err = cons.Consume(func(msg jetstream.Msg) {
		var event events.Event
		if err := json.Unmarshal(msg.Data(), &event); err != nil {
			c.logger.Error("unmarshal event", zap.Error(err), zap.String("subject", subject))
			_ = msg.Term()
			return
		}
		if err := handler(context.Background(), event); err != nil {
			c.logger.Error("handler failed", zap.Error(err), zap.String("subject", subject))
			_ = msg.Nak()
			return
		}
		_ = msg.Ack()
	})
	return err
}

// Close releases the underlying NATS connection.
func (c *Client) Close() { c.conn.Close() }

func sanitize(subject string) string {
	out := make([]byte, 0, len(subject))
	for i := 0; i < len(subject); i++ {
		c := subject[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
			out = append(out, c)
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
