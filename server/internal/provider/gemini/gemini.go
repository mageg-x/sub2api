package gemini

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"sub2api/server/internal/model"
)

// Provider Gemini API Provider实现
// 支持Google Gemini API接口
type Provider struct{}

// New 创建Gemini Provider实例
func New() *Provider {
	return &Provider{}
}

// Name 返回Provider名称
func (p *Provider) Name() string {
	return "gemini"
}

// BuildUpstreamURL 构建Gemini API的完整URL
// 标准 Gemini 路由使用 generativelanguage.googleapis.com。
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(account.BaseURL, "/")
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	url := base + path
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return url
}

// ApplyRequest 为请求添加Gemini所需的认证头
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// ParseUsage 从响应体解析token使用量
// 解析Gemini API响应中的usageMetadata字段
func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	var payload struct {
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`
			CandidatesTokenCount int64 `json:"candidatesTokenCount"`
			TotalTokenCount      int64 `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	in := payload.UsageMetadata.PromptTokenCount
	out := payload.UsageMetadata.CandidatesTokenCount
	// 如果输出token为0但有总数，计算输出
	if out == 0 && payload.UsageMetadata.TotalTokenCount > in {
		out = payload.UsageMetadata.TotalTokenCount - in
	}
	return in, out
}

// SupportsPath 判断Gemini Provider支持的API路径
func (p *Provider) SupportsPath(path string) bool {
	return strings.HasPrefix(path, "/v1beta/models/") || strings.HasPrefix(path, "/v1/models/")
}

// ParseStreamUsage 解析流式响应中的使用量
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
		in, out, ok := extractGeminiUsage(payload)
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

// extractGeminiUsage 从任意JSON结构中递归提取usageMetadata信息
func extractGeminiUsage(v any) (int64, int64, bool) {
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
			if in, out, ok := extractGeminiUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		for _, child := range value {
			if in, out, ok := extractGeminiUsage(child); ok {
				return in, out, true
			}
		}
	}
	return 0, 0, false
}

// ParseCacheUsage 解析缓存相关token
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractGeminiCacheUsage(payload)
}

func extractGeminiCacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usageMetadata"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				create := int64Value(usage["cachedContentTokenCount"])
				if create == 0 {
					create = int64Value(usage["cacheCreationTokenCount"])
				}
				read := int64Value(usage["cacheReadTokenCount"])
				if read == 0 {
					read = int64Value(usage["cachedInputTokenCount"])
				}
				if create > 0 || read > 0 {
					return create, read, true
				}
			}
		}
		for _, child := range value {
			if create, read, ok := extractGeminiCacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		for _, child := range value {
			if create, read, ok := extractGeminiCacheUsage(child); ok {
				return create, read, true
			}
		}
	}
	return 0, 0, false
}

// int64Value 安全地将任意类型转换为int64
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
