package main

import (
	"encoding/json"
	"os"
)

type resourceDescriptor struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description"`
	MimeType    string `json:"mimeType"`
}

func resourceDescriptors() []resourceDescriptor {
	return []resourceDescriptor{
		{
			URI:         "vibeguard://prompts/master",
			Name:        "Vibeguard Master Prompt",
			Description: "The master prompt that establishes Guarded Vibe Coding (GVC) discipline. Read this before producing or editing a vibeguard.yaml.",
			MimeType:    "text/markdown",
		},
		{
			URI:         "vibeguard://schema/declaration.json",
			Name:        "Vibeguard Declaration JSON Schema",
			Description: "JSON Schema for a vibeguard.yaml declaration. Validate against this before calling generate_project.",
			MimeType:    "application/json",
		},
	}
}

func (s *server) handleResourcesRead(req rpcRequest) {
	var p struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.respondError(req.ID, -32602, err.Error())
		return
	}
	path, mime := pathForURI(p.URI)
	if path == "" {
		s.respondError(req.ID, -32602, "unknown resource: "+p.URI)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		s.respondError(req.ID, -32603, err.Error())
		return
	}
	s.respond(req.ID, map[string]any{
		"contents": []map[string]any{
			{
				"uri":      p.URI,
				"mimeType": mime,
				"text":     string(data),
			},
		},
	})
}

// pathForURI resolves vibeguard:// URIs to repo files. In a hosted MCP these
// would be embedded; for local-stdio runs we read from the working repo.
func pathForURI(uri string) (path, mime string) {
	repo := vibeguardRepoRoot()
	switch uri {
	case "vibeguard://prompts/master":
		return repo + "/vibeguard_master_prompt.md", "text/markdown"
	case "vibeguard://schema/declaration.json":
		return repo + "/vibeguard_declaration_schema.json", "application/json"
	}
	return "", ""
}

func vibeguardRepoRoot() string {
	if env := os.Getenv("VIBEGUARD_REPO_ROOT"); env != "" {
		return env
	}
	wd, _ := os.Getwd()
	return wd
}
