# vibeguard — plain English for investors

## What it is

You write a short, structured description of an application — its data, its rules, its endpoints. Vibeguard reads that description and generates a complete, runnable system: a hardened Go API, a Postgres schema with the right migrations, a Next.js admin UI for your users to manage data, Kubernetes deployment files, and an OpenAPI specification.

The only code a developer writes by hand is **business logic** — and even that drops into a stub the generator places, already wrapped in the security boilerplate you can't accidentally bypass: tenant isolation, authentication, input validation, database transactions, audit logging, observability.

In one sentence: **you describe the shape of the application, vibeguard ships the building.**

## Why this matters

Modern software teams spend the majority of their time on the same plumbing on every project. Authentication. Multi-tenant data isolation. CRUD APIs. Admin dashboards. Database migrations. Deployment manifests. None of it is the actual product, but all of it has to be right — and getting it right repeatedly across teams is where security bugs, data leaks, and shipping delays come from.

At the same time, AI coding tools have made writing the *first* draft of all this plumbing nearly free, and made the *quality* of that plumbing wildly inconsistent. Teams now ship faster but with less confidence, and the code reviews that used to catch problems can't keep up.

Vibeguard's bet is that the right answer to both problems is the same one: stop hand-writing the plumbing. Generate it from a single source of truth, with security baked into the generator itself, and let humans (and AI) focus on the part of the application that's actually unique — the business logic.

## How it works

Three things happen when you give vibeguard a declaration:

1. **The data model becomes a tree.** You describe entities and how they nest — a Team has Tasks, a Task has Comments. Vibeguard derives the database schema, foreign keys, cascade rules, URL structure (`/teams/:id/tasks/:id/comments/:id`), and the navigation of the admin UI from that tree. One source of truth, many artifacts in sync.

2. **Security is generated, not requested.** Multi-tenant isolation is enforced by the database itself (Postgres row-level security, bound to the request's tenant before every query — not a check the developer might forget). Authentication wraps every endpoint. Input is validated against typed schemas. SQL is parameterized by construction. There is no path to production where these protections are absent — the generator simply doesn't emit code without them.

3. **Business logic plugs in cleanly.** When a developer needs to write custom logic — "prioritize this task using AI", "send a Slack notification when an invoice is paid" — vibeguard generates a stub function with the right signature, already inside the secure wrapper. The developer fills in the body. The framework handles everything around it: parsing the request, checking permissions, opening transactions, recording the trace, mapping errors to responses.

The same generator also produces the **Next.js admin UI** for managing the data — list pages, detail pages, forms — derived from the same tree. And it ships an **MCP integration** so AI coding assistants (Claude, Cursor, others) can edit the declaration directly with the guardrails on.

## What's real today

- The compiler pipeline (declaration → typed model → generated Go + SQL + Kubernetes + OpenAPI + Next.js) works end-to-end. Generated projects compile and run.
- The **parent-child entity tree** is in: a declaration like `parents: [Team]` on a `Task` entity now derives nested URLs, foreign keys with cascade, and frontend route nesting from one source.
- **Business-logic nodes** are in: declare `node: tasks.Prioritize` on a custom endpoint and the generator emits the secure wrapper plus a Go stub at `internal/<module>/nodes/<entity>_<func>.go`. The stub is preserved across re-generates; the wrapper is overwritten and lint-enforced.
- The **Next.js frontend generator** is in: a `web/` folder lands alongside the Go service with list/detail/create admin pages walking the entity tree, typed against the same TypeScript interfaces vibeguard derives from the declaration.
- Multi-tenant isolation through Postgres row-level security is implemented for real, not stubbed.
- Durable event processing via NATS JetStream is in place.
- The MCP server is live; AI assistants can talk to vibeguard today.
- A static analyzer flags five classes of dangerous patterns in generated or hand-edited code, including a new rule that prevents node bodies from bypassing the framework's secure wrapper.

## What's next

- Kubernetes operator that deploys the generated apps automatically.
- Event durability and observability hardening (the JetStream drainer worker, OTLP exporter wiring).
- A richer Next.js scaffold: shadcn/ui defaults, an auth shell wired to the Go service's `/auth/*` endpoints, parent-aware breadcrumbs.
- An evaluation framework that benchmarks how well different LLMs produce vibeguard declarations.

## The honest picture

**What's real and differentiated:** The architecture is sound and the hard parts are already working — real RLS, sealed prompts, durable events, MCP. Most "AI coding" tools are autocomplete on top of a blank file. Vibeguard takes the opposite stance: the boring, dangerous parts are *not* a blank file, they are pre-built, audited, and impossible to leave out. That position is rare and defensible.

**What's early:** This is one developer's work. There are no paying customers yet. The parent-tree, business-logic-node, and Next.js generator pillars are working end-to-end against the canonical fixture, but they're new — they have not yet been exercised on a real production application or a non-trivial third-party declaration. The Kubernetes operator and event-durability worker are scaffolded; the saas-grade hosted MCP is not started.

**Market risk:** Developer tooling is hard to monetize. The plausible commercial paths are (a) a hosted version that AI assistants connect to, with usage-based pricing and team controls; (b) enterprise licensing for organizations that need to prove to auditors what their AI-assisted code is doing. Neither is tested yet.

**Technical risk:** The bet is that 80% of common SaaS applications fit the declaration model well enough that the remaining 20% can be handled cleanly via business-logic nodes. If that ratio is wrong — if real applications constantly need to escape the model — the value proposition weakens. The next two branches (parent-tree, Next.js) are the test of that bet.

## Summary

Vibeguard is a bet on a specific shape of the future: applications described once, generated everywhere, with the security and integration plumbing handled by a substrate the developer doesn't need to think about. The foundations are credible. The upcoming branches turn the foundations into the full pitch — a single declaration producing a database, a backend, a frontend, and a deployment, with the developer writing only the business logic that's actually theirs.
