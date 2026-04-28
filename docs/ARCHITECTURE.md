# vibeguard architecture

> The four pillars and one substrate.

## The substrate

Every pillar consumes the same two packages:

- `internal/ir/` — typed Application. Single source of truth for "what does this declaration say."
- `internal/rules/` — master-prompt rules (`code/` analyzers + `decl/` validators). Single source of truth for "what does the master prompt forbid."

Anything that touches a declaration touches `internal/ir`. Anything that enforces an iron rule touches `internal/rules`. There is no parallel datatype, no duplicate definition. This is why adding a 4th pillar (the LLM layer) didn't require changing pillars 1-3.

## Pillar 1 — The Compiler

A real compiler: parse → validate → render.

```
vibeguard.yaml
    │
    │  parser/v1 (decode + post-resolve relationships, primary keys, update fields)
    ▼
ir.Application  ◀── validate (schema + 9 semantic rules in registry)
    │
    │  render.Engine (Backends are pure: IR → []FileSpec)
    ▼
{ go/, sql/, k8s/, openapi/, tests/ } ──── render.Writer (atomic write + region marker preservation)
    │
    ▼
Compilable Go project + DDL + manifests + spec
```

Multi-target by construction. The Go backend, the SQL backend, the K8s backend, the OpenAPI backend all see the same IR and emit independently. Adding a TypeScript backend is a `internal/render/ts/` directory implementing the same `Backend` interface — zero IR changes, zero parser changes.

### Key files

- `internal/ir/ir.go` — the typed Application
- `internal/ir/step.go` — sealed Step interface (18 step kinds; backends dispatch on concrete type)
- `internal/parser/parser.go` — YAML → IR with post-decode pointer resolution
- `internal/validate/validate.go` — registry of 9 semantic rules
- `internal/render/render.go` — Engine + FileSet + Mode (Write/DryRun/Diff)
- `internal/render/golang/templates.go` — text/template generators

### Defense in depth

Routes are only registered for enabled CRUD verbs. The repository simply *does not generate* a Delete method when `delete: false`. Even if a malicious caller crafts a request, the route returns 404 before reaching handler code, and the missing method means the handler couldn't process it anyway.

The Update repository method is built dynamically from `Entity.CRUD.UpdateFields`. An attacker cannot smuggle a non-whitelisted field — the request struct's only fields are the whitelisted ones; anything else doesn't deserialize.

## Pillar 2 — The Platform SDK

Hand-written, versioned, the only thing generated code calls. Adapter pattern throughout: each `platform/<area>/` is an interface package; `platform/<area>/<impl>/` is one driver.

### Real RLS enforcement (the key fix)

The original prototype's `WithTenant` just stored a context value. It never told Postgres anything. RLS policies declared in vibeguard.yaml were documentation, not enforcement.

Branch 4-7's `platform/db/postgres` installs `pgxpool.BeforeAcquire` and `AfterRelease` hooks:

```go
cfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
    rc, ok := db.FromContext(ctx)
    if !ok { return true }
    _, err := conn.Exec(ctx,
        "SELECT set_config('app.tenant_id',$1,true), "+
        "       set_config('app.user_id',$2,true), "+
        "       set_config('app.role',$3,true)",
        rc.TenantID, rc.UserID, rc.Role)
    return err == nil
}
```

So `CREATE POLICY tenant_isolation ON tasks USING (tenant_id::text = current_setting('app.tenant_id', true))` actually fires on every query. `AfterRelease` clears the settings, preventing leaks between checkouts.

### Other adapters

- `platform/events/jetstream` — durable, at-least-once delivery, explicit ACK, durable consumers
- `platform/events/natscore` — legacy at-most-once, kept for compatibility, deprecated
- `platform/events/outbox.go` — transactional outbox primitive (drainer + JetStream cutover land in next branch)
- `platform/workflow/inmem` — in-process saga, fine for tests
- `platform/workflow/pgsaga` — durable Postgres-backed state machine (schema shipped, worker in next branch)
- `platform/llm` — gateway + Anthropic driver + sealed versioned prompts + structured-output validation + repair
- `platform/{cache, secrets, objectstore, featureflag, observability}` — interfaces in this branch; reference adapters in next branch

### Sealed prompts (the LLM-safety lever)

Prompts live as `<name>/<version>.md` with YAML frontmatter:

```
---
model_default: claude-haiku-4-5
inputs: { type: object, required: [task] }
output_schema: { type: object, required: [priority, reason] }
sha256: 4c9f5e...
---
You are an expert agile coach. Prioritize this task (1-10).
...
```

The loader hashes the body and refuses if `sha256` doesn't match the manifest. To change the prompt body you must run `vibeguard prompts seal <path>` — the human gesture that says "I meant to change this." Combined with the version field, prompts can never silently mutate; cost attribution and eval traces line up across deploys.

## Pillar 3 — The K8s Operator

Closes the loop between declaration and runtime. The operator owns an `Application.vibeguard.dev/v1` CRD whose spec mirrors `vibeguard.yaml` 1:1 (no schema fork) and reconciles five concerns:

1. **Migrations** — `golang-migrate` against the declared database
2. **NATS streams + consumers** — JetStream `CreateOrUpdateStream`/`Consumer`
3. **Workload** — Deployment + Service + NetworkPolicy + HPA + PDB (mirroring the hardened pattern in `k8s/deployment.yaml`)
4. **Secrets** — ExternalSecret refs to `ClusterSecretStore`
5. **RLS drift** — periodic `pg_policies` introspection vs. declaration

Status conditions: `DeclarationValid`, `MigrationsApplied`, `EventsConfigured`, `WorkloadReady`, `NoDrift`.

The operator never re-generates code. It only consumes the artifacts the generator produces (the app image, the `<app>-migrations` ConfigMap, the `Application` CR).

Branch 4-7 ships the CRD types + go.mod + scaffold. The reconciler logic itself is the focus of `4-7-followup`.

## Pillar 4 — The LLM Layer

Three pieces, sharing the substrate:

### MCP server (`cmd/vibeguard-mcp`)

JSON-RPC 2.0 over stdio (the MCP spec). One process per client workspace. Tools exposed in this branch:

- `validate_declaration(yaml)` — runs `parser.Parse + validate.Run`
- `generate_project(yaml, out_dir, module_path)` — runs the full render engine
- `lint_project(root_dir, format)` — runs the master-prompt analyzers

Resources:

- `vibeguard://prompts/master` — the master prompt, embedded
- `vibeguard://schema/declaration.json` — the JSON schema, embedded

In `4-7-followup` the operator's status API gets exposed as `query_runtime_state` and `propose_remediation`, completing the diagnose-and-fix loop.

### Linter (`cmd/vibeguard lint` and `cmd/vibeguard-lint`)

`golang.org/x/tools/go/analysis` analyzers. One file per rule. SARIF output for GitHub PR annotations.

Rules in branch 4-7:

| ID | Detection |
|---|---|
| VG001 | `fmt.Sprintf` / `+` building SQL passed to `db.Exec`/`Query`/`QueryRow` |
| VG002 | Unhandled errors on critical sinks: `db.Exec`, `events.Publish`, `http.Write`, `gin.JSON`, `outbox.EnqueueTx` |
| VG005 | `context.Background()`/`TODO()` inside HTTP request handlers |
| VG006 | Generated file edited outside `// vibeguard:region(...)` markers |

VG003 (no Argon2id), VG004 (handler missing auth), VG007 (repo skips tenant), VG008 (decl-without-version-bump) land in the next branch.

### LLM gateway (`platform/llm`)

The runtime side of the LLM layer. Generated `external_call: { service: openai }` steps compile to `gateway.Call(...)` invocations. The gateway:

- Loads sealed, versioned prompts
- Validates structured output against the prompt's `output_schema`
- On validation failure, retries with a "fix this JSON" follow-up up to `MaxRepairs`
- Records cost (tokens × price) per tenant + per endpoint
- Optional response cache keyed by `sha256(model || prompt_version || normalized_inputs)`

## How the four pillars compose

```
                  ┌────────────────────┐
                  │    LLM client      │ Claude Desktop / Code / Cursor
                  └──────┬─────────────┘
                         │ MCP stdio JSON-RPC
                         ▼
                  ┌────────────────────┐
                  │  vibeguard-mcp     │ Pillar 4
                  │  (validate, gen,   │
                  │   lint, resources) │
                  └──┬─────────────────┘
                     │ uses
        ┌────────────┴────────────┐
        ▼                         ▼
┌────────────────┐       ┌────────────────────────┐
│  internal/ir   │◀─────▶│   internal/rules       │  THE SUBSTRATE
│  parser/       │       │   code/, decl/, runtime/│
│  validate/     │       └────────────────────────┘
│  render/       │
└────────────────┘
        │ emits
        ▼
┌──────────────────────────────────────────┐
│  Generated Go service (compiles, tests)  │ Pillar 1 output
│  + SQL migrations + K8s manifests +      │
│  OpenAPI spec                             │
└─────┬───────────────────────┬────────────┘
      │ runs against           │ deployed by
      ▼                        ▼
┌─────────────────┐      ┌─────────────────┐
│ platform/       │      │ vibeguard-      │
│ db/events/      │      │ operator (CRD)  │
│ workflow/llm/...│      │                 │ Pillar 3
│ Pillar 2        │      └─────────────────┘
└─────────────────┘
```

The MCP client never speaks to the operator directly — it goes through the `query_runtime_state` tool (next branch). The operator never speaks to the LLM client. The platform SDK is the only thing generated code touches. Each layer has one job.

## Stability promises

- `internal/ir/` — shape-stable on branch 4-7. Field additions are minor. Removals/renames need an `apiVersion` bump and a parser/v<N>/migrate_to_v<N+1> shim.
- `platform/v1/...` — public root for semver-stable APIs. `platform/internal/` is non-stable.
- The CRD `Application.vibeguard.dev/v1` — schema mirrors `vibeguard.yaml`; spec evolutions are additive, with `v1beta1` deprecation periods for breaking changes.

## See also

- [`QUICKSTART.md`](QUICKSTART.md) — try-it-now walkthrough
- [`ROADMAP.md`](ROADMAP.md) — what's next, by branch
- [`adr/`](adr/) — Architecture Decision Records
