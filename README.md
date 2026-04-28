# vibeguard

> Declaration-driven SDK for vibe coding & infrastructure. Slop in, production out.

`vibeguard` is the substrate that makes LLM-assisted application development *production-grade*. You write a small, security-first YAML declaration; vibeguard parses it into a typed IR, runs it through validators that enforce the master prompt's iron rules, and emits a compilable Go service backed by a hand-written platform SDK.

The result is a workflow where:

- The **declaration** is the source of truth, reviewed by humans and edited by LLMs.
- The **platform SDK** is hand-written, versioned, and the only thing generated code calls.
- The **operator** reconciles the declaration into actual cluster state.
- The **MCP server** exposes the entire workflow as tools any LLM (Claude Desktop, Claude Code, Cursor, Zed) can call.

Branch `4-7` is the foundational implementation of the four-pillar architecture (Compiler, Platform SDK, Operator, LLM Layer). See `docs/ROADMAP.md` for what's complete and what's next.

## What works today (branch 4-7)

| Subsystem | Status | Try it |
|---|---|---|
| Typed IR + YAML parser | ✅ | `vibeguard ir dump -f fixtures/sample_vibeguard.yaml` |
| Semantic validator (9 rules) | ✅ | `vibeguard validate -f fixtures/sample_vibeguard.yaml` |
| Multi-target generator (Go, SQL, K8s, OpenAPI, Next.js) | ✅ | `vibeguard generate -f fixtures/sample_vibeguard.yaml -o /tmp/app` |
| Parent-child entity tree (nested URLs, FK cascades) | ✅ | `parents: [Team]` on an entity; FKs + nested OpenAPI paths emerge automatically |
| Business-logic nodes (developer-owned Go funcs, framework wraps them) | ✅ | `node: tasks.Prioritize` on a custom_endpoint; stub at `internal/<mod>/nodes/<entity>_<func>.go` |
| Master-prompt static analyzer (5 rules + SARIF) | ✅ | `vibeguard lint ./...` |
| Real RLS enforcement in `platform/db/postgres` | ✅ | uses `pgxpool.BeforeAcquire` to bind `app.tenant_id` per checkout |
| `platform/events/jetstream` durable adapter | ✅ | publish/subscribe via JetStream with explicit ACK |
| `platform/llm` gateway + Anthropic driver + sealed prompts | ✅ | sha256-sealed prompts, structured-output validation, repair loop |
| MCP server (stdio) | ✅ | `vibeguard-mcp` exposes 3 tools + 2 resources |
| K8s operator CRD + scaffold | scaffold | types + reconciler placeholder; full reconciler in next branch |
| Transactional outbox + drainer | scaffold | schema in `platform/events/outbox.go`; drainer in next branch |
| Durable sagas (`workflow/pgsaga`) | scaffold | schema in `pgsaga/doc.go`; worker in next branch |
| Observability (OTel) | scaffold | `Init` is a no-op; OTLP wiring in next branch |
| Eval framework | not started | `eval/` lands in `4-7-eval` |
| TS/Python generator backends | not started | IR is shape-stable; lands when there's a user |

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for the four-pillar design and [`docs/ROADMAP.md`](docs/ROADMAP.md) for the next-branch sequencing.

## 60-second tour

```bash
# 1. Build
make build

# 2. See the typed IR for the reference declaration
./bin/vibeguard ir dump -f fixtures/sample_vibeguard.yaml

# 3. Validate against schema + 9 semantic rules
./bin/vibeguard validate -f fixtures/sample_vibeguard.yaml

# 4. Generate a complete Go service
./bin/vibeguard generate -f fixtures/sample_vibeguard.yaml -o /tmp/team-task-saas

# 5. The generated project compiles end-to-end
cd /tmp/team-task-saas
go mod tidy
go build ./...     # ✓ compiles
ls -R              # cmd/ internal/auth internal/billing internal/tasks internal/teams k8s/ migrations/ openapi.json

# 6. Lint the generated project (or your own) against the master prompt
/home/alex/code/biograph/slop/bin/vibeguard lint ./...
# 13 findings — VG006 markers warning per generated file

# 7. Try the MCP server (it speaks JSON-RPC over stdio)
echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | /home/alex/code/biograph/slop/bin/vibeguard-mcp
```

For a richer guided demo: see [`docs/QUICKSTART.md`](docs/QUICKSTART.md).

## Hooking up to Claude Desktop

```jsonc
// ~/Library/Application Support/Claude/claude_desktop_config.json
{
  "mcpServers": {
    "vibeguard": {
      "command": "/path/to/vibeguard-mcp",
      "env": {
        "VIBEGUARD_REPO_ROOT": "/path/to/your/vibeguard/repo"
      }
    }
  }
}
```

Restart Claude Desktop. You'll see vibeguard's three tools (`validate_declaration`, `generate_project`, `lint_project`) in the tool drawer, and the master prompt + JSON schema as MCP resources.

## Layout

```
cmd/
  vibeguard/             # multi-subcommand CLI
  vibeguard-mcp/         # MCP server (stdio JSON-RPC)
internal/
  ir/                    # typed declaration (single source of truth)
  parser/                # YAML → IR with polymorphic field decoders
  validate/              # 9 semantic rules
  rules/code/            # 4 master-prompt analyzers (VG001/VG002/VG005/VG006)
  render/                # multi-target render engine
    golang/              # Go service emitter (handlers + repos + main)
    sql/                 # DDL + RLS migrations
    k8s/                 # hardened Deployment + Service + NetworkPolicy
    openapi/             # OpenAPI 3.1 spec
  lint/                  # multichecker harness + SARIF formatter
  mcp/ (in cmd/)         # MCP server tool/resource implementations
platform/                # public SDK (separate Go module)
  db/                    # interface + postgres/ adapter (real RLS)
  events/                # interface + jetstream/ + natscore/ adapters + outbox
  workflow/              # interface + inmem/ + pgsaga/ adapters
  llm/                   # gateway + drivers/anthropic + sealed prompts
  cache/ secrets/ objectstore/ featureflag/ observability/
operator/                # K8s operator (separate Go module — kubebuilder layout)
fixtures/
  sample_project/        # the canonical generated-output fixture
  sample_vibeguard.yaml  # the reference declaration
docs/                    # ARCHITECTURE, QUICKSTART, ROADMAP, ADRs
```

## License

MIT — see `LICENSE`.
