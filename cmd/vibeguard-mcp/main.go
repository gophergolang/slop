// Command vibeguard-mcp is a Model Context Protocol server exposing
// vibeguard's compiler + linter + LLM gateway as tools any MCP client
// (Claude Desktop, Claude Code, Cursor, Zed) can call.
//
// Transport: stdio JSON-RPC 2.0 (MCP spec). One process per workspace.
//
// Configure in Claude Desktop's claude_desktop_config.json:
//
//	{
//	  "mcpServers": {
//	    "vibeguard": { "command": "vibeguard-mcp" }
//	  }
//	}
//
// Tools exposed (branch 4-7):
//
//   - validate_declaration  : Validate a vibeguard.yaml against schema + rules
//   - generate_project      : Render a project from a declaration
//   - lint_project          : Run master-prompt analyzers over a Go project
//
// Resources:
//
//   - vibeguard://prompts/master           — the master prompt
//   - vibeguard://schema/declaration.json  — the declaration JSON schema
//
// query_runtime_state and propose_remediation land in the follow-up branch
// once the operator's status API is reachable.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func main() {
	srv := newServer()
	srv.serve(os.Stdin, os.Stdout)
}

type server struct {
	w *bufio.Writer
}

func newServer() *server { return &server{} }

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *server) serve(r io.Reader, w io.Writer) {
	s.w = bufio.NewWriter(w)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1<<20), 1<<24)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.respondError(nil, -32700, "parse error: "+err.Error())
			continue
		}
		s.handle(req)
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "vibeguard-mcp: read: %v\n", err)
	}
}

func (s *server) handle(req rpcRequest) {
	switch req.Method {
	case "initialize":
		s.respond(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]any{"name": "vibeguard-mcp", "version": "0.7.0-4-7"},
		})
	case "tools/list":
		s.respond(req.ID, map[string]any{"tools": toolDescriptors()})
	case "tools/call":
		s.handleToolsCall(req)
	case "resources/list":
		s.respond(req.ID, map[string]any{"resources": resourceDescriptors()})
	case "resources/read":
		s.handleResourcesRead(req)
	case "ping":
		s.respond(req.ID, map[string]any{})
	case "shutdown":
		s.respond(req.ID, nil)
		os.Exit(0)
	default:
		if req.ID != nil {
			s.respondError(req.ID, -32601, "method not found: "+req.Method)
		}
	}
}

func (s *server) respond(id json.RawMessage, result any) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
	out, _ := json.Marshal(resp)
	s.w.Write(out)
	s.w.WriteByte('\n')
	s.w.Flush()
}

func (s *server) respondError(id json.RawMessage, code int, msg string) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
	out, _ := json.Marshal(resp)
	s.w.Write(out)
	s.w.WriteByte('\n')
	s.w.Flush()
}
