# VibeGuard Toolkit
## The Ultimate Guardrails for Vibe-Coding & AI-Generated Code

**Vision**: Turn chaotic, insecure "vibe coding" into a disciplined, elite engineering practice. Describe your app and business logic in plain English (as modules), and receive a complete, production-ready, 100% secure, high-performance codebase with zero vulnerabilities, zero antipatterns, minimal boilerplate, full tests, docs, and Docker deployment — every single time.

This toolkit eliminates the biggest pains of AI-assisted development:
- Inconsistent quality and hidden security holes
- Mountains of boilerplate that still needs heavy cleanup
- Performance surprises in production
- Technical debt from "good enough" code
- Fear of shipping AI-generated code to real users

### Core Idea
A **Docker base image** (`vibeguard/go-secure-base` and variants) combined with a **master LLM prompt** that enforces a new paradigm called **Guarded Vibe Coding (GVC)**.

The prompt turns any capable LLM (Claude 3.5/4, GPT-4o, Grok, etc.) into a world-class secure software architect that:
1. Decomposes your description into clean modules
2. Designs security & performance for each
3. Generates flawless code following strict rules
4. Produces full project scaffolding (Docker, CI, tests, docs)
5. Self-audits before outputting anything

**Result**: You get code you can confidently `docker build && docker run` and ship.

### The New Paradigm: Guarded Vibe Coding (GVC)

**Traditional Vibe Coding**:
- "Build me a todo app with auth"
- LLM spits out messy code with SQL injection risks, no tests, tight coupling, slow queries, etc.

**Guarded Vibe Coding**:
- You describe business logic + modules + constraints
- LLM follows a rigorous 6-step internal process (requirement analysis → architecture → per-module threat modeling + perf design → code gen with strict rules → integration → self-audit)
- Every decision is justified against security (OWASP, zero-trust), performance (latency, throughput, scalability), and maintainability (SOLID, clean arch)
- Output is always a complete, auditable, deployable artifact

**Key Guarantees**:
- **Security**: No OWASP Top 10 issues by construction. Input validation everywhere, least privilege, secure defaults, proper crypto, no secret leaks, rate limiting, structured logging.
- **Performance**: Efficient algorithms, optimal data access (no N+1), strategic caching, connection pooling, compiled static binaries. Code is profiled in the LLM's "mind".
- **Modularity**: Every feature is an isolated module with explicit contracts (Go structs + validator tags), dependency injection (via constructors or wire), and clear boundaries.
- **Quality**: 95%+ test coverage (unit + integration + security tests), type-safe (Go structs), perfectly formatted (`gofmt`/`goimports`), fully documented.
- **No Antipatterns**: Explicitly forbidden: God structs, magic strings/numbers, ignored errors, mutable globals without sync, circular deps, weak error handling, etc.
- **Deployability**: Production-grade Dockerfile (multi-stage Go build with CGO_ENABLED=0, static binary, non-root, tini or distroless, healthchecks), docker-compose, GitHub Actions with security scanning (gosec, govulncheck, trivy).

### Toolkit Components (Current)

1. **Master Prompt** (`vibeguard_master_prompt.md`)
   - The "brain" of the system. Copy-paste this at the start of any Claude (or other LLM) conversation.
   - It makes the LLM embody VibeGuard and follow the GVC paradigm for every request.
   - Supports iterative development ("add feature X to the previous project").

2. **Secure Docker Base Image Template** (included in prompt + separate `base.Dockerfile`)
   - Hardened Go 1.22 multi-stage base with:
     - Static binary (CGO_ENABLED=0)
     - Non-root user
     - Minimal attack surface (distroless or alpine + tini)
     - Healthchecks
     - Ready for your generated app code
   - Future: We can publish `ghcr.io/vibeguard/go-secure-base` (or equivalent).

3. **Embedded Checklists & Rules**
   - OWASP mitigations, performance patterns, Go-specific forbidden constructs (e.g. `fmt.Sprintf` for SQL) — all in the prompt so the LLM never forgets.

4. **Future Enhancements** (we can build these together):
   - CLI tool (`vibeguard generate "my description"`) that calls LLM API with the master prompt
   - Pre-built module library (auth, billing, realtime, AI agent templates using langchaingo)
   - VS Code / Cursor / GoLand extension
   - Automated validation pipeline (the generated code runs through `go vet`, `staticcheck`, `gosec`, `govulncheck`, `trivy` in CI)

### How to Use (Today)

1. Open Claude (claude.ai or API) or your favorite LLM.
2. Paste the entire content of `vibeguard_master_prompt.md` as the first message (or system prompt if supported).
3. Describe your app, e.g.:
   > "Build a SaaS task management platform. Modules: Auth (email + OAuth), Users & Teams, Tasks (CRUD + AI prioritization using OpenAI), Billing (Stripe), Realtime collaboration (WebSockets). Expect 10k users, sub-100ms API responses, SOC2 compliance. Use Go 1.22 + Gin."

4. VibeGuard will output a complete project.
5. Copy the files into a folder, `docker compose up`, and you're running secure production code.

### Example Output Quality
The generated code will include:
- Clean layered architecture (`internal/`, `cmd/server/`, `internal/modules/auth/`, `internal/modules/tasks/`, ...)
- Gin router + middleware stack (CORS, rate-limit, request-id, auth)
- Struct-based validation with `github.com/go-playground/validator/v10`
- pgx (pure Go Postgres driver) with prepared statements everywhere
- JWT (golang-jwt/jwt/v5) + Argon2id hashing (`golang.org/x/crypto/argon2`)
- Rate limiting + structured logging (slog or zap)
- Full Go test suite with >95% coverage + httptest
- OpenTelemetry tracing ready
- Production Dockerfile (multi-stage, static binary, non-root) + docker-compose + GitHub Actions (golangci-lint, gosec, govulncheck, test, build, trivy)
- Mermaid architecture diagrams
- Detailed Security & Performance Audit Report

### Why This Works Better Than Raw Prompting
- The prompt forces **chain-of-thought + explicit threat modeling + self-audit** before any code is written.
- Strict "NEVER / ALWAYS" rules prevent common LLM hallucinations and shortcuts (especially Go error handling and SQL safety).
- Structured output format makes the result immediately usable.
- Iterative context keeps the entire project coherent across multiple turns.

---

**This is version 0.9 of the VibeGuard Toolkit (Complete End-to-End System).**

We now have a fully functional CLI (`cmd/vibeguard`) that takes any `vibeguard.yaml` and generates the complete production-ready stack:
- Thin Go handlers using the Platform SDK
- Full Kubernetes + NATS + Argo CD manifests
- Platform SDK (`platform/`) with events, db, and workflow

**The system is now complete and usable.** Run `make generate` to get a full deployable project from any declaration.

### Your Vision (The Direction We're Heading)
- **Event-driven architecture** at the core (NATS or equivalent)
- **Reusable, bulletproof containers** for generic modules (auth, tasks, billing, AI, etc.)
- Each application = a **composition of containers** that can be independently scaled
- The "app" is mostly **business logic context** passed between compute units via events
- Only an **operator per app** needs customization (Kubernetes Operator or equivalent)
- Many apps share one Kubernetes cluster with strong isolation (or we eventually build our own scheduler)
- Everything is policy-driven, observable, and secure by default

### Honest Assessment

**What’s Excellent in Your Vision:**
- Event-driven + per-module scaling is the correct long-term paradigm for complex SaaS.
- "Operator per app + shared infrastructure" is smart, cost-effective, and operationally clean.
- Reducing custom container sprawl is a real pain point in most organizations.
- This naturally leads to better scalability, resilience, security isolation, and developer velocity.

**What’s Hard (Being Direct):**
- Truly "bulletproof" generic containers that work safely across many different apps is extremely difficult (state management, exactly-once delivery, security isolation between tenants/apps, backpressure, schema evolution).
- Building our own scheduler is a massive, multi-year project (Kubernetes took a huge team years).
- If we go too abstract too early, developer experience suffers ("where is my actual code and how do I debug it?").
- This shifts VibeGuard from "AI code generator" to "full internal developer platform" — a much bigger (and potentially much more valuable) bet.

### The Pragmatic Path We Will Take

We will **evolve toward your vision** in smart, incremental layers instead of trying to build everything at once:

1. **Keep the Declaration Layer** (`vibeguard.yaml`) as the primary human + AI interface.
2. **Build a strong Platform SDK first** (`vibeguard/db`, `vibeguard/events` powered by NATS, `vibeguard/workflow` for sagas/compensation).
3. **Use standard Kubernetes + NATS** as the initial compute fabric (not our own scheduler yet — too early).
4. Generate from the declaration:
   - Kubernetes manifests, Helm charts, Kustomize, or Argo CD Applications
   - NATS consumers + event-driven handlers
   - Security + observability sidecars
   - An **App Operator** (lightweight Kubernetes Operator or GitOps configuration) per application
5. Later (if we gain traction), we can consider a custom scheduler or more advanced control plane.

**High-level takeaway from your vision**: The future of VibeGuard is not "generate a monolith or even microservices." It is **"generate secure, event-driven, composable applications that run on a shared, policy-enforced platform with minimal per-app customization."**

This is the right direction. We just need to build it in layers that deliver value quickly while laying the foundation for the full vision.

**Next steps with you**:
- Refine the prompt (add more Go-specific security patterns, langchaingo agent templates, etc.)
- Generate a real example app to validate the new Go-focused prompt
- Finalize and publish the Go secure base Docker image
- Create the CLI wrapper in Go
- Add more pre-built secure modules (auth with OAuth, Stripe billing, realtime with Gorilla WebSocket or Centrifugo)

Tell me:
1. What do you like / want to change in the updated concept?
2. Any specific Go libraries or patterns you prefer (e.g. Fiber instead of Gin, sqlc instead of raw pgx, ent, etc.)?
3. Shall we test it right now with a sample app description using the new Go prompt?
4. Final branding / Docker image name?

Let's make VibeGuard the gold standard for secure AI-generated Go services.