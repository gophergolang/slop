// Package llm is the runtime LLM gateway for vibeguard apps.
//
// Generated handlers that include an external_call step with service: openai
// or service: anthropic call into this gateway. The gateway provides:
//
//   - Multi-driver: Anthropic, OpenAI, and any OpenAI-compatible endpoint
//   - Sealed, versioned prompts: prompt bodies are sha256-checked against
//     the version manifest at boot. A drifted prompt refuses to load — the
//     declaration must bump the version explicitly.
//   - Structured-output validation: outputs are parsed + validated against a
//     JSON Schema declared in the prompt frontmatter. On failure, the
//     gateway retries with a "fix this JSON" prompt up to MaxRepairs times.
//   - Cost tracking: per-tenant + per-endpoint token counters exposed via
//     the platform/observability OTel meter.
//   - Optional response cache keyed by sha256(model, prompt_version,
//     normalized_inputs).
//
// The gateway never embeds API keys. Drivers resolve credentials via the
// platform/secrets package.
package llm

import (
	"context"
	"errors"
	"time"
)

// Gateway is the public surface generated handlers consume.
type Gateway interface {
	Call(ctx context.Context, req Request) (*Response, error)
}

// Request describes one LLM invocation.
type Request struct {
	// PromptName + PromptVersion select the sealed prompt body. Mismatched
	// version (or a body whose sha256 differs from the manifest) is a hard
	// error — the version must bump explicitly.
	PromptName    string
	PromptVersion string

	// Variables are interpolated into the prompt template per its
	// declared inputs schema. Extra keys cause a validation error.
	Variables map[string]any

	// TenantID + EndpointPath are recorded against the cost metrics so an
	// operator can attribute spend per customer + per route.
	TenantID     string
	EndpointPath string

	// CacheTTL > 0 enables response caching keyed by
	// sha256(model || prompt_version || normalized(variables)).
	CacheTTL time.Duration

	// MaxRepairs caps the structured-output retry loop. If zero, the
	// gateway uses its default (2). Set to -1 to disable repair.
	MaxRepairs int

	// Model overrides the prompt's model_default. Useful for evals.
	Model string
}

// Response is the gateway's structured return value.
type Response struct {
	Raw          string
	Parsed       any
	PromptTokens int
	OutputTokens int
	Model        string
	Cached       bool
	Repaired     bool
	LatencyMS    int64
}

// Driver is the per-provider transport. Drivers do not know about prompt
// sealing, output validation, caching, or cost tracking — those are gateway
// concerns layered on top.
type Driver interface {
	Name() string
	Complete(ctx context.Context, req DriverRequest) (DriverResponse, error)
}

// DriverRequest is the lowered form passed to a Driver after the gateway has
// loaded the prompt body and interpolated variables.
type DriverRequest struct {
	Model     string
	System    string
	Messages  []Message
	MaxTokens int
}

// Message is one chat-style message.
type Message struct {
	Role    string // "user" | "assistant"
	Content string
}

// DriverResponse is the lowered form returned by a Driver before validation
// + repair + caching.
type DriverResponse struct {
	Text         string
	PromptTokens int
	OutputTokens int
}

// ErrPromptSealMismatch is returned by the prompt loader when a prompt body's
// sha256 differs from its manifest. Generated handlers surface this as a 5xx.
var ErrPromptSealMismatch = errors.New("llm: prompt body sha256 does not match version manifest")
