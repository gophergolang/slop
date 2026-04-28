# VibeGuard

**Secure, Event-Driven, Composable Applications — Generated from Declaration**

VibeGuard turns natural language descriptions into production-ready, secure, event-driven Go applications that run on Kubernetes with NATS — while keeping humans in full control of the important parts.

---

## The Problem

Most AI code generation tools fail in production because they generate too much code that humans don't own. This leads to:

- Brittle long-term maintainability
- Painful code reviews
- Fear of making changes
- Infrastructure lock-in
- Distrust from senior engineers

**VibeGuard solves this with a fundamentally different approach.**

---

## The Architecture (3 Layers)

```
┌─────────────────────────────────────────────────────────────┐
│                    DECLARATION LAYER                        │
│  vibeguard.yaml — Human + AI contract                       │
│  • CRUD whitelists (security-first)                         │
│  • Logic DSL (validate → load → saga → emit → return)       │
│  • Multi-tenancy, RLS, compliance, events                   │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                   PLATFORM SDK (The Real Moat)              │
│  github.com/vibeguard/platform                              │
│  • events   — NATS-powered, tenant-aware, reliable          │
│  • db       — Abstract, tenant-aware, RLS-enforcing         │
│  • workflow — Saga + compensation runner                    │
│  • policy   — Runtime security & compliance enforcement     │
│                                                             │
│  Hand-written • Versioned • Tested • Pluggable              │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│                  GENERATED THIN LAYER                       │
│  Only the minimal code that benefits from the declaration   │
│  • Thin handlers (HTTP + validation + SDK calls)            │
│  • Kubernetes manifests + NATS consumers                    │
│  • Argo CD GitOps configuration                             │
│                                                             │
│  Small • Readable • Human-owned • Easy to modify            │
└─────────────────────────────────────────────────────────────┘
```

---

## Quick Start

```bash
# 1. Clone or download VibeGuard
git clone https://github.com/vibeguard/vibeguard
cd vibeguard

# 2. Generate a full project from a declaration
make generate

# 3. Run it locally
cd my-generated-app
go mod tidy
go run ./cmd/server

# 4. Deploy to Kubernetes
kubectl apply -k k8s/
```

Or use the CLI directly:

```bash
go run ./cmd/vibeguard -f my-app.yaml
```

---

## Key Components

### 1. Declaration DSL (v0.5)

A clean, powerful, future-proofed step-based language:

```yaml
custom_endpoints:
  - path: /api/v1/tasks/:id/prioritize
    method: POST
    logic:
      steps:
        - name: load_task
          type: load
          entity: Task
        - name: call_ai
          type: external_call
          service: openai
        - name: update_priority
          type: update
          entity: Task
        - name: emit_event
          type: emit
          event: TaskPrioritized
          topic: tasks.prioritized
```

**Supported step types**: `validate`, `load`, `authorize`, `external_call`, `update/create/delete`, `emit`, `consume`, `retry`, `parallel`, `saga`, `compensate`, `policy`, `if`, `transaction`, and more.

### 2. Platform SDK (`platform/`)

The real product — hand-written, production-grade libraries:

- **`events`** — NATS-powered publisher/subscriber with standard `Event` envelope
- **`db`** — Tenant-aware Postgres client with built-in RLS
- **`workflow`** — Saga + automatic compensation runner

All generated code uses these interfaces. You can swap NATS for Kafka or Postgres for CockroachDB without changing generated code.

### 3. Generator + CLI

`vibeguard generate -f declaration.yaml` produces:

- Thin, maintainable Go handlers
- Full Kubernetes manifests (Deployment, NetworkPolicy, etc.)
- NATS consumer configurations
- Argo CD Application for GitOps
- Production-hardened Dockerfile

### 4. Kubernetes + GitOps

Every generated project includes:

- Production-grade Deployment (non-root, read-only FS, probes, resource limits)
- NetworkPolicy for security isolation
- NATS consumers derived from your declaration
- Argo CD Application for GitOps deployment

---

## Philosophy

> Use AI aggressively where it excels (repetitive, rule-based, security-critical scaffolding) while keeping humans firmly in control of everything that determines long-term success (complex business logic, infrastructure, operational control, and code ownership).

This is the only approach that produces systems real engineering organizations will trust and maintain.

---

## Roadmap

| Phase | Status | Focus |
|-------|--------|-------|
| 1     | ✅     | Declaration DSL + Platform SDK foundation |
| 2     | ✅     | Kubernetes + GitOps generation |
| 3     | ✅     | Working CLI (`vibeguard generate`) |
| 4     | 🔜     | Real Kubernetes Operator (CRD + controller) |
| 5     | 🔜     | Advanced workflow engine + visual editor |
| 6     | 🔜     | Multi-cluster + self-service platform |

---

## Example Output

After running `make generate`, you get a complete project ready to deploy:

```bash
my-app/
├── cmd/server/main.go           # Wires Platform SDK
├── internal/tasks/
│   └── handler.go               # Thin handler using events + db
├── k8s/
│   ├── deployment.yaml          # Production-hardened
│   ├── nats-consumers.yaml      # Auto-generated from declaration
│   └── argocd-application.yaml  # GitOps
├── platform/                    # events + db + workflow
└── Dockerfile
```

---

## Why This Won't Get Laughed At

- **Humans own the hard parts** (Platform SDK)
- **Generated code stays thin** and easy to review
- **Infrastructure is pluggable** (swap DB/event systems easily)
- **Security is enforced at runtime** (not just at generation time)
- **Clear evolution path** to full internal developer platform

---

## Contributing

VibeGuard is built in layers. The most valuable contributions are:

1. Improving the Platform SDK (reliability, new backends)
2. Enhancing the Declaration DSL (new step types, better ergonomics)
3. Making the generator smarter (better thin code patterns)
4. Building the real Kubernetes Operator

---

## License

MIT — Use it, break it, improve it.

---

**VibeGuard** — Secure by declaration. Reliable by design. Human-owned at the core.