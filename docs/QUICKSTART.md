# vibeguard quickstart

This walks through a complete end-to-end vibeguard session — declaration → validate → generate → compile → lint → MCP. About 5 minutes.

## Prerequisites

- Go 1.25+
- (Optional) Postgres + NATS JetStream if you want to actually run the generated app
- (Optional) Claude Desktop / Claude Code / Cursor for the MCP demo

## Build the binaries

```bash
cd /path/to/slop
make build
ls -la bin/
# vibeguard       — the multi-subcommand CLI
# vibeguard-mcp   — the MCP server
```

If you don't have `make`:

```bash
go build -o bin/vibeguard ./cmd/vibeguard
go build -o bin/vibeguard-mcp ./cmd/vibeguard-mcp
```

## Step 1 — read the typed IR

The compiler turns `vibeguard.yaml` into a typed Go object. Inspect what it sees:

```bash
./bin/vibeguard ir dump -f fixtures/sample_vibeguard.yaml
```

You should see:

```
Application: team-task-saas (apiVersion=vibeguard.dev/v1, version=1.0.0)
  compliance: [SOC2 GDPR]
  multi_tenancy: enabled (isolation=row, tenant_id_field=tenant_id)

module auth (type=identity)
  entity User (table=users, sensitivity=confidential)
    crud: [create read update[[full_name email]]]
    fields (6): id:uuid, tenant_id:uuid, email:string, password_hash:string, full_name:string, role:enum
...
module tasks (type=business)
  entity Task (table=tasks, sensitivity=internal)
    crud: [create read list update[[title description status priority due_date assignee_id]] soft_delete]
    custom: POST /api/v1/tasks/:id/prioritize — 7 steps
      - ir.ValidateStep validate_input
      - ir.LoadStep load_task
      - ir.AuthorizeStep check_permissions
      - ir.ExternalCallStep call_ai
      - ir.UpdateStep update_task
      - ir.EmitEventStep emit_event
      - ir.ReturnStep return_response
```

This is what every backend (Go, SQL, K8s, OpenAPI) consumes.

## Step 2 — validate

```bash
./bin/vibeguard validate -f fixtures/sample_vibeguard.yaml
```

Output:

```
[WARN] VG-VAL-007  custom endpoint declares no logic.steps — handler will be empty
           modules[auth].entities[User].api.custom_endpoints[/auth/register]
...
0 errors, 3 warnings
```

3 warnings on the auth endpoints (which don't yet declare logic.steps) are real. The rest of the declaration is clean.

To see what bad declarations look like, copy the file, break something, and re-run:

```bash
cp fixtures/sample_vibeguard.yaml /tmp/bad.yaml
sed -i 's/full_name, email/full_name, email, missing_field/' /tmp/bad.yaml
./bin/vibeguard validate -f /tmp/bad.yaml
# [ERR ] VG-VAL-004  update lists field "missing_field" which is not declared
```

## Step 3 — generate

```bash
./bin/vibeguard generate -f fixtures/sample_vibeguard.yaml -o /tmp/team-task-saas
```

Output:

```
✓ generated 18 files into /tmp/team-task-saas/
    go       14 files (27465 bytes)
    sql      2 files (1943 bytes)
    k8s      1 files (2207 bytes)
    openapi  1 files (5339 bytes)
```

Inspect:

```bash
find /tmp/team-task-saas -type f | sort
# /tmp/team-task-saas/cmd/server/main.go
# /tmp/team-task-saas/go.mod
# /tmp/team-task-saas/internal/auth/user.go
# /tmp/team-task-saas/internal/auth/user_handler.go
# /tmp/team-task-saas/internal/auth/user_repository.go
# ... (one set per entity)
# /tmp/team-task-saas/k8s/deployment.yaml
# /tmp/team-task-saas/migrations/0001_init.up.sql
# /tmp/team-task-saas/migrations/0001_init.down.sql
# /tmp/team-task-saas/openapi.json
```

Look at `internal/tasks/task_repository.go`. Note:

- The Update path enforces the declaration's whitelist (only `title`, `description`, `status`, `priority`, `due_date`, `assignee_id` can be set).
- There is no Delete method — the declaration says `delete: false`.
- All queries pass through `db.DB`, which means `app.tenant_id` is bound on every checkout (RLS fires automatically).

## Step 4 — compile the generated project

```bash
cd /tmp/team-task-saas
# Point at your local platform module:
echo "
replace github.com/vibeguard/platform => /path/to/slop/platform
" >> go.mod
go mod tidy
go build ./...
# (no output = success)
```

That's a real Go binary. The thin handlers wire the platform SDK; the platform SDK is hand-written, versioned, and the only place infra-specific code lives.

## Step 5 — lint against the master prompt

```bash
cd /tmp/team-task-saas
/path/to/slop/bin/vibeguard lint ./...
```

You'll see VG006 warnings on every generated file ("file is generator-marked; verify edits are within markers"). The other rules (VG001 sprintf-SQL, VG002 unhandled error, VG005 context.Background in handler) don't fire because the generated code is rule-clean by construction. Try editing a handler to use `fmt.Sprintf` for a SQL string and the linter catches it.

## Step 6 — try the MCP server

```bash
cd /path/to/slop
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./bin/vibeguard-mcp
```

You'll see the JSON-RPC handshake plus the three exposed tools.

To wire it into Claude Desktop, edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```jsonc
{
  "mcpServers": {
    "vibeguard": {
      "command": "/path/to/slop/bin/vibeguard-mcp",
      "env": { "VIBEGUARD_REPO_ROOT": "/path/to/slop" }
    }
  }
}
```

Restart. In a Claude conversation:

> Use the vibeguard MCP. Read `vibeguard://prompts/master`, then read my project's `vibeguard.yaml`, then call `validate_declaration`. If it's clean, propose two new entities for an audit log feature.

Claude will pull the master prompt, validate against the schema, and propose a diff — exactly the loop the master prompt describes, but now driven by tools instead of free-form text.

## What's next

See `docs/ROADMAP.md` for what lands in the next branch — including the operator reconciler, durable sagas, the outbox drainer, and the eval framework.
