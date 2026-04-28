package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type promptDescriptor struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Arguments   []promptArgSpec `json:"arguments,omitempty"`
}

type promptArgSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

func promptDescriptors() []promptDescriptor {
	return []promptDescriptor{
		{
			Name:        "new_saas_project",
			Description: "Start a new SaaS project — vibeguard drafts your backend declaration, generates the full service (Go API + Postgres + K8s + OpenAPI + Next.js admin UI), and guides deployment to fly.io, Railway, or GCP Cloud Run.",
			Arguments: []promptArgSpec{
				{
					Name:        "description",
					Description: "What the SaaS does (e.g. 'a project management tool for small teams'). Provide as much or as little as you know — vibeguard will ask for the rest.",
					Required:    false,
				},
			},
		},
	}
}

func (s *server) handlePromptsGet(req rpcRequest) {
	var p struct {
		Name      string            `json:"name"`
		Arguments map[string]string `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.respondError(req.ID, -32602, err.Error())
		return
	}
	switch p.Name {
	case "new_saas_project":
		s.respond(req.ID, newSaasProjectPrompt(p.Arguments["description"]))
	default:
		s.respondError(req.ID, -32602, "unknown prompt: "+p.Name)
	}
}

func newSaasProjectPrompt(description string) map[string]any {
	var opening string
	if description != "" {
		opening = fmt.Sprintf("The user wants to build: %s\n\n", description)
	} else {
		opening = "Start by asking the user in one sentence what their SaaS does — who uses it and what's the core value.\n\n"
	}

	body := strings.TrimSpace(`
You are using vibeguard to generate the backend for a new SaaS project.
vibeguard turns a single YAML declaration into a production-grade Go API, Postgres schema, Kubernetes manifests, OpenAPI spec, and a Next.js admin UI.
The developer writes only business logic; everything else is generated.

## Your workflow

**Step 1 — Understand the domain**
` + opening + `Ask clarifying questions until you have enough to draft:
- The main data entities (e.g. Team, Task, Invoice) and how they nest (parent-child)
- The CRUD operations each entity needs
- Any custom business logic endpoints (e.g. /tasks/:id/prioritize)
- Whether multi-tenancy is required (almost always yes for SaaS)
- Auth requirements (register/login/refresh are generated automatically)

**Step 2 — Read the declaration format**
Read the resource vibeguard://schema/declaration.json so you understand the exact YAML structure.
Read vibeguard://prompts/master for the security and architecture rules.

**Step 3 — Draft the vibeguard.yaml**
Write a complete vibeguard.yaml. Show it to the user and explain the key decisions:
- Which entities are parent→child (this drives nested URLs and FK cascades)
- Which fields are included in CRUD update whitelists
- Where node: endpoints are placed for custom business logic
Ask the user to review the declaration before proceeding.

**Step 4 — Validate**
Call validate_declaration with the YAML text.
If there are errors, fix them and validate again. Explain any warnings to the user.

**Step 5 — Generate**
Ask the user where to put the generated project (default: ./<app-name>).
Call generate_project with:
  - yaml: the validated declaration text
  - out_dir: the chosen directory
  - module_path: github.com/<user>/<app-name> (ask if unsure)
Report what was generated: Go service, SQL migrations, K8s manifests, OpenAPI spec, Next.js admin UI in web/.

**Step 6 — Orient the developer**
Show the developer the two things they own:
1. The vibeguard.yaml — their source of truth; re-run generate_project any time they change it
2. internal/<module>/nodes/*.go stubs — where they write business logic; these files survive re-generation

Show the frontend next step: cd <out_dir>/web && npm install && npm run dev

**Step 7 — Deployment guidance (ask first)**
Ask: "Would you like help deploying this to a cloud platform?"
If yes, ask which platform:

- **fly.io** (fastest for solo developers):
  1. fly launch --no-deploy (auto-detects the Dockerfile)
  2. fly postgres create && fly postgres attach
  3. Set secrets: fly secrets set DATABASE_URL=... NATS_URL=...
  4. fly deploy

- **Railway** (zero-config, good for small teams):
  1. Connect the GitHub repo in the Railway dashboard
  2. Add a Postgres plugin — Railway injects DATABASE_URL automatically
  3. Set remaining env vars in the Railway UI
  4. Deploy triggers on every push to main

- **GCP Cloud Run** (scales to zero, good for variable traffic):
  1. Build and push the Docker image to Artifact Registry
  2. Apply the generated k8s/ manifests to a GKE cluster, or use Cloud Run directly with the Dockerfile
  3. Use Cloud SQL for Postgres and configure the DATABASE_URL secret in Secret Manager

Walk through whichever platform they choose step by step.
`)

	return map[string]any{
		"description": "Start a new SaaS project with vibeguard",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": map[string]any{
					"type": "text",
					"text": body,
				},
			},
		},
	}
}
