package claude

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"sub2api/server/internal/model"
)

type Provider struct{}

func New() *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "claude"
}

func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(account.BaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	url := base + path
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return url
}

func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("x-api-key", token)
	req.Header.Del("Authorization")
	return nil
}

func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	var payload struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	return payload.Usage.InputTokens, payload.Usage.OutputTokens
}

func (p *Provider) SupportsPath(path string) bool {
	switch path {
	case "/v1/messages", "/v1/messages/count_tokens":
		return true
	default:
		return false
	}
}

func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	var inMax int64
	var outMax int64
	var found bool
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(line) == 0 {
			continue
		}
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		in, out, ok := extractClaudeUsage(payload)
		if !ok {
			continue
		}
		found = true
		if in > inMax {
			inMax = in
		}
		if out > outMax {
			outMax = out
		}
	}
	return inMax, outMax, found
}

func extractClaudeUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["input_tokens"])
				out := int64Value(usage["output_tokens"])
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		for _, child := range value {
			if in, out, ok := extractClaudeUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		for _, child := range value {
			if in, out, ok := extractClaudeUsage(child); ok {
				return in, out, true
			}
		}
	}
	return 0, 0, false
}

func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
