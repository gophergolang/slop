# VibeGuard Declaration Standard v0.1
## The Missing Layer for Reliable, Secure, Auditable AI Code Generation

### Why This Exists
Natural language is ambiguous. Even with the best prompt, an LLM can:
- Forget to implement full CRUD for an entity
- Accidentally expose dangerous operations (mass delete, admin endpoints)
- Inconsistently apply security rules across modules
- Generate different patterns on every run

**The Declaration Standard** solves this by forcing a **formal, machine-readable, human-reviewable spec** first.

**Flow**:
1. User describes app in natural language
2. VibeGuard (LLM) produces a **strict YAML declaration**
3. User reviews / edits the YAML (especially CRUD whitelists and auth requirements)
4. VibeGuard then generates **100% of the code strictly from the declaration** — no deviations allowed
5. The declaration is committed alongside the code (single source of truth)

This turns "vibe coding" into **declarative secure engineering**.

---

### Core Principles of the Declaration

- **Explicit over Implicit** — Everything must be declared. No "magic" CRUD.
- **Whitelist by Default** — Most operations are **disabled** unless explicitly allowed (security-first).
- **Security & Compliance First** — Every entity declares auth requirements, data sensitivity, retention policy.
- **Go-Native** — Types map cleanly to Go structs, `pgx`, `validator/v10` tags, and Gin handlers.
- **Auditable** — The YAML can be diffed, reviewed in PRs, and used to generate OpenAPI + docs automatically.
- **Extensible** — Supports custom business logic endpoints, integrations, and non-functional requirements.

### Tree-of-data: `parents`

An entity may declare zero or more `parents` (entity names). The parent graph is a DAG (typically a tree) and drives:

- **Nested URL paths** — `/api/v1/teams/:team_id/tasks/:id` is derived when `Task.parents: [Team]` and `api.base_path` is unset. An explicit `api.base_path` always overrides the derived path.
- **Foreign keys** — the SQL backend emits `FOREIGN KEY (<parent_table_singular>_id) REFERENCES <parent_table>(id) ON DELETE CASCADE` for each parent edge whose conventional FK column is present on the child.
- **Tenant boundaries** — a tenant-bound child cannot have a non-tenant-bound parent (`VG-VAL-012`). Cycles (`VG-VAL-011`) and missing parents (`VG-VAL-010`) are rejected before code generation.
- **Frontend navigation** — the Next.js generator (planned) walks the tree to produce nested admin routes and breadcrumbs.

Example:

```yaml
- name: Comment
  table: comments
  parents: [Task]
  fields:
    - { name: id, type: uuid, primary: true }
    - { name: tenant_id, type: uuid }
    - { name: task_id, type: uuid }
    - { name: body, type: text }
```

### Business-logic nodes (`node:`)

Custom endpoints can dispatch to a developer-authored Go function instead of a YAML step DSL. Set `node: <pkg>.<Func>` on a `custom_endpoint`. The generator emits the secure wrapper (parse, validate, auth, tenant context, transaction, observability span) and calls the named function at the one point where business logic actually lives. The developer fills in the function body; the framework owns everything around it.

```yaml
custom_endpoints:
  - path: /api/v1/tasks/:id/prioritize
    method: POST
    node: tasks.Prioritize
    request: PrioritizeRequest
    response: PrioritizeResponse
    auth_required: true
    roles_allowed: [owner, admin, member]
```

When both `node:` and `logic.steps:` are set, `node:` wins. The step DSL remains supported for purely declarative endpoints.

---

## Roadmap & Production Architecture Vision (Strategic Direction)

### Honest Diagnosis — The Real Problem Most AI Code Tools Ignore
Most AI-assisted development tools fail in production for one fundamental reason:

**They generate too much code that humans don't actually own or trust.**

When 70-90% of a codebase is LLM output, you eventually get:
- Senior engineers who refuse to maintain it ("this feels like magic that will bite us")
- Painful, low-value code reviews
- Fear of making changes (because regeneration might introduce subtle bugs)
- Infrastructure lock-in (you can't easily move from Postgres to CockroachDB or TiDB)
- Compliance and security teams who don't trust the generated code
- Poor long-term velocity

**VibeGuard's Core Thesis** (the honest foundation we are building on):
> Use AI aggressively where it is strongest (repetitive, rule-based, security-critical scaffolding and boilerplate) while keeping humans firmly in control of everything that actually determines long-term success in production systems (complex business logic, infrastructure decisions, operational control, and code ownership).

This is the only path that produces something real engineering organizations will actually adopt and respect.

### Recommended Long-term Architecture (The Real Roadmap)

For **true production systems** at scale (especially SaaS with multiple teams, compliance needs, and changing infrastructure), we need **three clear layers**:

#### Layer 1: Declaration (Human + AI Collaboration Surface)
- `vibeguard.yaml` — the contract
- Security policies, business intent, CRUD whitelists, logic steps
- **This layer stays exactly as we have it** (or gets richer)

#### Layer 2: Operator / Runtime Abstraction Layer (The Missing Piece)
This is the **key evolution** for production viability.

Instead of generating raw `pgxpool` or `kafka-go` calls directly, the generated code calls into **well-maintained, hand-written operator libraries**:

- `github.com/vibeguard/db` — abstract database client (Postgres today, but pluggable to CockroachDB, TiDB, PlanetScale, etc.)
- `github.com/vibeguard/events` — abstract event bus (in-memory for dev → Kafka/NATS/RabbitMQ in prod)
- `github.com/vibeguard/cache`, `github.com/vibeguard/observability`, `github.com/vibeguard/auth`, etc.

**These operators are versioned, tested, and owned by humans** (or a platform team).

The first version of the Platform SDK now lives in `platform/`:
- `events` — NATS-powered publisher/subscriber with standard `Event` envelope
- `db` — Tenant-aware Postgres with RLS support
- `workflow` — Saga + compensation runner

This is the foundation for the full event-driven + operator-per-app vision.

**Why this matters for production**:
- You can change your database or event backend **without touching generated code**.
- Much smaller generated surface area → humans can actually read and trust the code.
- Better operational control and observability.

#### Layer 3: Generated Application Code (Thin & Focused)
- Only generates **thin handlers + repositories + wiring**
- All heavy lifting (DB access, events, caching, auth, observability) goes through the Operator layer
- Humans own the important business logic and can easily modify it

### Do We Need "DB Operators" and "Event Operators"?

**Yes — but not as full Kubernetes Operators** (at least not initially).

**Recommended approach**:
- **Library Operators** (Go packages) first — this gives 80% of the benefit with much less complexity.
- Later (Phase 4), if a company wants a full internal developer platform, we can evolve to real Kubernetes Operators + a control plane (similar to how Crossplane, Argo CD, or Backstage work).

**Current Recommendation for the Roadmap**:

| Phase | Focus | Goal | Viability |
|-------|-------|------|---------|
| **Phase 1** (Now) | Declaration + Logic DSL + Full Code Generation | Fast, secure starting point | Excellent for MVPs & small teams |
| **Phase 2** (Next) | Introduce thin **Operator Layer** (`vibeguard/db`, `vibeguard/events`) | Better maintainability + infrastructure flexibility | **Critical for production** |
| **Phase 3** | Higher-level constructs (Saga, Workflow, Compensation, Reusable Policies) | Express complex distributed business logic | High value |
| **Phase 4** (Optional) | Full Control Plane + Kubernetes Operators | Self-service internal platform for large orgs | Only if you need it |

### Developer Experience (devX) Considerations

**Good devX means**:
- Humans feel **in control**, not fighting generated code
- Small, reviewable diffs when business logic changes
- Easy to debug (clear stack traces, good logging from operators)
- Can run locally with in-memory operators (Postgres + in-memory events)
- Can promote to production by just changing config (same code, different operators)

**The Operator Layer is the key to achieving this.**

### My Honest Recommendation

1. **Keep the Declaration Layer** as the primary interface (it's excellent).
2. **Prioritize building the Operator Layer** (`vibeguard/db` + `vibeguard/events`) **before** making the logic DSL even more complex.
3. Make the generated code **thin** — it should mostly be "call the operator, handle errors, return".
4. Only after the Operator Layer is solid, expand the DSL with more advanced constructs (sagas, compensation, etc.).

This path gives us:
- Strong security & auditability (declaration)
- Excellent long-term maintainability (humans own real code)
- True production flexibility (swap infrastructure easily)
- Great developer experience (feels like normal Go development with superpowers)

---

## Advanced Features (v0.2+)

```yaml
apiVersion: vibeguard.dev/v1
kind: Application
metadata:
  name: string                  # kebab-case, becomes Go module name
  version: string               # semver
  description: string
  owner: string
  compliance:
    - SOC2
    - GDPR
    - HIPAA
    - PCI-DSS

spec:
  modules:                      # list of bounded contexts
    - name: string              # e.g. "auth", "tasks", "billing"
      type: identity | business | integration | realtime
      description: string

      entities:                 # database tables / domain models
        - name: string          # PascalCase, becomes Go struct (e.g. Task)
          table: string         # snake_case table name
          description: string
          sensitivity: public | internal | confidential | restricted

          fields:
            - name: string
              type: uuid | string | int | int64 | float64 | bool | timestamp | json | enum | text
              primary: boolean
              unique: boolean
              nullable: boolean
              default: any
              validate:           # validator/v10 rules
                - required
                - email
                - max=200
                - min=3
                - oneof=todo in_progress done
              db:
                index: boolean
                unique_index: boolean
                check: string     # SQL CHECK constraint

          relationships:
            - to: string          # entity name
              type: belongs_to | has_many | has_one | many_to_many
              foreign_key: string
              on_delete: cascade | restrict | set_null

          crud:                   # WHITELIST — only what is true is generated
            create: boolean
            read: boolean
            list: boolean
            update:                 # either true (all fields) or list of allowed fields
              - title
              - status
            delete: boolean
            soft_delete: boolean    # adds deleted_at column + filters

          api:
            base_path: string       # e.g. /api/v1/tasks
            auth_required: boolean
            roles_allowed: [admin, member, owner]   # RBAC
            rate_limit: string      # "100/minute" or "1000/hour"
            public_endpoints: [list]  # only these are unauthenticated (rare)
            custom_endpoints:       # non-CRUD business logic
              - path: /tasks/:id/assign
                method: POST
                description: "Assign task to user"
                request: AssignTaskRequest
                response: Task
                auth_required: true
                roles_allowed: [admin, member]

          business_rules:
            - "A task cannot be marked done if it has no due date"
            - "Only the owner or assignee can update status"

      integrations:
        - name: stripe | openai | sendgrid | ...
          config:
            webhook_secret_env: STRIPE_WEBHOOK_SECRET
            features: [subscriptions, invoices]

      non_functional:
        caching:
          enabled: boolean
          strategy: write_through | write_behind | cache_aside
          ttl_seconds: int
        observability:
          tracing: true
          metrics: true
          structured_logging: true
        scaling:
          stateless: true
          horizontal: true
```

---

### Example Declaration (Task Management SaaS)

```yaml
apiVersion: vibeguard.dev/v1
kind: Application
metadata:
  name: task-management-saas
  version: "1.2.0"
  description: "Real-time collaborative task platform with AI prioritization"
  compliance: [SOC2, GDPR]

spec:
  modules:
    - name: auth
      type: identity
      entities:
        - name: User
          table: users
          sensitivity: confidential
          fields:
            - name: id
              type: uuid
              primary: true
            - name: email
              type: string
              unique: true
              validate: [required, email]
            - name: password_hash
              type: string
              validate: [required]
            - name: role
              type: enum
              values: [admin, member, viewer]
              default: member
          crud:
            create: true
            read: true
            list: false          # never list all users
            update: [email, role]
            delete: false        # soft delete only via admin job
          api:
            base_path: /auth
            auth_required: false
            custom_endpoints:
              - path: /auth/login
                method: POST
                request: LoginRequest
                response: AuthResponse
              - path: /auth/refresh
                method: POST

    - name: tasks
      type: business
      entities:
        - name: Task
          table: tasks
          sensitivity: internal
          fields:
            - name: id
              type: uuid
              primary: true
            - name: title
              type: string
              validate: [required, max=200]
            - name: description
              type: text
              nullable: true
            - name: status
              type: enum
              values: [todo, in_progress, done, archived]
              default: todo
            - name: due_date
              type: timestamp
              nullable: true
            - name: user_id
              type: uuid
              validate: [required]
          relationships:
            - to: User
              type: belongs_to
              foreign_key: user_id
              on_delete: cascade
          crud:
            create: true
            read: true
            list: true
            update: [title, description, status, due_date]
            delete: false        # soft delete only
            soft_delete: true
          api:
            base_path: /api/v1/tasks
            auth_required: true
            roles_allowed: [admin, member]
            rate_limit: "200/minute"
            custom_endpoints:
              - path: /api/v1/tasks/:id/prioritize
                method: POST
                description: "AI-powered prioritization (calls OpenAI)"
                request: PrioritizeRequest
                response: Task
                auth_required: true

    - name: billing
      type: integration
      integrations:
        - name: stripe
          config:
            webhook_secret_env: STRIPE_WEBHOOK_SECRET
            features: [subscriptions, invoices, usage-based]
```

---

### How VibeGuard Must Use the Declaration (Mandatory Rules in Prompt)

When the LLM sees a user request, it **MUST**:

1. **Generate the Declaration First** (as a fenced YAML block)
2. **Explicitly ask the user** to review and approve/edit the declaration (especially CRUD whitelists and `roles_allowed`)
3. Only after approval (or in subsequent turn when user says "use this declaration"), generate all code **strictly derived from it**:
   - Only generate repository methods for whitelisted CRUD operations
   - Only generate Gin handlers/routes for whitelisted endpoints + custom_endpoints
   - Only expose fields listed in `update: [...]`
   - Apply `soft_delete` filters automatically when declared
   - Generate exact `validate` tags from the spec
   - Generate RBAC middleware checks from `roles_allowed`
   - Generate OpenAPI spec + Swagger UI from the declaration
   - Never add extra endpoints or operations not present in the YAML

This guarantees **zero drift** between spec and implementation.

---

### Benefits for Security & Maintainability

- **Attack Surface Control**: `delete: false` on User entity prevents accidental account deletion endpoints.
- **Least Privilege**: `list: false` on User prevents data enumeration attacks.
- **Compliance Evidence**: The YAML itself becomes audit artifact ("we declared and enforced that PII is never listed").
- **Fast Iteration**: User can edit the YAML and say "regenerate from this spec" — much faster than describing changes in prose.
- **Future CLI**: `vibeguard generate --from declaration.yaml` can become fully deterministic (or still use LLM for complex business logic while staying inside the spec).

---

### Next Evolution (v0.2 ideas)

- Add `policies:` section for row-level security (RLS) rules
- Add `events:` for event-driven architecture (publish/subscribe)
- Support multiple API versions with deprecation
- Generate Terraform / Kubernetes manifests from the same declaration

---

## Advanced Features (v0.2+)

### 1. Row-Level Security (RLS) Policies
Declare fine-grained access control that translates to database policies or application-level filters.

```yaml
spec:
  modules:
    - name: tasks
      policies:
        row_level_security:
          - entity: Task
            condition: "user_id = {user_id} OR EXISTS (SELECT 1 FROM team_members WHERE team_id = tasks.team_id AND user_id = {user_id})"
            apply_to: [select, update, delete]
          - entity: Task
            condition: "tenant_id = {tenant_id}"
            apply_to: [select, update, delete]
```

**Generated effect**: Automatic `WHERE` clauses in all repository queries + PostgreSQL RLS policies when using `pgx` + `SET app.current_user_id = 'xxx'`.

### 2. Event-Driven Architecture
Declare domain events that should be published on entity changes.

```yaml
spec:
  modules:
    - name: tasks
      events:
        - name: TaskCreated
          trigger: after_create
          entity: Task
          publish_to: internal          # or kafka, rabbitmq, webhook
          payload:
            include: [id, title, user_id, status]
        - name: TaskCompleted
          trigger: after_update
          entity: Task
          condition: "old.status != 'done' AND new.status = 'done'"
          publish_to: webhook
          endpoint: "https://hooks.slack.com/..."
```

**Generated effect**: Event structs + publisher interfaces + handlers in `internal/events/`.

### 3. sqlc Support (Type-Safe SQL)
Instead of raw `pgx` queries, declare that you want `sqlc` for compile-time safe queries.

```yaml
spec:
  modules:
    - name: tasks
      entities:
        - name: Task
          sqlc:
            enabled: true
            queries: queries/tasks.sql     # path to sqlc query file
            generate:
              - GetTaskByID
              - ListTasksByUser
              - UpdateTaskStatus
```

VibeGuard will generate the `sqlc.yaml` config + run `sqlc generate` during scaffolding.

### 4. Multi-Tenancy
Built-in support for SaaS multi-tenant applications.

```yaml
spec:
  global:
    multi_tenancy:
      enabled: true
      tenant_id_field: tenant_id
      isolation: row          # row | schema | database

  modules:
    - name: tasks
      entities:
        - name: Task
          fields:
            - name: tenant_id
              type: uuid
              validate: [required]
              db:
                index: true
          crud:
            # tenant_id is automatically filtered in all queries
            list: true
```

**Generated effect**:
- Every entity gets `tenant_id` filter in repositories
- Middleware automatically extracts tenant from JWT or subdomain
- `SET app.current_tenant_id` for RLS

---

## Handling Complex Business Logic in Handlers

### The Problem
CRUD + simple custom endpoints cover many cases, but real applications have **complex business logic** inside handlers:
- Multi-step processes (validate → load → check permissions → call AI → update → emit events → notify)
- Conditional branching
- External service orchestration
- Transactional boundaries

Without structure, the LLM has to "guess" the implementation, which risks inconsistency and bugs.

### Solution: Structured `logic` Steps

We expanded the spec with a `logic` section under `custom_endpoints`. It uses a **step-based DSL** that VibeGuard interprets into clean, idiomatic Go code.

#### Cleaned & Future-Proofed Logic DSL (v0.5)

The DSL has been cleaned and strengthened. It is now **powerful, consistent, and designed for the future event-driven + operator-per-app architecture**.

**Core Step Types (Recommended Set)**

| Type            | Purpose                                              | Key Fields                              | Future-Proof Note |
|-----------------|------------------------------------------------------|-----------------------------------------|-------------------|
| `validate`      | Input validation                                     | `schema`                                | — |
| `load`          | Load entity (tenant + RLS enforced)                  | `entity`, `id_path`                     | — |
| `authorize`     | RBAC + custom condition                              | `condition`                             | — |
| `external_call` | Call any external service (AI, Stripe, etc.)         | `service`, `action`, `prompt_template`  | Works with future agent runtimes |
| `update` / `create` / `delete` | Standard mutations                     | `entity`, `fields` / `id_path`          | — |
| `query`         | Safe read query (tenant filter required)             | `query`, `args`                         | — |
| `emit`          | Publish event (NATS topic ready)                     | `event`, `topic`, `payload`             | Core for event-driven vision |
| `consume`       | Subscribe / react to event                           | `event`, `topic`                        | Enables reactive flows |
| `retry`         | Retry wrapper with backoff                           | `attempts`, `backoff`                   | Essential for reliability |
| `parallel`      | Run steps concurrently                               | `steps`                                 | High throughput |
| `saga`          | Multi-step transaction with compensation             | `steps`, `compensate`                   | Critical for distributed systems |
| `compensate`    | Undo previous steps on failure                       | `steps`                                 | Paired with saga |
| `policy`        | Enforce runtime security/compliance policy           | `policy`                                | Runtime guardrails |
| `cache`         | Read/write cache (Redis-ready)                       | `cache_key`, `ttl`, `value`             | Performance |
| `log`           | Structured logging                                   | `level`, `message`                      | Observability |
| `if`            | Conditional branching                                | `condition`, `then`, `else`             | — |
| `transaction`   | DB transaction wrapper                               | `steps`, `mode`                         | — |
| `return`        | HTTP response                                        | `status`, `body`                        | — |
| `custom`        | Escape hatch for complex logic                       | `description`                           | — |

#### Example: AI Task Prioritization with Full Logic

```yaml
custom_endpoints:
  - path: /api/v1/tasks/:id/prioritize
    method: POST
    description: "AI-powered task prioritization"
    request: PrioritizeRequest
    response: Task
    logic:
      description: |
        1. Validate input
        2. Load task with tenant + RLS check
        3. Verify user can prioritize (assignee or creator)
        4. Call OpenAI to get priority + reason
        5. Update task priority and status
        6. Emit TaskPrioritized event
        7. Return updated task
      steps:
        - name: validate_input
          type: validate
          schema: PrioritizeRequest

        - name: load_task
          type: load
          entity: Task
          id_path: ":id"

        - name: check_permissions
          type: authorize
          roles: [admin, member]
          condition: "task.assignee_id == user.id OR task.created_by == user.id"

        - name: call_ai
          type: external_call
          service: openai
          action: chat_completion
          prompt_template: |
            You are an expert agile coach. Prioritize this task (1-10).
            Title: {{.task.title}}
            Description: {{.task.description}}
            Current status: {{.task.status}}
            Return ONLY valid JSON: {"priority": 7, "reason": "..."}
          output_var: ai_result

        - name: update_task
          type: update
          entity: Task
          fields:
            priority: "{{.ai_result.priority}}"
            status: in_progress
            # due_date and other fields remain unchanged

        - name: emit_event
          type: emit_event
          event: TaskPrioritized
          payload:
            task_id: "{{.task.id}}"
            new_priority: "{{.ai_result.priority}}"
            reason: "{{.ai_result.reason}}"
            prioritized_by: "{{.user.id}}"

        - name: return_response
          type: return
          status: 200
          body: "{{.task}}"
```

**What VibeGuard generates from this:**
- A handler that follows the steps **exactly**
- Proper error handling at each step
- Transaction wrapping (if multiple DB writes)
- OpenTelemetry spans for each step
- Clean, readable Go code with clear comments

This removes almost all ambiguity while still allowing the LLM to write beautiful, production-grade Go.

#### Example 2: Process Refund (Complex Financial Flow)

```yaml
custom_endpoints:
  - path: /api/v1/orders/:id/refund
    method: POST
    description: "Process full or partial refund via Stripe + update DB + notify customer"
    request: RefundRequest
    response: RefundResponse
    logic:
      description: "Load order → Authorize → Call Stripe → Update order + create refund record → Notify customer → Emit event"
      steps:
        - name: load_order
          type: load
          entity: Order
          id_path: ":id"

        - name: check_eligible
          type: authorize
          condition: "order.status == 'paid' AND order.refunded_amount < order.total_amount"

        - name: calculate_refund
          type: custom
          description: "Calculate refund_amount = min(request.amount, order.total - order.refunded_amount)"

        - name: call_stripe
          type: external_call
          service: stripe
          action: refunds.create
          args: ["{{.order.stripe_payment_intent_id}}", "{{.refund_amount}}"]

        - name: update_order
          type: update
          entity: Order
          fields:
            refunded_amount: "{{.order.refunded_amount + .refund_amount}}"
            status: "{{ if eq .refund_amount .order.total_amount }}refunded{{ else }}partially_refunded{{ end }}"

        - name: create_refund_record
          type: create
          entity: Refund
          fields:
            order_id: "{{.order.id}}"
            amount: "{{.refund_amount}}"
            stripe_refund_id: "{{.stripe_result.id}}"
            reason: "{{.request.reason}}"

        - name: notify_customer
          type: notify
          channel: email
          template: "refund_processed"
          to: "{{.order.customer_email}}"
          args:
            amount: "{{.refund_amount}}"
            order_number: "{{.order.number}}"

        - name: emit_refund_event
          type: emit_event
          event: OrderRefunded
          payload:
            order_id: "{{.order.id}}"
            amount: "{{.refund_amount}}"

        - name: return_success
          type: return
          status: 200
          body: |
            {
              "refund_id": "{{.refund.id}}",
              "amount": {{.refund_amount}},
              "status": "succeeded"
            }
```

#### Example 3: Bulk Import Users (Transaction + Parallel + If)

```yaml
custom_endpoints:
  - path: /api/v1/admin/import-users
    method: POST
    description: "Import users from CSV with validation, duplicate check, welcome email, and progress tracking"
    request: ImportUsersRequest
    response: ImportResult
    logic:
      description: "Validate CSV → For each row: check duplicate → create user in transaction → send welcome email (parallel) → log result"
      steps:
        - name: validate_csv
          type: validate
          schema: ImportUsersRequest

        - name: process_rows
          type: transaction
          mode: sequential
          steps:
            - name: check_duplicate
              type: query
              query: "SELECT id FROM users WHERE email = $1 AND tenant_id = $2"
              args: ["{{.row.email}}", "{{.tenant_id}}"]
              output_var: existing_user

            - name: create_or_skip
              type: if
              condition: "existing_user == nil"
              then:
                - name: create_user
                  type: create
                  entity: User
                  fields:
                    email: "{{.row.email}}"
                    full_name: "{{.row.full_name}}"
                    role: member
                - name: send_welcome
                  type: notify
                  channel: email
                  template: "welcome_new_user"
                  to: "{{.row.email}}"
                  parallel: true
              else:
                - name: log_duplicate
                  type: log
                  level: warn
                  message: "Skipping duplicate email: {{.row.email}}"

        - name: return_summary
          type: return
          status: 200
          body: "{{.import_result}}"
```

---

## JSON Schema Validation

A formal JSON Schema is provided at `vibeguard_declaration_schema.json`.

You can validate any declaration with:

```bash
# Using ajv (Node.js)
npx ajv validate -s vibeguard_declaration_schema.json -d my-app.yaml

# Or in Go (using gojsonschema)
go run validate.go my-app.yaml
```

This enables IDE autocompletion, linting in CI, and early error detection.

---

**This is the foundation for making VibeGuard truly production-grade and "set it and forget it" reliable.**
- Integrate this declaration standard **directly into the master prompt** (add it as Step 0)?
- Create a sample `declaration.yaml` for a real app and generate the full Go code from it?
- Add support for `sqlc` or `ent` in the declaration (instead of raw pgx)?
- Turn this into a formal JSON Schema for IDE validation?

This is the key missing piece you identified — let's lock it in.