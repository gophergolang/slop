package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/vibeguard/platform/llm/prompts"
)

// PromptStore is what the gateway needs from the prompt loader.
type PromptStore interface {
	Load(name, version string) (*prompts.Prompt, error)
}

// gateway is the default Gateway implementation.
type gateway struct {
	driver  Driver
	store   PromptStore
	cache   ResponseCache
	metrics Metrics
}

// NewGateway constructs the default Gateway.
func NewGateway(driver Driver, store PromptStore, opts ...Option) Gateway {
	g := &gateway{driver: driver, store: store, metrics: noopMetrics{}}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Option configures a Gateway.
type Option func(*gateway)

// WithCache enables response caching.
func WithCache(c ResponseCache) Option { return func(g *gateway) { g.cache = c } }

// WithMetrics installs a metrics recorder.
func WithMetrics(m Metrics) Option { return func(g *gateway) { g.metrics = m } }

// Metrics records per-call counters. The default is no-op; production
// installations wire a Prometheus implementation.
type Metrics interface {
	RecordCall(ctx context.Context, attrs CallAttrs)
}

// CallAttrs is the attribute bag for a recorded call.
type CallAttrs struct {
	Driver        string
	Model         string
	PromptName    string
	PromptVersion string
	TenantID      string
	EndpointPath  string
	PromptTokens  int
	OutputTokens  int
	LatencyMS     int64
	Status        string // "ok" | "repaired" | "failed"
}

type noopMetrics struct{}

func (noopMetrics) RecordCall(context.Context, CallAttrs) {}

// ResponseCache is the (optional) cache contract.
type ResponseCache interface {
	Get(ctx context.Context, key string) (*Response, bool)
	Set(ctx context.Context, key string, resp *Response, ttl time.Duration)
}

// Call is the gateway's single entry point.
func (g *gateway) Call(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()
	prompt, err := g.store.Load(req.PromptName, req.PromptVersion)
	if err != nil {
		return nil, fmt.Errorf("load prompt: %w", err)
	}
	model := req.Model
	if model == "" {
		model = prompt.Frontmatter.ModelDefault
	}

	cacheKey := ""
	if req.CacheTTL > 0 && g.cache != nil {
		cacheKey = cacheKeyFor(model, req.PromptName, req.PromptVersion, req.Variables)
		if cached, ok := g.cache.Get(ctx, cacheKey); ok {
			cached.Cached = true
			return cached, nil
		}
	}

	rendered, err := renderTemplate(prompt.Body, req.Variables)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}
	driverReq := DriverRequest{
		Model:    model,
		Messages: []Message{{Role: "user", Content: rendered}},
	}

	resp, parsed, repaired, err := g.callWithRepair(ctx, driverReq, prompt, req.MaxRepairs)
	status := "ok"
	if err != nil {
		status = "failed"
	} else if repaired {
		status = "repaired"
	}
	g.metrics.RecordCall(ctx, CallAttrs{
		Driver:        g.driver.Name(),
		Model:         model,
		PromptName:    req.PromptName,
		PromptVersion: req.PromptVersion,
		TenantID:      req.TenantID,
		EndpointPath:  req.EndpointPath,
		PromptTokens:  resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
		LatencyMS:    time.Since(start).Milliseconds(),
		Status:       status,
	})
	if err != nil {
		return nil, err
	}
	out := &Response{
		Raw:          resp.Text,
		Parsed:       parsed,
		PromptTokens: resp.PromptTokens,
		OutputTokens: resp.OutputTokens,
		Model:        model,
		Repaired:     repaired,
		LatencyMS:    time.Since(start).Milliseconds(),
	}
	if cacheKey != "" {
		g.cache.Set(ctx, cacheKey, out, req.CacheTTL)
	}
	return out, nil
}

func (g *gateway) callWithRepair(ctx context.Context, dreq DriverRequest, prompt *prompts.Prompt, maxRepairs int) (DriverResponse, any, bool, error) {
	if maxRepairs == 0 {
		maxRepairs = 2
	}
	if maxRepairs < 0 {
		maxRepairs = 0
	}

	resp, err := g.driver.Complete(ctx, dreq)
	if err != nil {
		return resp, nil, false, err
	}
	parsed, parseErr := validateOutput(resp.Text, prompt.Frontmatter.OutputSchema)
	if parseErr == nil {
		return resp, parsed, false, nil
	}

	for attempt := 1; attempt <= maxRepairs; attempt++ {
		dreq.Messages = append(dreq.Messages,
			Message{Role: "assistant", Content: resp.Text},
			Message{Role: "user", Content: fmt.Sprintf("That output failed validation: %s. Reply with valid JSON ONLY, no prose.", parseErr.Error())},
		)
		resp, err = g.driver.Complete(ctx, dreq)
		if err != nil {
			return resp, nil, true, err
		}
		parsed, parseErr = validateOutput(resp.Text, prompt.Frontmatter.OutputSchema)
		if parseErr == nil {
			return resp, parsed, true, nil
		}
	}
	return resp, nil, true, fmt.Errorf("output failed validation after %d repairs: %w", maxRepairs, parseErr)
}

func validateOutput(raw string, schemaMap map[string]any) (any, error) {
	if len(schemaMap) == 0 {
		// No schema declared — accept the raw text.
		return raw, nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("not valid JSON: %w", err)
	}
	schemaJSON, _ := json.Marshal(schemaMap)
	c := jsonschema.NewCompiler()
	if err := c.AddResource("inline://output", bytes.NewReader(schemaJSON)); err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	sch, err := c.Compile("inline://output")
	if err != nil {
		return nil, fmt.Errorf("compile schema: %w", err)
	}
	if err := sch.Validate(parsed); err != nil {
		return nil, fmt.Errorf("schema: %w", err)
	}
	return parsed, nil
}

func renderTemplate(body string, vars map[string]any) (string, error) {
	tmpl, err := template.New("prompt").Option("missingkey=error").Parse(body)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func cacheKeyFor(model, promptName, promptVersion string, vars map[string]any) string {
	canonical, _ := json.Marshal(vars)
	return promptName + ":" + promptVersion + ":" + model + ":" + prompts.SHA256Hex(canonical)
}
