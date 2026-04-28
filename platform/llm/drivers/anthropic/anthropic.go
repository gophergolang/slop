// Package anthropic is the Anthropic Claude driver for the platform/llm gateway.
//
// The driver speaks the Messages API directly via net/http to keep dependency
// surface tight; no large SDK pulled in for what is fundamentally one POST.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/vibeguard/platform/llm"
)

// Driver implements llm.Driver against the Anthropic Messages API.
type Driver struct {
	APIKey  string
	BaseURL string // defaults to https://api.anthropic.com
	HTTP    *http.Client
}

// New constructs a Driver. APIKey defaults to ANTHROPIC_API_KEY env var.
func New() *Driver {
	return &Driver{
		APIKey:  os.Getenv("ANTHROPIC_API_KEY"),
		BaseURL: "https://api.anthropic.com",
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Name reports the driver name (used in metrics + cost attribution).
func (d *Driver) Name() string { return "anthropic" }

// Complete invokes the Anthropic Messages API.
func (d *Driver) Complete(ctx context.Context, req llm.DriverRequest) (llm.DriverResponse, error) {
	if d.APIKey == "" {
		return llm.DriverResponse{}, fmt.Errorf("anthropic: no API key (set ANTHROPIC_API_KEY)")
	}
	body := messagesRequest{
		Model:     req.Model,
		System:    req.System,
		MaxTokens: maxTokensOrDefault(req.MaxTokens),
		Messages:  toAPIMessages(req.Messages),
	}
	buf, err := json.Marshal(body)
	if err != nil {
		return llm.DriverResponse{}, fmt.Errorf("marshal: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.BaseURL+"/v1/messages", bytes.NewReader(buf))
	if err != nil {
		return llm.DriverResponse{}, err
	}
	httpReq.Header.Set("content-type", "application/json")
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("x-api-key", d.APIKey)

	resp, err := d.HTTP.Do(httpReq)
	if err != nil {
		return llm.DriverResponse{}, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode/100 != 2 {
		return llm.DriverResponse{}, fmt.Errorf("anthropic: status %d: %s", resp.StatusCode, string(respBody))
	}
	var out messagesResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return llm.DriverResponse{}, fmt.Errorf("decode: %w", err)
	}
	text := ""
	for _, c := range out.Content {
		if c.Type == "text" {
			text += c.Text
		}
	}
	return llm.DriverResponse{
		Text:         text,
		PromptTokens: out.Usage.InputTokens,
		OutputTokens: out.Usage.OutputTokens,
	}, nil
}

func maxTokensOrDefault(n int) int {
	if n > 0 {
		return n
	}
	return 4096
}

func toAPIMessages(in []llm.Message) []apiMessage {
	out := make([]apiMessage, 0, len(in))
	for _, m := range in {
		out = append(out, apiMessage{Role: m.Role, Content: m.Content})
	}
	return out
}

type messagesRequest struct {
	Model     string       `json:"model"`
	System    string       `json:"system,omitempty"`
	MaxTokens int          `json:"max_tokens"`
	Messages  []apiMessage `json:"messages"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
