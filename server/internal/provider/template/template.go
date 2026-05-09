package template

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// Provider 是新增上游 provider 时的标准骨架。
// 复制本目录后，优先改 Name / Contract / NormalizeGateway / 协议转换逻辑。
type Provider struct{}

// 这些编译期断言用于强制新 provider 明确自己实现了哪些能力。
var (
	_ provider.Provider                  = (*Provider)(nil)
	_ provider.ContractProvider          = (*Provider)(nil)
	_ provider.AccountCapabilityProvider = (*Provider)(nil)
	_ provider.GatewayResponseAdapter    = (*Provider)(nil)
	_ provider.CacheUsageParser          = (*Provider)(nil)
	_ provider.StreamUsageParser         = (*Provider)(nil)
)

// New 创建 Template Provider 实例
func New(_ config.Config) *Provider {
	return &Provider{}
}

// Name 返回 provider 名称标识
func (p *Provider) Name() string {
	return "template"
}

// Contract 明确该 provider 的能力边界。
// 新增 provider 时，必须先把这个契约改正确，再写具体实现。
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: true,  // 需要响应适配器
		RequiresStreamUsageParser:      true,  // 需要流式用量解析
		RequiresCacheUsageParser:       true,  // 需要缓存用量解析
		SupportsOAuth:                  false, // 不支持 OAuth
	}
}

// NormalizeGateway 负责把公开 OpenAI 协议请求归一化为该 provider 的上游语义。
// 对于 OpenAI 兼容路径，转换为上游原生格式
func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	req.Provider = p.Name()
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost // 默认 POST 方法
	}
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath // 兜底：使用公开路径
	}

	_ = account
	_ = cred
	// 将 OpenAI 兼容格式转换为上游原生格式
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions", "/v1/responses", "/backend-api/codex/responses":
		converted, modelName, stream, includeUsage, err := convertOpenAIRequest(req.PublicPath, req.Body)
		if err != nil {
			return req, err
		}
		req.Body = converted            // 替换为上游原生请求体
		req.Model = modelName           // 更新模型名
		req.Stream = stream             // 更新流式标志
		req.IncludeUsage = includeUsage // 更新用量标志
		req.UpstreamStream = stream     // 上游使用流式
		req.UpstreamMethod = http.MethodPost
	}
	return req, nil
}

// BuildUpstreamURL 构建上游请求 URL。
// 拼接账号 BaseURL 和路径，支持查询参数
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	if base == "" {
		base = "https://example.com" // 默认 BaseURL
	}
	if rawQuery == "" {
		return base + path
	}
	return base + path + "?" + rawQuery
}

// ApplyRequest 设置上游认证头。
// 使用 Bearer Token 认证方式
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	_ = account
	return nil
}

// ParseUsage 解析非流式 usage。
// 从响应体中提取 input_tokens 和 output_tokens
func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	var payload struct {
		Usage struct {
			Input  int64 `json:"input_tokens"`
			Output int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	return payload.Usage.Input, payload.Usage.Output
}

// SupportsPath 判断当前 provider 支持哪些路径。
// 支持 Chat Completions、Responses 和 Codex 路径
func (p *Provider) SupportsPath(path string) bool {
	switch path {
	case "/v1/chat/completions", "/chat/completions", "/v1/responses", "/responses", "/backend-api/codex/responses":
		return true
	default:
		// 支持带 ID 后缀的 Responses 路径
		return strings.HasPrefix(path, "/v1/responses/") ||
			strings.HasPrefix(path, "/responses/") ||
			strings.HasPrefix(path, "/backend-api/codex/responses/")
	}
}

// ParseStreamUsage 解析流式 usage。
// 遍历 SSE 数据行，提取最大的 input_tokens 和 output_tokens
func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	var inMax int64
	var outMax int64
	var found bool
	// 按行分割 SSE 数据
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue // 跳过非 data 行
		}
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) {
			continue // 跳过空行和 [DONE] 标记
		}
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		// 递归提取用量，取最大值
		in, out, ok := extractTemplateUsage(payload)
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

// ParseCacheUsage 解析缓存 usage。
// 从响应体中提取 cached_tokens 和 cache_creation_input_tokens
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractTemplateCacheUsage(payload)
}

// AccountCapability 返回账号配置能力。
// 定义了 UI 上显示的配置项和认证方式
func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "Template",
		Notice:             "新增 provider 时请根据上游实际认证方式修改。",
		DefaultBaseURL:     "https://example.com",
		BaseURLPlaceholder: "https://example.com",
		DefaultAuthMode:    "api_key",
		AuthModes: []provider.AccountAuthMode{
			{Value: "api_key", Label: "API Key"},
		},
		APIKeyField: provider.CapabilityField{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "textarea",
			Required:    true,
			Placeholder: "sk-...",
			Storage:     "credentials",
		},
	}
}

// DefaultOAuthRedirectURI 返回 OAuth 默认重定向 URI
// Template provider 不支持 OAuth，返回空字符串
func (p *Provider) DefaultOAuthRedirectURI(meta map[string]string) string {
	_ = meta
	return ""
}

// BuildOAuthAuthorizationURL 构建 OAuth 授权 URL
// Template provider 不支持 OAuth，返回空
func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	_ = input
	return "", nil
}

// ExchangeOAuthCode 使用授权码交换 OAuth Token
// Template provider 不支持 OAuth，返回空
func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	_ = ctx
	_ = client
	_ = input
	return nil, nil
}

// RefreshOAuthToken 刷新 OAuth Token
// Template provider 不支持 OAuth，返回空
func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	_ = ctx
	_ = client
	_ = input
	return nil, nil
}

// extractTemplateUsage 递归提取用量信息
// 在 JSON 结构中查找 usage.input_tokens 和 usage.output_tokens
func extractTemplateUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 在当前对象中查找 usage 字段
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["input_tokens"])
				out := int64Value(usage["output_tokens"])
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		// 递归检查子对象
		for _, child := range value {
			if in, out, ok := extractTemplateUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		// 递归检查数组元素
		for _, child := range value {
			if in, out, ok := extractTemplateUsage(child); ok {
				return in, out, true
			}
		}
	}
	return 0, 0, false
}

// extractTemplateCacheUsage 递归提取缓存用量信息
// 在 JSON 结构中查找 usage.cached_tokens 和 usage.cache_creation_input_tokens
// 返回值：(cache_creation_input_tokens, cached_tokens, found)
func extractTemplateCacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 在当前对象中查找 usage 字段
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				read := int64Value(usage["cached_tokens"])
				create := int64Value(usage["cache_creation_input_tokens"])
				if read > 0 || create > 0 {
					return create, read, true
				}
			}
		}
		// 递归检查子对象
		for _, child := range value {
			if create, read, ok := extractTemplateCacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		// 递归检查数组元素
		for _, child := range value {
			if create, read, ok := extractTemplateCacheUsage(child); ok {
				return create, read, true
			}
		}
	}
	return 0, 0, false
}

// int64Value 将任意数值类型转换为 int64
// 支持 float64、int、int32、int64 和 json.Number
func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}
