package golang

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"

	"github.com/vibeguard/vibeguard/internal/ir"
	"github.com/vibeguard/vibeguard/internal/render"
)

// emitNodes returns the file set for one entity's node-backed custom endpoints:
//
//   - <module>/nodes/nodes.go (KeepIfExists, once per module — the Nodes
//     receiver type and the Deps struct)
//   - <module>/nodes/<entity>_<func>.go (KeepIfExists, one per endpoint —
//     the developer-owned stub: request/response types + method body)
//   - <module>/<entity>_node_handler.go (overwritten — the secure wrapper:
//     parse, validate, auth, role check, tenant context, dispatch to node)
//
// Returns nil when ent has no node-backed endpoints.
func (b *Backend) emitNodes(app *ir.Application, mod ir.Module, ent *ir.Entity, pkg, dir string) (render.FileSet, error) {
	endpoints := nodeEndpoints(ent)
	if len(endpoints) == 0 {
		return nil, nil
	}
	var fs render.FileSet

	pkgFile, err := b.emitNodesPackage(mod, pkg, dir)
	if err != nil {
		return nil, err
	}
	fs = append(fs, pkgFile)

	for _, ep := range endpoints {
		stub, err := b.emitNodeStub(ent, ep, pkg, dir)
		if err != nil {
			return nil, err
		}
		fs = append(fs, stub)
	}

	wrapper, err := b.emitNodeHandler(app, mod, ent, endpoints, pkg, dir)
	if err != nil {
		return nil, err
	}
	fs = append(fs, wrapper)
	return fs, nil
}

// nodeEndpoints returns the subset of ent's custom endpoints that have a
// non-empty Node reference, in declaration order.
func nodeEndpoints(ent *ir.Entity) []ir.CustomEndpoint {
	var out []ir.CustomEndpoint
	for _, ep := range ent.API.CustomEndpoints {
		if strings.TrimSpace(ep.Node) != "" {
			out = append(out, ep)
		}
	}
	return out
}

// nodeFunc returns the method name to hang on *Nodes for the given endpoint.
// It accepts both bare names ("Prioritize") and dotted package-qualified
// references ("tasks.Prioritize") — only the suffix after the last dot is
// significant.
func nodeFunc(ep ir.CustomEndpoint) string {
	n := strings.TrimSpace(ep.Node)
	if i := strings.LastIndex(n, "."); i >= 0 {
		n = n[i+1:]
	}
	return n
}

// nodeStubFile returns the relative path of the developer-owned stub file for
// the given (entity, endpoint).
func nodeStubFile(ent *ir.Entity, ep ir.CustomEndpoint) string {
	return strings.ToLower(snake(ent.Name)) + "_" + snake(nodeFunc(ep)) + ".go"
}

func (b *Backend) emitNodesPackage(mod ir.Module, pkg, dir string) (render.FileSpec, error) {
	tmpl := template.Must(template.New("nodes_pkg").Funcs(funcMap).Parse(nodesPkgTmpl))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Pkg string
	}{pkg}); err != nil {
		return render.FileSpec{}, err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return render.FileSpec{}, fmt.Errorf("format nodes/nodes.go: %w\n%s", err, buf.String())
	}
	return render.FileSpec{
		Path:         dir + "/nodes/nodes.go",
		Mode:         0o644,
		Content:      src,
		KeepIfExists: true,
	}, nil
}

func (b *Backend) emitNodeStub(ent *ir.Entity, ep ir.CustomEndpoint, pkg, dir string) (render.FileSpec, error) {
	tmpl := template.Must(template.New("node_stub").Funcs(funcMap).Parse(nodeStubTmpl))
	fn := nodeFunc(ep)
	requestType := ep.Request
	if requestType == "" {
		requestType = fn + "Request"
	}
	responseType := ep.Response
	if responseType == "" {
		responseType = fn + "Response"
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Endpoint     ir.CustomEndpoint
		Func         string
		RequestType  string
		ResponseType string
		EntityName   string
	}{ep, fn, requestType, responseType, ent.Name}); err != nil {
		return render.FileSpec{}, err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return render.FileSpec{}, fmt.Errorf("format node stub %s: %w\n%s", fn, err, buf.String())
	}
	return render.FileSpec{
		Path:         dir + "/nodes/" + nodeStubFile(ent, ep),
		Mode:         0o644,
		Content:      src,
		KeepIfExists: true,
	}, nil
}

func (b *Backend) emitNodeHandler(app *ir.Application, mod ir.Module, ent *ir.Entity, endpoints []ir.CustomEndpoint, pkg, dir string) (render.FileSpec, error) {
	tmpl := template.Must(template.New("node_handler").Funcs(funcMap).Parse(nodeHandlerTmpl))

	type endpointView struct {
		Endpoint     ir.CustomEndpoint
		Func         string
		RequestType  string
		ResponseType string
		HTTPMethod   string
		Path         string
		HasRoles     bool
		Roles        []string
	}

	views := make([]endpointView, 0, len(endpoints))
	hasAnyAuth := false
	hasAnyRoles := false
	for _, ep := range endpoints {
		fn := nodeFunc(ep)
		req := ep.Request
		if req == "" {
			req = fn + "Request"
		}
		resp := ep.Response
		if resp == "" {
			resp = fn + "Response"
		}
		if ep.AuthRequired {
			hasAnyAuth = true
		}
		if len(ep.RolesAllowed) > 0 {
			hasAnyRoles = true
		}
		method := strings.ToUpper(strings.TrimSpace(ep.Method))
		if method == "" {
			method = "POST"
		}
		views = append(views, endpointView{
			Endpoint:     ep,
			Func:         fn,
			RequestType:  req,
			ResponseType: resp,
			HTTPMethod:   method,
			Path:         ep.Path,
			HasRoles:     len(ep.RolesAllowed) > 0,
			Roles:        ep.RolesAllowed,
		})
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].Func < views[j].Func })

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		ModulePath   string
		Pkg          string
		ModuleName   string
		Entity       *ir.Entity
		Endpoints    []endpointView
		HasAnyAuth   bool
		HasAnyRoles  bool
		MultiTenant  bool
	}{
		ModulePath:  b.ModulePath,
		Pkg:         pkg,
		ModuleName:  mod.Name,
		Entity:      ent,
		Endpoints:   views,
		HasAnyAuth:  hasAnyAuth,
		HasAnyRoles: hasAnyRoles,
		MultiTenant: app.Global.MultiTenancy.Enabled,
	}); err != nil {
		return render.FileSpec{}, err
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return render.FileSpec{}, fmt.Errorf("format %s node handler: %w\n%s", ent.Name, err, buf.String())
	}
	return render.FileSpec{
		Path:    dir + "/" + snake(ent.Name) + "_node_handler.go",
		Mode:    0o644,
		Content: src,
		Header:  headerLine,
	}, nil
}

const nodesPkgTmpl = `// Package nodes holds developer-authored business logic for the
// {{.Pkg}} module. The vibeguard generator scaffolds Nodes and Deps once
// (this file is preserved across re-generates) and one stub file per
// node-backed custom endpoint. Edit the stub bodies; do not edit the
// per-entity *_node_handler.go wrappers.
//
// The framework owns the wrappers: parsing, validation, auth/role checks,
// tenant binding, and tracing all happen there. Nodes only contain business
// logic and only touch the platform SDK through the Deps struct.
package nodes

import (
	"github.com/vibeguard/platform/db"
	"github.com/vibeguard/platform/events"
	"github.com/vibeguard/platform/llm"
	"go.uber.org/zap"
)

// Nodes is the receiver for every node method. Add fields here to hold
// per-application state (caches, prepared statements, third-party clients).
// Re-running the generator preserves this file.
type Nodes struct{}

// New returns a Nodes ready to receive method calls.
func New() *Nodes { return &Nodes{} }

// Deps is the platform-SDK surface a node may use. The generated wrapper
// builds this struct on every request, with tenant + user already bound on
// the context the node receives.
//
// Fields can be nil when the application doesn't configure that subsystem
// (e.g. LLM is nil if no integration declares an llm provider). Nodes that
// rely on a particular dep should nil-check it.
type Deps struct {
	DB     db.DB
	Bus    events.Publisher
	LLM    llm.Gateway
	Logger *zap.Logger
}
`

const nodeStubTmpl = `// Developer-owned business logic for endpoint
//   {{.Endpoint.Method}} {{.Endpoint.Path}}
// generated once by vibeguard; the wrapper that calls this lives in
// ../{{snake .EntityName}}_node_handler.go and is regenerated.
package nodes

import (
	"context"
	"errors"
)

// {{.RequestType}} is the validated request body. The vibeguard wrapper has
// already parsed JSON into this struct and rejected malformed input.
type {{.RequestType}} struct {
	// TODO: add request fields.
}

// {{.ResponseType}} is the response body returned to the caller.
type {{.ResponseType}} struct {
	// TODO: add response fields.
}

// {{.Func}} implements the business logic for {{.Endpoint.Method}} {{.Endpoint.Path}}.
// ctx already carries tenant + user bindings; d exposes the platform SDK.
// Return an error to signal failure — the wrapper maps it to an HTTP status.
func (n *Nodes) {{.Func}}(ctx context.Context, req {{.RequestType}}, d Deps) ({{.ResponseType}}, error) {
	_ = ctx
	_ = req
	_ = d
	return {{.ResponseType}}{}, errors.New("{{.Func}}: not implemented")
}
`

const nodeHandlerTmpl = `package {{.Pkg}}

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/vibeguard/platform/db"
	"github.com/vibeguard/platform/events"
	"github.com/vibeguard/platform/llm"

	"{{.ModulePath}}/internal/{{.Pkg}}/nodes"
)

// {{goName .Entity.Name}}NodeHandler routes node-backed custom endpoints declared
// on entity {{.Entity.Name}}. Wrappers handle parse, validate, auth, role
// enforcement, and tenant binding; business logic lives in the nodes package.
type {{goName .Entity.Name}}NodeHandler struct {
	nodes *nodes.Nodes
	deps  nodes.Deps
}

// New{{goName .Entity.Name}}NodeHandler builds the wrapper.
func New{{goName .Entity.Name}}NodeHandler(database db.DB, bus events.Publisher, gw llm.Gateway, logger *zap.Logger) *{{goName .Entity.Name}}NodeHandler {
	return &{{goName .Entity.Name}}NodeHandler{
		nodes: nodes.New(),
		deps: nodes.Deps{
			DB:     database,
			Bus:    bus,
			LLM:    gw,
			Logger: logger,
		},
	}
}

{{ range .Endpoints }}
// {{.Func}} wraps node ` + "`{{.Endpoint.Node}}`" + `.
// Endpoint: {{.HTTPMethod}} {{.Endpoint.Path}}
func (h *{{goName $.Entity.Name}}NodeHandler) {{.Func}}(c *gin.Context) {
	ctx := c.Request.Context()
{{- if $.MultiTenant }}
	rc, _ := db.FromContext(ctx)
{{- end }}
{{- if .Endpoint.AuthRequired }}
{{- if $.MultiTenant }}
	if rc.UserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "auth required"})
		return
	}
{{- else }}
	// auth required — declaration says auth_required: true; integrate your
	// auth middleware so the request never reaches here unauthenticated.
{{- end }}
{{- end }}
{{- if .HasRoles }}
	if !nodeRoleAllowed(rc.Role, []string{ {{ range $i, $r := .Roles }}{{ if $i }}, {{ end }}{{ printf "%q" $r }}{{ end }} }) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
{{- end }}

	var req nodes.{{.RequestType}}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.nodes.{{.Func}}(ctx, req, h.deps)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
{{ end }}

// Register{{goName .Entity.Name}}NodeRoutes registers every node-backed route
// declared on entity {{.Entity.Name}}.
func Register{{goName .Entity.Name}}NodeRoutes(r gin.IRouter, h *{{goName .Entity.Name}}NodeHandler) {
{{- range .Endpoints }}
	r.{{ginVerb .HTTPMethod}}("{{.Path}}", h.{{.Func}})
{{- end }}
}

{{ if .HasAnyRoles }}
// nodeRoleAllowed reports whether role appears in allowed.
func nodeRoleAllowed(role string, allowed []string) bool {
	for _, a := range allowed {
		if a == role {
			return true
		}
	}
	return false
}
{{ end }}

// keep imports referenced even when unused by some endpoint variants.
var (
	_ = http.StatusOK
	_ = (*gin.Context)(nil)
	_ = (*zap.Logger)(nil)
	_ events.Publisher
	_ llm.Gateway
{{- if not .MultiTenant }}
	_ = db.RequestContext{}
{{- end }}
)
`
