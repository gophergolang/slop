package main

import (
	"encoding/json"
	"os"

	"github.com/vibeguard/vibeguard/internal/static"
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
	text, mime := contentForURI(p.URI)
	if text == "" && mime == "" {
		s.respondError(req.ID, -32602, "unknown resource: "+p.URI)
		return
	}
	s.respond(req.ID, map[string]any{
		"contents": []map[string]any{
			{
				"uri":      p.URI,
				"mimeType": mime,
				"text":     text,
			},
		},
	})
}

// contentForURI resolves vibeguard:// URIs to their text content.
// When VIBEGUARD_REPO_ROOT is set the file on disk takes precedence over
// the embedded copy, so local development edits are reflected immediately.
func contentForURI(uri string) (text, mime string) {
	switch uri {
	case "vibeguard://prompts/master":
		return readOrEmbed("vibeguard_master_prompt.md", static.MasterPrompt), "text/markdown"
	case "vibeguard://schema/declaration.json":
		return readOrEmbed("vibeguard_declaration_schema.json", static.DeclarationSchema), "application/json"
	}
	return "", ""
}

func readOrEmbed(filename, embedded string) string {
	repo := os.Getenv("VIBEGUARD_REPO_ROOT")
	if repo == "" {
		return embedded
	}
	data, err := os.ReadFile(repo + "/" + filename)
	if err != nil {
		return embedded
	}
	return string(data)
}
