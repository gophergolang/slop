package ir

// Step is the sealed interface for the 18 step-DSL node kinds. Backends
// dispatch on the concrete type; new node types are additive only.
type Step interface {
	stepNode()
	StepName() string
}

// baseStep is embedded by every concrete step.
type baseStep struct{ Name string }

func (b baseStep) StepName() string { return b.Name }
func (b baseStep) stepNode()        {}

// ValidateStep validates the request body against a named schema.
type ValidateStep struct {
	baseStep
	Schema string
}

// LoadStep loads an entity by id from the database.
type LoadStep struct {
	baseStep
	Entity    string
	IDPath    string // path like ":id" or ".body.task_id"
	OutputVar string
}

// AuthorizeStep checks roles/conditions before continuing.
type AuthorizeStep struct {
	baseStep
	Roles     []string
	Condition string
}

// ExternalCallStep invokes a declared integration (openai, stripe, etc.).
type ExternalCallStep struct {
	baseStep
	Service        string
	Action         string
	PromptTemplate string
	OutputVar      string
}

// UpdateStep updates an entity's fields.
type UpdateStep struct {
	baseStep
	Entity string
	Fields map[string]string // value templates
}

// CreateStep inserts a new entity.
type CreateStep struct {
	baseStep
	Entity string
	Fields map[string]string
}

// DeleteStep deletes an entity (soft if the entity declares soft_delete).
type DeleteStep struct {
	baseStep
	Entity string
	IDPath string
}

// QueryStep executes a list query.
type QueryStep struct {
	baseStep
	Entity    string
	Where     string
	OutputVar string
}

// EmitEventStep publishes a domain event.
type EmitEventStep struct {
	baseStep
	Event   string
	Payload map[string]string
}

// ConsumeStep registers a subscription.
type ConsumeStep struct {
	baseStep
	Subject string
	Queue   string
}

// IfStep is a structured conditional.
type IfStep struct {
	baseStep
	Condition string
	Then      []Step
	Else      []Step
}

// ParallelStep runs branches concurrently.
type ParallelStep struct {
	baseStep
	Branches [][]Step
}

// SagaStep starts a durable saga.
type SagaStep struct {
	baseStep
	Name  string
	Input map[string]string
}

// CompensateStep registers a compensation closure.
type CompensateStep struct {
	baseStep
	For string // step name to compensate
}

// RetryStep wraps inner steps with retry semantics.
type RetryStep struct {
	baseStep
	MaxAttempts int
	Backoff     string
	Inner       []Step
}

// CacheStep caches the result of inner steps.
type CacheStep struct {
	baseStep
	Key   string
	TTL   string
	Inner []Step
}

// LogStep emits a structured log line.
type LogStep struct {
	baseStep
	Level   string
	Message string
}

// PolicyStep applies a declarative policy.
type PolicyStep struct {
	baseStep
	Name string
}

// TransactionStep wraps inner steps in a single tx.
type TransactionStep struct {
	baseStep
	Inner []Step
}

// ReturnStep is the terminal step that produces the HTTP response.
type ReturnStep struct {
	baseStep
	Status int
	Body   string
}
