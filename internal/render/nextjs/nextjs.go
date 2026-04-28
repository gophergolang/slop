// Package nextjs is the Next.js (App Router) backend for the vibeguard
// render engine.
//
// It emits a self-contained `web/` directory with:
//
//   - The minimum project skeleton (package.json, tsconfig.json,
//     next.config.mjs, tailwind config, root layout + landing page, global
//     stylesheet).
//   - A typed API client driven by the same OpenAPI surface the Go backend
//     emits (`web/lib/api.ts`).
//   - One `Entity` TypeScript interface per declared entity in
//     `web/lib/types.ts`.
//   - Admin pages walking the entity tree: list / detail / create routes
//     under `web/app/admin/...`. Nested URL paths derived from
//     `ir.EffectiveBasePath` (Team → Task becomes
//     `app/admin/teams/[teamId]/tasks/...` when no explicit api.base_path
//     overrides it).
//
// The static skeleton files are written with KeepIfExists so a developer
// can edit the design system, root layout, etc. without losing changes on
// re-generate. The per-entity admin pages are always overwritten — they
// are mechanical projections of the declaration.
package nextjs

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/vibeguard/vibeguard/internal/ir"
	"github.com/vibeguard/vibeguard/internal/render"
)

// Backend implements render.Backend for Next.js output.
type Backend struct{}

// New constructs a Next.js backend.
func New() *Backend { return &Backend{} }

// Name reports "nextjs".
func (Backend) Name() string { return "nextjs" }

// Plan emits the web/ tree.
func (b Backend) Plan(app *ir.Application) (render.FileSet, error) {
	var fs render.FileSet
	fs = append(fs, staticFiles(app)...)
	typesFile, err := emitTypes(app)
	if err != nil {
		return nil, err
	}
	fs = append(fs, typesFile)
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			pages, err := emitEntityPages(ent)
			if err != nil {
				return nil, fmt.Errorf("entity %s pages: %w", ent.Name, err)
			}
			fs = append(fs, pages...)
		}
	}
	return fs, nil
}

// ---- static files -------------------------------------------------------

func staticFiles(app *ir.Application) render.FileSet {
	appName := app.Metadata.Name
	if appName == "" {
		appName = "app"
	}
	const apiBaseEnv = "NEXT_PUBLIC_API_BASE"
	pkg := fmt.Sprintf(`{
  "name": %q,
  "private": true,
  "version": "0.1.0",
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start"
  },
  "dependencies": {
    "next": "14.2.5",
    "react": "18.3.1",
    "react-dom": "18.3.1"
  },
  "devDependencies": {
    "@types/node": "20.14.10",
    "@types/react": "18.3.3",
    "@types/react-dom": "18.3.0",
    "autoprefixer": "10.4.19",
    "postcss": "8.4.39",
    "tailwindcss": "3.4.6",
    "typescript": "5.5.3"
  }
}
`, appName+"-web")

	tsconfig := `{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": false,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "plugins": [{ "name": "next" }],
    "baseUrl": ".",
    "paths": { "@/*": ["./*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx"],
  "exclude": ["node_modules"]
}
`

	nextConfig := `/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
};
export default nextConfig;
`

	tailwind := `import type { Config } from "tailwindcss";
const config: Config = {
  content: ["./app/**/*.{ts,tsx}", "./lib/**/*.{ts,tsx}"],
  theme: { extend: {} },
  plugins: [],
};
export default config;
`

	postcss := `export default {
  plugins: { tailwindcss: {}, autoprefixer: {} },
};
`

	globals := `@tailwind base;
@tailwind components;
@tailwind utilities;

body { font-family: ui-sans-serif, system-ui, sans-serif; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 6px 12px; border-bottom: 1px solid #e5e7eb; }
input, select, textarea { border: 1px solid #d1d5db; padding: 6px 8px; border-radius: 4px; width: 100%; }
button { padding: 6px 12px; border-radius: 4px; background: #111827; color: white; }
button:disabled { opacity: 0.5; }
.field { margin-bottom: 12px; }
.label { font-size: 12px; color: #6b7280; margin-bottom: 4px; display: block; }
`

	rootLayout := fmt.Sprintf(`import "./globals.css";

export const metadata = { title: %q };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>
        <header style={{ padding: "12px 24px", borderBottom: "1px solid #e5e7eb" }}>
          <a href="/" style={{ fontWeight: 600 }}>{%q}</a>
        </header>
        <main style={{ padding: 24 }}>{children}</main>
      </body>
    </html>
  );
}
`, appName, appName)

	landing := fmt.Sprintf(`import Link from "next/link";

export default function Home() {
  return (
    <div>
      <h1 style={{ fontSize: 24, fontWeight: 600, marginBottom: 16 }}>{%q}</h1>
      <p style={{ marginBottom: 16 }}>
        Generated admin UI. Pages under <code>/admin</code> are auto-derived from
        the vibeguard declaration.
      </p>
      <ul>
%s
      </ul>
    </div>
  );
}
`, appName, landingLinks(app))

	apiClient := `// Generated by vibeguard. The OpenAPI spec the Go backend emits is the
// source of truth for paths and shapes; this thin wrapper keeps fetch
// boilerplate and credential propagation in one place.
const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(API_BASE + path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error("API " + res.status + ": " + text);
  }
  return res.status === 204 ? (undefined as T) : (await res.json() as T);
}

export const api = {
  get:  <T>(path: string)               => request<T>("GET",    path),
  list: <T>(path: string)               => request<T[]>("GET",  path),
  post: <T>(path: string, body: unknown) => request<T>("POST",   path, body),
  patch:<T>(path: string, body: unknown) => request<T>("PATCH",  path, body),
  del:  <T>(path: string)               => request<T>("DELETE", path),
};
`

	gitignore := `node_modules/
.next/
.env*.local
out/
`

	envExample := fmt.Sprintf("# %s lives at this base by default; override per environment.\nNEXT_PUBLIC_API_BASE=http://localhost:8080\n", apiBaseEnv)

	return render.FileSet{
		{Path: "web/package.json", Mode: 0o644, Content: []byte(pkg), KeepIfExists: true},
		{Path: "web/tsconfig.json", Mode: 0o644, Content: []byte(tsconfig), KeepIfExists: true},
		{Path: "web/next.config.mjs", Mode: 0o644, Content: []byte(nextConfig), KeepIfExists: true},
		{Path: "web/tailwind.config.ts", Mode: 0o644, Content: []byte(tailwind), KeepIfExists: true},
		{Path: "web/postcss.config.mjs", Mode: 0o644, Content: []byte(postcss), KeepIfExists: true},
		{Path: "web/.gitignore", Mode: 0o644, Content: []byte(gitignore), KeepIfExists: true},
		{Path: "web/.env.local.example", Mode: 0o644, Content: []byte(envExample), KeepIfExists: true},
		{Path: "web/app/globals.css", Mode: 0o644, Content: []byte(globals), KeepIfExists: true},
		{Path: "web/app/layout.tsx", Mode: 0o644, Content: []byte(rootLayout), KeepIfExists: true},
		{Path: "web/app/page.tsx", Mode: 0o644, Content: []byte(landing)},
		{Path: "web/lib/api.ts", Mode: 0o644, Content: []byte(apiClient)},
	}
}

func landingLinks(app *ir.Application) string {
	var b strings.Builder
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			route := nextjsRoute(ent)
			fmt.Fprintf(&b, "        <li><Link href=\"/admin/%s\">%s</Link></li>\n", route, ent.Name)
		}
	}
	return b.String()
}

// ---- types --------------------------------------------------------------

func emitTypes(app *ir.Application) (render.FileSpec, error) {
	var b strings.Builder
	b.WriteString("// Generated by vibeguard from the declaration. DO NOT EDIT.\n\n")
	for _, mod := range app.Modules {
		for _, ent := range mod.Entities {
			fmt.Fprintf(&b, "export interface %s {\n", ent.Name)
			for _, f := range ent.Fields {
				opt := ""
				if f.Nullable {
					opt = "?"
				}
				fmt.Fprintf(&b, "  %s%s: %s;\n", f.Name, opt, tsType(f))
			}
			if ent.SoftDelete {
				b.WriteString("  deleted_at?: string | null;\n")
			}
			b.WriteString("}\n\n")
		}
	}
	return render.FileSpec{Path: "web/lib/types.ts", Mode: 0o644, Content: []byte(b.String())}, nil
}

func tsType(f *ir.Field) string {
	switch f.Type {
	case ir.FieldString, ir.FieldText, ir.FieldUUID, ir.FieldEnum, ir.FieldTimestamp, ir.FieldDecimal:
		return "string"
	case ir.FieldInt, ir.FieldBigInt:
		return "number"
	case ir.FieldBool:
		return "boolean"
	case ir.FieldJSON:
		return "Record<string, unknown>"
	}
	return "string"
}

// ---- per-entity pages ---------------------------------------------------

// nextjsRoute converts ir.EffectiveBasePath to a Next.js App-Router folder
// path under web/app/admin/. e.g. "/api/v1/teams/:team_id/tasks" becomes
// "teams/[team_id]/tasks". A bare entity ("/api/v1/teams") becomes "teams".
func nextjsRoute(ent *ir.Entity) string {
	base := ir.EffectiveBasePath(ent)
	base = strings.TrimPrefix(base, "/")
	parts := strings.Split(base, "/")
	// drop "api/v1/" prefix when present
	if len(parts) >= 2 && parts[0] == "api" {
		parts = parts[2:]
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		if strings.HasPrefix(p, ":") {
			out = append(out, "["+p[1:]+"]")
			continue
		}
		out = append(out, p)
	}
	return strings.Join(out, "/")
}

// pageData feeds the per-page templates.
type pageData struct {
	Entity      *ir.Entity
	Route       string // e.g. "teams/[team_id]/tasks"
	APIPath     string // e.g. "/api/v1/teams/:team_id/tasks"
	APIPathTS   string // same as APIPath but with ${params.team_id} interpolations
	APIItemPath string
	ListFields  []*ir.Field
	UpdateFields []*ir.Field
}

func emitEntityPages(ent *ir.Entity) (render.FileSet, error) {
	route := nextjsRoute(ent)
	if route == "" {
		return nil, nil
	}
	apiPath := ir.EffectiveBasePath(ent)
	data := pageData{
		Entity:       ent,
		Route:        route,
		APIPath:      apiPath,
		APIPathTS:    toTSPath(apiPath),
		APIItemPath:  toTSPath(apiPath) + "/${params.id}",
		ListFields:   listFields(ent),
		UpdateFields: ent.CRUD.UpdateFields,
	}

	var fs render.FileSet
	dir := "web/app/admin/" + route

	if ent.CRUD.List {
		f, err := renderPage(listTmpl, data, dir+"/page.tsx")
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	} else {
		// emit a placeholder so the route is reachable but says list disabled
		f, err := renderPage(listDisabledTmpl, data, dir+"/page.tsx")
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	if ent.CRUD.Read {
		f, err := renderPage(detailTmpl, data, dir+"/[id]/page.tsx")
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}

	if ent.CRUD.Create {
		f, err := renderPage(createTmpl, data, dir+"/new/page.tsx")
		if err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}
	return fs, nil
}

// listFields chooses up to 5 fields to render in the list view: primary
// key first (if any), then non-nullable simple-typed fields, skipping
// password-like names.
func listFields(ent *ir.Entity) []*ir.Field {
	var out []*ir.Field
	if ent.PrimaryKey != nil {
		out = append(out, ent.PrimaryKey)
	}
	for _, f := range ent.Fields {
		if len(out) >= 5 {
			break
		}
		if f.Primary {
			continue
		}
		if strings.Contains(strings.ToLower(f.Name), "password") {
			continue
		}
		switch f.Type {
		case ir.FieldString, ir.FieldEnum, ir.FieldUUID, ir.FieldTimestamp,
			ir.FieldInt, ir.FieldBigInt, ir.FieldBool:
			out = append(out, f)
		}
	}
	return out
}

func toTSPath(p string) string {
	parts := strings.Split(p, "/")
	for i, seg := range parts {
		if strings.HasPrefix(seg, ":") {
			parts[i] = "${params." + seg[1:] + "}"
		}
	}
	return strings.Join(parts, "/")
}

// renderPage uses non-default delimiters ([[ ]]) so JSX's `{{ }}` style
// objects don't collide with Go's template syntax.
func renderPage(tmpl string, data pageData, path string) (render.FileSpec, error) {
	t := template.Must(template.New(path).Delims("[[", "]]").Funcs(template.FuncMap{
		"tsType": tsType,
	}).Parse(tmpl))
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return render.FileSpec{}, err
	}
	return render.FileSpec{
		Path:    path,
		Mode:    0o644,
		Content: buf.Bytes(),
	}, nil
}

// ---- page templates -----------------------------------------------------
//
// The templates intentionally use a single client component per page and
// keep markup minimal. The output is meant to be a working starting point
// the developer styles; vibeguard does not ship a design system.

const listTmpl = `"use client";

// Generated by vibeguard. DO NOT EDIT.
// Source: declaration entity [[.Entity.Name]].

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import type { [[.Entity.Name]] } from "@/lib/types";

export default function [[.Entity.Name]]List() {
  const params = useParams<Record<string, string>>();
  const [items, setItems] = useState<[[.Entity.Name]][] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.list<[[.Entity.Name]]>(` + "`[[.APIPathTS]]`" + `).then(setItems).catch((e) => setError(String(e)));
  }, []);

  if (error) return <div style={{ color: "crimson" }}>{error}</div>;
  if (!items) return <div>Loading…</div>;

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 600, marginBottom: 12 }}>[[.Entity.Name]]</h1>
[[- if .Entity.CRUD.Create ]]
      <Link href={` + "`/admin/[[.Route]]/new`" + `}>+ New [[.Entity.Name]]</Link>
[[- end ]]
      <table style={{ marginTop: 16 }}>
        <thead>
          <tr>
[[- range .ListFields ]]
            <th>[[.Name]]</th>
[[- end ]]
            <th></th>
          </tr>
        </thead>
        <tbody>
          {items.map((it) => (
            <tr key={String(it.[[.Entity.PrimaryKey.Name]])}>
[[- range .ListFields ]]
              <td>{String(it.[[.Name]] ?? "")}</td>
[[- end ]]
              <td>
[[- if .Entity.CRUD.Read ]]
                <Link href={` + "`/admin/[[.Route]]/${it.[[.Entity.PrimaryKey.Name]]}`" + `}>view</Link>
[[- end ]]
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
`

const listDisabledTmpl = `// Generated by vibeguard. DO NOT EDIT.
// Source: declaration entity [[.Entity.Name]] (crud.list = false).

import Link from "next/link";

export default function [[.Entity.Name]]List() {
  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 600, marginBottom: 12 }}>[[.Entity.Name]]</h1>
      <p>List view is disabled in the declaration.</p>
[[- if .Entity.CRUD.Create ]]
      <Link href={` + "`/admin/[[.Route]]/new`" + `}>+ New [[.Entity.Name]]</Link>
[[- end ]]
    </div>
  );
}
`

const detailTmpl = `"use client";

// Generated by vibeguard. DO NOT EDIT.
// Source: declaration entity [[.Entity.Name]].

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { api } from "@/lib/api";
import type { [[.Entity.Name]] } from "@/lib/types";

export default function [[.Entity.Name]]Detail() {
  const params = useParams<Record<string, string>>();
  const [item, setItem] = useState<[[.Entity.Name]] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    api.get<[[.Entity.Name]]>(` + "`[[.APIItemPath]]`" + `).then(setItem).catch((e) => setError(String(e)));
  }, []);

  if (error) return <div style={{ color: "crimson" }}>{error}</div>;
  if (!item) return <div>Loading…</div>;

[[ if .UpdateFields ]]
  async function save(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!item) return;
    setPending(true);
    const form = new FormData(e.currentTarget);
    const patch: Record<string, unknown> = {};
[[- range .UpdateFields ]]
    {
      const v = form.get("[[.Name]]");
      if (v !== null && v !== "") patch["[[.Name]]"] = v;
    }
[[- end ]]
    try {
      const updated = await api.patch<[[.Entity.Name]]>(` + "`[[.APIItemPath]]`" + `, patch);
      setItem(updated);
    } catch (e) {
      setError(String(e));
    } finally {
      setPending(false);
    }
  }
[[ end ]]

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 600, marginBottom: 12 }}>[[.Entity.Name]]</h1>
      <pre style={{ background: "#f9fafb", padding: 12, borderRadius: 6 }}>{JSON.stringify(item, null, 2)}</pre>
[[ if .UpdateFields ]]
      <form onSubmit={save} style={{ marginTop: 24, maxWidth: 480 }}>
        <h2 style={{ fontSize: 16, fontWeight: 600, marginBottom: 12 }}>Update</h2>
[[- range .UpdateFields ]]
        <div className="field">
          <label className="label" htmlFor="[[.Name]]">[[.Name]]</label>
          <input id="[[.Name]]" name="[[.Name]]" defaultValue={String(item.[[.Name]] ?? "")} />
        </div>
[[- end ]]
        <button type="submit" disabled={pending}>Save</button>
      </form>
[[ end ]]
    </div>
  );
}
`

const createTmpl = `"use client";

// Generated by vibeguard. DO NOT EDIT.
// Source: declaration entity [[.Entity.Name]].

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import type { [[.Entity.Name]] } from "@/lib/types";

export default function Create[[.Entity.Name]]() {
  const params = useParams<Record<string, string>>();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setPending(true);
    const form = new FormData(e.currentTarget);
    const body: Record<string, unknown> = {};
[[- range .Entity.Fields ]]
[[- if not .Primary ]]
    {
      const v = form.get("[[.Name]]");
      if (v !== null && v !== "") body["[[.Name]]"] = v;
    }
[[- end ]]
[[- end ]]
    try {
      const created = await api.post<[[.Entity.Name]]>(` + "`[[.APIPathTS]]`" + `, body);
      router.push(` + "`/admin/[[.Route]]/${(created as any).[[.Entity.PrimaryKey.Name]]}`" + `);
    } catch (e) {
      setError(String(e));
    } finally {
      setPending(false);
    }
  }

  return (
    <div>
      <h1 style={{ fontSize: 22, fontWeight: 600, marginBottom: 12 }}>New [[.Entity.Name]]</h1>
      {error && <div style={{ color: "crimson", marginBottom: 12 }}>{error}</div>}
      <form onSubmit={submit} style={{ maxWidth: 480 }}>
[[- range .Entity.Fields ]]
[[- if not .Primary ]]
        <div className="field">
          <label className="label" htmlFor="[[.Name]]">[[.Name]]</label>
          <input id="[[.Name]]" name="[[.Name]]" />
        </div>
[[- end ]]
[[- end ]]
        <button type="submit" disabled={pending}>Create</button>
      </form>
    </div>
  );
}
`
