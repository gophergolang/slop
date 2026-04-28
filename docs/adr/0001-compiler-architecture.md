# ADR 0001 — Compiler-style architecture for the generator

## Status

Accepted (branch 4-7).

## Context

The original prototype's `cmd/vibeguard/main.go` (152 lines) read 6 fields out of ~150 in a vibeguard.yaml and produced a hardcoded hello-world handler. The full declaration described a complete app — fields, CRUD whitelists, RLS policies, custom endpoints with a step-DSL — but the generator only saw the top-level metadata. The mismatch between concept doc and executable code was the single biggest credibility hole in the project.

We need a generator that:

1. Reads the entire declaration (every field, every step, every relationship)
2. Produces compilable Go code, SQL migrations, K8s manifests, and an OpenAPI spec
3. Is structured so additional output targets (TypeScript, Python) can be added later without rewriting the parser
4. Emits diagnostics that name a specific rule + source path, not a stack trace
5. Is testable with golden-file fixtures

## Decision

We adopt a **compiler-style architecture** with three phases and a stable typed IR in the middle:

```
vibeguard.yaml ──► parser ──► IR ──► validate ──► render ──► files
                              ▲
                              │
                              one shape, one source of truth
```

Each phase is its own package:

- `internal/parser/` — YAML decoding (apiVersion-routed). Tolerant: unknown fields ignored, missing optional fields default. Strict validation is the next phase's job.
- `internal/ir/` — typed `Application` with sealed `Step` interface. Pointer-resolved relationships, primary keys, and update field whitelists. The contract every other package depends on.
- `internal/validate/` — registry of semantic rules. Each rule is one file, takes `*ir.Application`, returns `[]Issue`. Adding a rule is a one-file PR.
- `internal/render/` — engine + Mode (Write/DryRun/Diff). Backends are pure: `Plan(*ir.Application) (FileSet, error)`.
- `internal/render/{golang,sql,k8s,openapi,tests}/` — one backend per output target. Each produces an in-memory `FileSet`; the engine writes atomically.

## Consequences

**Good:**

- The IR is the single source of truth for "what does this declaration say." Validators, the generator, the linter, the MCP server, and the operator all consume it.
- Adding a backend (e.g. TypeScript) is a sibling directory, no IR or parser changes.
- Adding a validation rule is one file in `internal/validate/rules/`.
- Apirevisions are routed in `parser/v<N>/` — backends only ever target the *current* IR.
- Golden-file tests work because backends are pure.

**Tradeoffs:**

- More packages than a one-file generator. The seam between parser and IR forces an explicit contract that never gets short-circuited.
- The IR carries pointer-resolved fields (`Entity.PrimaryKey *Field`, `CRUD.UpdateFields []*Field`) that the YAML doesn't have. This is intentional — backends should never re-resolve relationships at emit time.
- Sealed `Step` interface means adding a new step kind requires a code change in both `internal/ir/step.go` and the relevant backends. This is a feature, not a bug — the schema and the generator stay in lockstep.

## Alternatives considered

- **Stay with one big main.go**, grow it field-by-field. Rejected: the existing prototype already proved this doesn't scale; the diff between intent and implementation only grows.
- **`dave/jennifer` for everything.** A Go-AST programmatic builder gives compile-time safety on emitted Go but is the wrong abstraction for SQL migrations, YAML manifests, and JSON specs. We use it only for the step-DSL emitter (`internal/render/golang/emit_steps.go`) where the variable bindings would be SQL-injection-prone in a string template; everything else uses `text/template`.
- **Protobuf as the canonical IR.** Tempting for cross-language reuse, but proto can't model the pointer-resolved fields cleanly, and our backends are Go for the foreseeable future. We reserve `internal/ir/ir.proto` as a sibling artifact to be emitted from Go — never the canonical definition.

## See also

- [`docs/ARCHITECTURE.md`](../ARCHITECTURE.md) — the four pillars and how this compiler fits
- [`docs/ROADMAP.md`](../ROADMAP.md) — when polyglot backends would land
