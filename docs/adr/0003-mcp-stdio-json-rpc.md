# ADR 0003 — MCP server transport: stdio JSON-RPC, no library

## Status

Accepted (branch 4-7). Will revisit if/when hosted multi-tenant MCP becomes a real requirement (see `docs/ROADMAP.md` § `4-7-saas`).

## Context

The MCP server is the user-facing surface for the LLM Layer pillar. Targets: Claude Desktop, Claude Code, Cursor, Zed — all run MCP servers as child processes per workspace, communicating over stdio with JSON-RPC 2.0 (the MCP spec's default transport).

Two transport options:

- **stdio JSON-RPC** — one process per workspace, no auth, no network surface
- **SSE** — long-lived HTTP connections, supports multiple concurrent clients, needs auth

Two implementation options:

- **A library** like `github.com/mark3labs/mcp-go`
- **Hand-rolled JSON-RPC handler** over `bufio.Scanner` (~150 LOC)

## Decision

Use **stdio transport** and a **hand-rolled JSON-RPC handler** for branch 4-7.

The decision is a function of scope: the MCP surface vibeguard exposes is small (3 tools, 2 resources, the `initialize`/`shutdown`/`ping` lifecycle) and the protocol is well-documented. A 150-line `cmd/vibeguard-mcp/main.go` handles it cleanly with zero external dependencies, which keeps the binary tiny and the build fast. We can switch to SSE in a later branch by mounting the same handler set on an HTTP server — the tool implementations don't depend on the transport.

When SSE arrives (multi-user hosted MCP), `mark3labs/mcp-go` becomes attractive because it provides resource subscriptions, sampling, and progress notifications out of the box. Today we don't need any of those.

## Consequences

**Good:**

- Zero dependencies in `cmd/vibeguard-mcp` beyond `internal/{parser,validate,render,lint}` and the standard library.
- ~10MB binary is mostly the platform SDK + render templates, not transport machinery.
- The protocol surface is visible — anyone reading `main.go` can see exactly what the server accepts and how it responds.
- The tools (`validate_declaration`, `generate_project`, `lint_project`) are pure functions over the substrate. Adding a new tool is one case in `tools_call.go`.

**Tradeoffs:**

- We will rewrite some boilerplate when SSE arrives. ~50 lines of marshaling and dispatch — not enough to justify a library upfront.
- No built-in sampling support. If a tool wants to delegate "what should I name this entity?" back to the caller's LLM (the MCP `sampling/createMessage` flow), we'll need to add a few lines of plumbing. Future, not now.

## Alternatives considered

- **`mark3labs/mcp-go` from day one.** Adds ~3MB binary + 2 transitive deps for behavior we don't yet use. Holding it back until we have SSE + sampling needs is the right tradeoff.
- **gRPC instead of JSON-RPC.** Not part of the MCP spec; clients wouldn't speak it.
- **Skip MCP, expose tools via plain HTTP.** Would re-invent everything Claude Desktop / Cursor / Zed already speak. The whole point of MCP is one protocol, many clients.

## Verification

- Smoke test: `printf '{"jsonrpc":"2.0","id":1,"method":"initialize"}\n{"jsonrpc":"2.0","id":2,"method":"tools/list"}\n' | vibeguard-mcp` returns the two expected JSON-RPC responses.
- End-to-end: configure Claude Desktop with the binary, restart, observe the three tools in the tool drawer, call `validate_declaration` with the sample yaml.

## See also

- MCP spec: <https://modelcontextprotocol.io>
- `cmd/vibeguard-mcp/main.go` — the implementation (~150 LOC)
- `docs/ROADMAP.md` § `4-7-saas` — when SSE + a library land
