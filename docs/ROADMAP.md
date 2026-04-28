# Roadmap

What ships when. Branch `4-7` is the foundation — the four pillars exist and the killer demo (declaration → typed IR → compilable Go → master-prompt lint → MCP) works end-to-end. Everything below is what makes the foundation production-ready.

## Branch `4-7` (this branch) — Foundation

**Compiler:** typed IR, polymorphic YAML parser, 9 semantic validators, render engine, Go + SQL + K8s + OpenAPI backends. Works against the full `fixtures/sample_vibeguard.yaml` and emits a project that compiles.

**Platform SDK:** real RLS enforcement (`pgxpool.BeforeAcquire` + `set_config`), JetStream durable adapter, in-process saga, sealed versioned prompts + Anthropic LLM driver with structured-output validation + repair loop. Reference adapters for db, events, workflow, llm. Interface scaffolds for cache, secrets, objectstore, featureflag, observability.

**Operator:** CRD types (`Application.vibeguard.dev/v1`), Go module scaffold, kubebuilder PROJECT layout. Reconciler is a no-op stub.

**LLM Layer:** MCP server (stdio JSON-RPC) with 3 tools and 2 resources. Linter binary + 4 analyzers (VG001/VG002/VG005/VG006) with SARIF output.

**Tests:** parser round-trip, validate negative cases, render engine, prompt seal mismatch.

## Branch `4-7-followup` — Make the Operator real

Goal: `Application` CRs reach all 5 conditions True against a real cluster.

- `operator/internal/controller/application_controller.go` — full reconciler using `sigs.k8s.io/controller-runtime`
- `operator/internal/reconcilers/{nats,migrations,workload,secrets,rls}.go` — one file per concern
- `operator/internal/webhook/application_webhook.go` — validating admission webhook (same validators as `vibeguard validate`)
- envtest + ginkgo/gomega controller tests
- kuttl end-to-end tests on kind: real Postgres, real NATS JetStream
- Helm chart at `operator/charts/vibeguard-operator/`
- The MCP server's `query_runtime_state` tool wired to read operator status

## Branch `4-7-durability` — Make events + sagas durable for real

- `platform/events/jetstream/drainer.go` — outbox → JetStream worker, advisory-lock leader election, exponential backoff, dead-letter handling
- `platform/workflow/pgsaga/worker.go` — saga state machine recovery loop, FOR UPDATE SKIP LOCKED claim
- Generator switch: emit `outbox.EnqueueTx` calls in handlers behind `spec.platform.events.outbox=true`, then default true after one release
- testcontainers integration tests (Postgres + NATS JetStream)
- `platform/observability/otel.go` — real OTLP HTTP/gRPC exporter wiring

## Branch `4-7-eval` — The benchmark

The marketing wedge: vibeguard becomes a public benchmark for "how good is each LLM at building production systems?"

- `eval/cmd/vibeguard-eval` — runner with `--models claude-opus-4.7,gpt-5,llama4-70b`
- `eval/harness/` — task loop, content-addressed response cache, parallel exec
- `eval/graders/` — `schema_valid`, `rule_compliance`, `structural_similarity`, `compiles`, `tests_pass`, `llm_judge`
- `eval/tasks/{decl_from_readme,fill_logic_steps,remediate_runtime_error}/`
- Nightly GH Action publishing leaderboard to gh-pages

## Branch `4-7-rules` — Cover the rest of the master prompt

The remaining static analyzers:

- VG003 — password hashing must call `argon2.IDKey`
- VG004 — every `auth_required: true` endpoint registers a route with auth middleware (cross-references the declaration)
- VG007 — repository methods missing tenant filter
- VG008 — declaration changed without `metadata.version` bump (with `--git`)

Plus VG006 promotion to checksum-based detection: `.vibeguard-checksums.json` written by the generator, lint compares.

## Branch `4-7-polyglot` — TypeScript backend

If a real user asks for it. The IR is shape-stable so this is purely additive: `internal/render/ts/` with `Hono` + `Drizzle` + `nats.js` as the reference stack. The MCP server's `generate_project` gets a `target` parameter. Same declaration → either Go or TS service.

## Branch `4-7-saas` — Hosted MCP

When personal-workspace stdio is no longer enough:

- SSE transport on the same `mcp.Server`
- OIDC sidecar proxy
- Per-organization quota + cost dashboards
- Multi-tenant `Application` CRs reconciled by a single operator across customer namespaces

## Out of scope (deliberate non-goals)

- **Crossplane-style infrastructure provisioning.** Vibeguard owns app lifecycle, not Postgres/NATS/Redis lifecycle. The declaration assumes those exist; the operator wires apps to them.
- **Custom scheduler.** K8s + JetStream is plenty. Building a scheduler is multi-year and a distraction.
- **All-in-one frameworks.** Vibeguard composes existing infra; it does not replace `golang-migrate`, `pgx`, `nats.go`, controller-runtime, gin, etc. Each tool stays best-in-class for its job.

## Stability promises across branches

- `internal/ir/` is shape-stable. Field additions only.
- `platform/v1/` follows semver — only deprecation, never removal, between major versions.
- The `Application.vibeguard.dev/v1` CRD spec evolves additively; breaking changes get a `v1beta1` overlap period.
