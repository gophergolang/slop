// Package static embeds the vibeguard master prompt and declaration schema
// into the binary so the MCP server works without VIBEGUARD_REPO_ROOT.
package static

import _ "embed"

//go:embed master_prompt.md
var MasterPrompt string

//go:embed declaration_schema.json
var DeclarationSchema string
