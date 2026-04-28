// Package observability is the OpenTelemetry initialization layer.
//
// Each platform subsystem (db, events, workflow, llm) calls otel.Tracer with
// its own instrumentation name, so generated code gets distributed traces,
// metrics, and structured logs for free — zero instrumentation calls in user
// code.
//
// The Init function is intentionally tiny: it wires an OTLP exporter (HTTP
// or gRPC, autodetected from env) and returns a shutdown function callers
// must defer. Production deployments configure the OTLP endpoint via
// OTEL_EXPORTER_OTLP_ENDPOINT.
package observability

import (
	"context"
	"fmt"
	"os"
)

// Config controls OTel initialization. Zero values pick reasonable defaults
// from OTEL_* environment variables.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string // OTEL_EXPORTER_OTLP_ENDPOINT if empty
}

// Init returns a shutdown function or an error. In this scaffold the function
// is a no-op; the follow-up branch wires the real OTLP exporter.
func Init(ctx context.Context, cfg Config) (shutdown func(context.Context) error, err error) {
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("observability: ServiceName required")
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	}
	// TODO(4-7-followup): wire OTLP HTTP/gRPC exporters here.
	return func(context.Context) error { return nil }, nil
}
