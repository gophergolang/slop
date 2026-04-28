# vibeguard

> Point Claude at vibeguard. Describe your SaaS. Claude handles the backend — you own the frontend.

When you start a new SaaS project in Claude Code, add vibeguard to your MCP config and tell Claude what you're building. Claude drafts the declaration, validates it, generates a production-grade Go API + Postgres schema + Kubernetes deployment + OpenAPI spec + Next.js admin UI, and walks you through deploying it. You write business logic and your unique frontend. Everything else is generated.

- **Full backend from one file.** A `vibeguard.yaml` declaration produces a hardened Go service, SQL migrations with real row-level security, Kubernetes manifests, OpenAPI spec, and a Next.js admin UI — all in sync.
- **Security you can't accidentally skip.** Multi-tenant isolation, authentication, input validation, and audit logging are emitted by the generator. There's no code path that leaves them out.
- **You write only what's unique.** Business logic drops into typed Go stubs the generator places for you. Claude fills the rest in.

---

## Add to Claude in 30 seconds

**Claude Desktop** (`~/Library/Application Support/Claude/claude_desktop_config.json` on Mac, `%APPDATA%\Claude\claude_desktop_config.json` on Windows):

```jsonc
{
  "mcpServers": {
    "vibeguard": {
      "command": "/path/to/vibeguard-mcp"
    }
  }
}
```

**Claude Code** (`.claude/mcp.json` in your project, or global `~/.claude/mcp.json`):

```jsonc
{
  "mcpServers": {
    "vibeguard": {
      "command": "/path/to/vibeguard-mcp"
    }
  }
}
```

No environment variables needed — the binary carries everything it needs.

Build the binary: `make build` → `bin/vibeguard-mcp`. Or `go install github.com/vibeguard/vibeguard/cmd/vibeguard-mcp@latest`.

---

## Starting a new SaaS project

1. **Add vibeguard to your Claude MCP config** (above).

2. **Open a new project folder in Claude Code.**

3. **Tell Claude what you're building:**
   > *"I'm building a project management tool for small teams. Use vibeguard for the backend."*

   Claude loads the `new_saas_project` prompt from vibeguard and walks you through the workflow automatically.

4. **Claude handles the rest:**
   - Asks clarifying questions about your data model, CRUD needs, and custom endpoints
   - Drafts a `vibeguard.yaml` and shows it to you for review
   - Validates it (iterates until clean)
   - Generates the full backend into your project directory
   - Asks if you want deployment guidance

You never need to touch Go, SQL, or Kubernetes directly unless you want to.

---

## What you write vs what vibeguard generates

| You write | Vibeguard generates |
|---|---|
| `vibeguard.yaml` *(Claude drafts this)* | Go API handlers + repositories + `main.go` |
| `internal/*/nodes/*.go` *(your business logic)* | SQL migrations + row-level security policies |
| Your novel frontend | Kubernetes manifests + Dockerfile |
| | OpenAPI 3.1 spec |
| | Next.js admin UI (`web/`) |

The declaration is the source of truth. Re-run `vibeguard generate` any time you change it — generated files are overwritten, your business logic stubs are preserved.

---

## When you're ready to deploy

Claude guides you through whichever platform you choose. Here's the shape of each path.

**fly.io** — fastest for solo developers, deploys in minutes:
```bash
fly launch --no-deploy   # detects the generated Dockerfile
fly postgres create && fly postgres attach
fly secrets set DATABASE_URL=... NATS_URL=...
fly deploy
```

**Railway** — zero-config, good for small teams:
Connect your GitHub repo in the Railway dashboard, add a Postgres plugin (DATABASE_URL is injected automatically), set remaining env vars, and every push to `main` deploys.

**GCP Cloud Run** — scales to zero, good for variable traffic:
Build and push the Docker image to Artifact Registry, then apply the generated `k8s/` manifests to GKE or deploy the image directly to Cloud Run. Use Cloud SQL for Postgres and Secret Manager for credentials.

Ask Claude: *"Walk me through deploying this to fly.io"* — it will run the commands with you step by step.

---

## Reference

### What works today

| Subsystem | Status |
|---|---|
| Typed IR + YAML parser | ✅ |
| Semantic validator (9 rules) | ✅ |
| Multi-target generator (Go, SQL, K8s, OpenAPI, Next.js) | ✅ |
| Parent-child entity tree (nested URLs, FK cascades, frontend nav) | ✅ |
| Business-logic nodes (developer-owned Go funcs, secure wrapper generated) | ✅ |
| Master-prompt static analyzer (5 rules + SARIF) | ✅ |
| Real RLS enforcement (`pgxpool.BeforeAcquire` binds `app.tenant_id` per checkout) | ✅ |
| `platform/events/jetstream` durable adapter | ✅ |
| `platform/llm` gateway + Anthropic driver + sealed prompts | ✅ |
| MCP server with tools, resources, and `new_saas_project` prompt | ✅ |
| K8s operator CRD + reconciler scaffold | scaffold |
| Transactional outbox + drainer | scaffold |
| Observability (OTel) | scaffold |
| Eval framework | not started |

### 60-second CLI tour

```bash
# 1. Build
make build

# 2. See the typed IR for the reference declaration
./bin/vibeguard ir dump -f fixtures/sample_vibeguard.yaml

# 3. Validate against schema + semantic rules
./bin/vibeguard validate -f fixtures/sample_vibeguard.yaml

# 4. Generate a complete project (Go + SQL + K8s + OpenAPI + Next.js)
./bin/vibeguard generate -f fixtures/sample_vibeguard.yaml -o /tmp/app

# 5. Compile the generated Go service
cd /tmp/app && go mod tidy && go build ./...

# 6. Start the Next.js admin UI
cd /tmp/app/web && npm install && npm run dev

# 7. Lint against the master prompt
vibeguard lint ./...
```

### Project layout

```
cmd/
  vibeguard/             # multi-subcommand CLI
  vibeguard-mcp/         # MCP server (stdio JSON-RPC)
internal/
  ir/                    # typed declaration — the single source of truth
  parser/                # YAML → IR
  validate/              # 9 semantic rules
  static/                # embedded master prompt + declaration schema
  rules/code/            # master-prompt analyzers (VG001/VG002/VG005/VG006/VG009)
  render/
    golang/              # Go service emitter (handlers + repos + main + nodes)
    sql/                 # DDL + RLS migrations
    k8s/                 # hardened Deployment + Service + NetworkPolicy
    openapi/             # OpenAPI 3.1 spec
    nextjs/              # App-Router Next.js admin UI
  lint/                  # multichecker harness + SARIF formatter
platform/                # public SDK (separate Go module)
  db/                    # interface + postgres/ adapter (real RLS)
  events/                # interface + jetstream/ + natscore/ adapters + outbox
  workflow/              # interface + inmem/ + pgsaga/ adapters
  llm/                   # gateway + drivers/anthropic + sealed prompts
  cache/ secrets/ objectstore/ featureflag/ observability/
operator/                # K8s operator (separate Go module — kubebuilder layout)
fixtures/
  sample_project/        # canonical generated-output fixture
  sample_vibeguard.yaml  # reference declaration
docs/                    # ARCHITECTURE, QUICKSTART, ROADMAP, ADRs
```

### MCP tools and resources

| Name | Type | What it does |
|---|---|---|
| `new_saas_project` | Prompt | Loads the full SaaS workflow into Claude's context — draft, validate, generate, deploy |
| `validate_declaration` | Tool | Validates YAML text against schema + 9 semantic rules |
| `generate_project` | Tool | Generates Go + SQL + K8s + OpenAPI + Next.js from a declaration |
| `lint_project` | Tool | Runs master-prompt analyzers over a Go project directory |
| `vibeguard://prompts/master` | Resource | The master prompt (Guarded Vibe Coding discipline) |
| `vibeguard://schema/declaration.json` | Resource | JSON Schema for `vibeguard.yaml` |

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md), [`docs/QUICKSTART.md`](docs/QUICKSTART.md), and [`docs/ROADMAP.md`](docs/ROADMAP.md) for the full picture.

## License

MIT — see `LICENSE`.
