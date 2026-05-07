package antigravity

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
	return "antigravity"
}

func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(account.BaseURL, "/")
	if base == "" {
		base = "https://cloudcode-pa.googleapis.com"
	}
	url := base + path
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return url
}

func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "antigravity/1.21.9 windows/amd64")
	return nil
}

func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	var payload struct {
		Response struct {
			UsageMetadata struct {
				PromptTokenCount     int64 `json:"promptTokenCount"`
				CandidatesTokenCount int64 `json:"candidatesTokenCount"`
				TotalTokenCount      int64 `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		} `json:"response"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	in := payload.Response.UsageMetadata.PromptTokenCount
	out := payload.Response.UsageMetadata.CandidatesTokenCount
	if out == 0 && payload.Response.UsageMetadata.TotalTokenCount > in {
		out = payload.Response.UsageMetadata.TotalTokenCount - in
	}
	return in, out
}

func (p *Provider) SupportsPath(path string) bool {
	return strings.HasPrefix(path, "/v1internal:")
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
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) {
			continue
		}
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		in, out, ok := extractAntigravityUsage(payload)
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

func extractAntigravityUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usageMetadata"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["promptTokenCount"])
				out := int64Value(usage["candidatesTokenCount"])
				total := int64Value(usage["totalTokenCount"])
				if out == 0 && total > in {
					out = total - in
				}
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		for _, child := range value {
			if in, out, ok := extractAntigravityUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		for _, child := range value {
			if in, out, ok := extractAntigravityUsage(child); ok {
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
