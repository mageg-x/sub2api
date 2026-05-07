package openai

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"sub2api/server/internal/model"
)

// Provider OpenAI API Provider实现
// 支持OpenAI兼容的API接口
type Provider struct{}

// New 创建OpenAI Provider实例
func New() *Provider {
	return &Provider{}
}

// Name 返回Provider名称
func (p *Provider) Name() string {
	return "openai"
}

// BuildUpstreamURL 构建OpenAI API的完整URL
// 如果账号配置了自定义BaseURL则使用，否则使用默认的api.openai.com
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(account.BaseURL, "/")
	if base == "" {
		// 默认使用OpenAI官方API
		base = "https://api.openai.com"
	}
	url := base + path
	if rawQuery != "" {
		url += "?" + rawQuery
	}
	return url
}

// ApplyRequest 为请求添加OpenAI所需的认证头
// 使用Bearer Token进行认证
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return nil
}

// ParseUsage 从响应体解析token使用量
// 解析OpenAI API响应中的usage字段
func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	var payload struct {
		Usage struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			InputTokens      int64 `json:"input_tokens"`
			OutputTokens     int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	// 优先使用prompt_tokens/completion_tokens（旧版字段名）
	in := payload.Usage.PromptTokens
	out := payload.Usage.CompletionTokens
	// 备用input_tokens/output_tokens（新版字段名）
	if in == 0 {
		in = payload.Usage.InputTokens
	}
	if out == 0 {
		out = payload.Usage.OutputTokens
	}
	return in, out
}

// SupportsPath 判断OpenAI Provider支持的API路径
func (p *Provider) SupportsPath(path string) bool {
	switch path {
	case "/v1/chat/completions", "/v1/responses", "/v1/embeddings":
		return true
	default:
		return false
	}
}

// ParseStreamUsage 解析流式响应中的使用量
// 遍历流式数据块，找出usage信息
func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	var inMax int64
	var outMax int64
	var found bool
	// 按行分割流式响应
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		// 跳过非data行
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		// 跳过空行和结束标记
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) {
			continue
		}
		// 解析JSON
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		// 提取使用量
		in, out, ok := extractOpenAIUsage(payload)
		if !ok {
			continue
		}
		found = true
		// 取最大值
		if in > inMax {
			inMax = in
		}
		if out > outMax {
			outMax = out
		}
	}
	return inMax, outMax, found
}

// extractOpenAIUsage 从任意JSON结构中递归提取usage信息
func extractOpenAIUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 检查是否有usage字段
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["prompt_tokens"])
				out := int64Value(usage["completion_tokens"])
				if in == 0 {
					in = int64Value(usage["input_tokens"])
				}
				if out == 0 {
					out = int64Value(usage["output_tokens"])
				}
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		// 递归检查子元素
		for _, child := range value {
			if in, out, ok := extractOpenAIUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		for _, child := range value {
			if in, out, ok := extractOpenAIUsage(child); ok {
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
	return extractOpenAICacheUsage(payload)
}

func extractOpenAICacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				create := int64Value(usage["cache_creation_input_tokens"])
				if create == 0 {
					create = int64Value(usage["cache_creation_tokens"])
				}
				read := int64Value(usage["cache_read_input_tokens"])
				if read == 0 {
					read = int64Value(usage["cache_read_tokens"])
				}
				if create > 0 || read > 0 {
					return create, read, true
				}
				if details, ok := usage["input_tokens_details"].(map[string]any); ok {
					create = int64Value(details["cache_creation_input_tokens"])
					read = int64Value(details["cache_read_input_tokens"])
					if create > 0 || read > 0 {
						return create, read, true
					}
				}
			}
		}
		for _, child := range value {
			if create, read, ok := extractOpenAICacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		for _, child := range value {
			if create, read, ok := extractOpenAICacheUsage(child); ok {
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
