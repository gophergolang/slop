// Package ir is the typed intermediate representation for vibeguard
// declarations. Frontends (today: YAML; future: TS DSL, JSON-from-LLM) parse
// into this IR; backends (Go, SQL, K8s, OpenAPI, tests) consume it.
//
// The IR is the contract that lets pillar 1 (the compiler) and pillar 4 (the
// LLM layer) share one definition of "what does this declaration say." Every
// validator, generator, and lint rule reads from this IR — never from the
// raw YAML.
//
// Stability: shape-stable on branch 4-7. Field additions are minor; removals
// or renames require an apiVersion bump and a parser/v<N>/migrate_to_v<N+1>
// shim.
package ir

// Application is the root of a parsed declaration.
type Application struct {
	APIVersion string
	Kind       string
	Metadata   Metadata
	Global     Global
	Modules    []Module
}

// Metadata mirrors the declaration's metadata block.
type Metadata struct {
	Name        string
	Version     string
	Description string
	Compliance  []string
}

// Global is cross-cutting configuration that applies to every module.
type Global struct {
	MultiTenancy MultiTenancy
}

// MultiTenancy controls how generated code isolates tenants.
type MultiTenancy struct {
	Enabled       bool
	TenantIDField string
	Isolation     string // "row" | "schema" | "database"
}

// ModuleType is one of the declared module categories.
type ModuleType string

const (
	ModuleIdentity    ModuleType = "identity"
	ModuleBusiness    ModuleType = "business"
	ModuleIntegration ModuleType = "integration"
	ModuleRealtime    ModuleType = "realtime"
	ModuleAI          ModuleType = "ai"
)

// Module is one cohesive feature area.
type Module struct {
	Name         string
	Type         ModuleType
	Entities     []*Entity
	Integrations []Integration
	Policies     Policies
	Events       []Event
}

// Sensitivity classifies the data an entity holds.
type Sensitivity string

const (
	SensitivityPublic       Sensitivity = "public"
	SensitivityInternal     Sensitivity = "internal"
	SensitivityConfidential Sensitivity = "confidential"
	SensitivityRestricted   Sensitivity = "restricted"
)

// Entity is one persistent type. The PrimaryKey pointer is resolved
// post-parse and points into the Fields slice.
//
// Parents express the leap-style tree-of-data: an entity may declare zero or
// more parents by entity name. The post-parse resolver fills ParentRefs.
// Generated URL paths, foreign-key constraints, cascade rules, and frontend
// navigation derive from this tree.
type Entity struct {
	Name          string
	Table         string
	Sensitivity   Sensitivity
	Fields        []*Field
	PrimaryKey    *Field
	Relationships []Relationship
	Parents       []string  // declared parent entity names
	ParentRefs    []*Entity // resolved post-parse
	CRUD          CRUD
	API           API
	Module        *Module // back-pointer set during parse
	BusinessRules []string
	SoftDelete    bool
}

// Field is one column on an entity.
type Field struct {
	Name       string
	Type       FieldType
	Nullable   bool
	Unique     bool
	Primary    bool
	EnumValues []string
	Default    *string
	Validators []string
	DBHints    DBHints
}

// FieldType is the declared type. The render backends map this to language
// types (Go) and column types (SQL).
type FieldType string

const (
	FieldString    FieldType = "string"
	FieldText      FieldType = "text"
	FieldInt       FieldType = "int"
	FieldBigInt    FieldType = "bigint"
	FieldBool      FieldType = "bool"
	FieldUUID      FieldType = "uuid"
	FieldTimestamp FieldType = "timestamp"
	FieldEnum      FieldType = "enum"
	FieldJSON      FieldType = "json"
	FieldDecimal   FieldType = "decimal"
)

// DBHints carries persistence hints declared per field.
type DBHints struct {
	Index bool
}

// Relationship describes a foreign-key edge to another entity.
type Relationship struct {
	To         string  // declared entity name
	Type       string  // "belongs_to" | "has_many" | "has_one"
	ForeignKey string
	Resolved   *Entity // pointer after post-parse resolution
}

// CRUD declares which operations the generator emits + which fields are
// settable on update.
type CRUD struct {
	Create       bool
	Read         bool
	List         bool
	Update       []string // field whitelist
	Delete       bool
	SoftDelete   bool
	UpdateFields []*Field // pointer-resolved post-parse
}

// API holds HTTP-layer concerns.
type API struct {
	BasePath        string
	AuthRequired    bool
	RolesAllowed    []string
	RateLimit       string
	CustomEndpoints []CustomEndpoint
}

// CustomEndpoint is a non-CRUD route declared either with a structured logic
// block (the step DSL) or with a Node reference (a developer-authored Go
// function the generator wraps with secure boilerplate). When Node is set it
// takes precedence; Logic remains supported for purely declarative endpoints.
type CustomEndpoint struct {
	Path         string
	Method       string
	Description  string
	Request      string
	Response     string
	AuthRequired bool
	RolesAllowed []string
	Node         string // "<package>.<Func>" — handed to the Go backend's node renderer
	Logic        Logic
}

// Logic is the step-DSL declaration.
type Logic struct {
	Description string
	Steps       []Step
}

// Integration is a third-party connector (openai, stripe, ...).
type Integration struct {
	Name     string
	Config   map[string]string
	Features []string
}

// Policies wraps RLS + other declarative policies.
type Policies struct {
	RowLevelSecurity []RLSPolicy
}

// RLSPolicy is one Postgres CREATE POLICY clause derived from the declaration.
type RLSPolicy struct {
	Entity    string
	Condition string   // raw SQL expression with {tenant_id}/{user_id} interpolations
	ApplyTo   []string // "select" | "update" | "delete"
}

// Event is a domain event the application publishes.
type Event struct {
	Name      string
	Trigger   string // "after_create" | "after_update" | ...
	Entity    string
	Condition string
	PublishTo string // "internal" | external subject
}
