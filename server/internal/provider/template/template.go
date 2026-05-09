package template

import (
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
)

func New(_ config.Config) *Provider {
	return &Provider{}
}

func (p *Provider) Name() string {
	return "template"
}

// Contract 明确该 provider 的能力边界。
// 新增 provider 时，必须先把这个契约改正确，再写具体实现。
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: true,
		RequiresStreamUsageParser:      true,
		RequiresCacheUsageParser:       true,
		SupportsOAuth:                  false,
	}
}

// NormalizeGateway 负责把公开 OpenAI 协议请求归一化为该 provider 的上游语义。
func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	req.Provider = p.Name()
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost
	}
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath
	}

	_ = account
	_ = cred
	// TODO: 在这里做 OpenAI 请求 -> 上游原生请求转换，并填写：
	// req.Body / req.InternalPath / req.Stream / req.UpstreamStream / req.IncludeUsage
	return req, nil
}

// BuildUpstreamURL 构建上游请求 URL。
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	if base == "" {
		base = "https://example.com"
	}
	if rawQuery == "" {
		return base + path
	}
	return base + path + "?" + rawQuery
}

// ApplyRequest 设置上游认证头。
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	_ = account
	return nil
}

// ParseUsage 解析非流式 usage。
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
func (p *Provider) SupportsPath(path string) bool {
	switch path {
	case "/v1/chat/completions", "/v1/responses":
		return true
	default:
		return false
	}
}

// ParseStreamUsage 解析流式 usage。
func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	_ = body
	// TODO: 从流式事件中提取 usage
	return 0, 0, false
}

// ParseCacheUsage 解析缓存 usage。
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	_ = body
	// TODO: 提取缓存读/写 token
	return 0, 0, false
}

// AccountCapability 返回账号配置能力。
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

// 如果这个 provider 支持 OAuth，再补下面三个接口：
func (p *Provider) DefaultOAuthRedirectURI(meta map[string]string) string {
	_ = meta
	return ""
}

func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	_ = input
	return "", nil
}

func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	_ = ctx
	_ = client
	_ = input
	return nil, nil
}

func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	_ = ctx
	_ = client
	_ = input
	return nil, nil
}
