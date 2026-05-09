package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// Provider Claude API Provider实现
// 支持Anthropic Claude API接口
type Provider struct {
	clientID string
}

// 编译期断言：Claude provider 必须持续满足核心契约。
var (
	_ provider.Provider                  = (*Provider)(nil)
	_ provider.ContractProvider          = (*Provider)(nil)
	_ provider.AccountCapabilityProvider = (*Provider)(nil)
	_ provider.GatewayResponseAdapter    = (*Provider)(nil)
	_ provider.CacheUsageParser          = (*Provider)(nil)
	_ provider.StreamUsageParser         = (*Provider)(nil)
	_ provider.OAuthStarter              = (*Provider)(nil)
	_ provider.OAuthExchanger            = (*Provider)(nil)
	_ provider.OAuthRefresher            = (*Provider)(nil)
)

// New 创建Claude Provider实例
func New(cfg config.Config) *Provider {
	return &Provider{clientID: cfg.Claude.ClientID}
}

const (
	authorizeURL    = "https://claude.ai/oauth/authorize"
	tokenURL        = "https://platform.claude.com/v1/oauth/token"
	redirectURI     = "https://platform.claude.com/oauth/code/callback"
	oauthScopeValue = "org:create_api_key user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
)

// Name 返回Provider名称
func (p *Provider) Name() string {
	return "claude"
}

// Contract 返回 Claude provider 的能力契约。
// Claude 需要在插件内部完成 OpenAI 协议到 Anthropic 原生协议的双向转换。
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: true,
		RequiresStreamUsageParser:      true,
		RequiresCacheUsageParser:       true,
		SupportsOAuth:                  true,
	}
}

// NormalizeGateway 归一化网关请求
// 对于 OpenAI 兼容路径（/v1/chat/completions、/v1/responses），转换为 Claude 原生 /v1/messages 格式
func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	req.Provider = p.Name()
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost // 默认 POST 方法
	}
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	_ = account
	_ = cred
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath // 兜底：使用公开路径
	}
	// 将 OpenAI 兼容格式转换为 Claude 原生格式
	switch normalizeOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions", "/v1/responses", "/backend-api/codex/responses":
		converted, modelName, stream, includeUsage, err := convertOpenAIRequest(req.PublicPath, req.Body)
		if err != nil {
			return req, err
		}
		req.Body = converted              // 替换为 Claude 原生请求体
		req.Model = modelName             // 更新模型名
		req.Stream = stream               // 更新流式标志
		req.IncludeUsage = includeUsage   // 更新用量标志
		req.InternalPath = "/v1/messages" // 统一路由到 Claude Messages 接口
		req.UpstreamMethod = http.MethodPost
		req.UpstreamStream = stream // 上游使用流式
	}
	// 模型列表接口使用 GET 方法
	if req.PublicPath == "/v1/models" {
		req.UpstreamMethod = http.MethodGet
	}
	return req, nil
}

// BuildUpstreamURL 构建Claude API的完整URL
// 使用api.anthropic.com作为默认端点
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

// ApplyRequest 为请求添加 Claude 所需的认证头
// OAuth 认证使用 Bearer Token，API Key 认证使用 x-api-key 头
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01") // Claude API 版本
	switch strings.TrimSpace(account.AuthType) {
	case "oauth":
		req.Header.Del("x-api-key") // OAuth 不需要 x-api-key
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		req.Header.Set("x-api-key", token) // API Key 认证
		req.Header.Del("Authorization")    // 不需要 Bearer Token
	}
	return nil
}

// ParseUsage 从响应体解析token使用量
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

// SupportsPath 判断Claude Provider支持的API路径
func (p *Provider) SupportsPath(path string) bool {
	switch path {
	case "/v1/messages", "/v1/messages/count_tokens", "/v1/messages/batches", "/v1/models",
		"/v1/chat/completions", "/chat/completions", "/v1/responses", "/responses", "/backend-api/codex/responses":
		return true
	default:
		return strings.HasPrefix(path, "/v1/messages/batches/") ||
			strings.HasPrefix(path, "/v1/responses/") ||
			strings.HasPrefix(path, "/responses/") ||
			strings.HasPrefix(path, "/backend-api/codex/responses/")
	}
}

// ParseStreamUsage 解析流式响应中的使用量
// 遍历 SSE 数据行，提取 usage 信息，取最大值
func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	var inMax int64
	var outMax int64
	var found bool
	// 按行分割流式响应
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		// 跳过非 data 行
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
		// 递归提取 usage
		in, out, ok := extractClaudeUsage(payload)
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

// extractClaudeUsage 从任意 JSON 结构中递归提取 usage 信息
// 查找 input_tokens 和 output_tokens 字段
func extractClaudeUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 检查是否有 usage 字段
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["input_tokens"])
				out := int64Value(usage["output_tokens"])
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		// 递归搜索子元素
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

// ParseCacheUsage 解析缓存相关 Token（缓存创建和缓存读取）
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractClaudeCacheUsage(payload)
}

// AccountCapability 返回 Claude 账号能力描述
// 支持 OAuth 和 API Key 两种认证模式
func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "Claude",
		Notice:             "Claude OAuth 与 Claude API Key 是两套认证体系，建议分别建账号。",
		DefaultBaseURL:     "https://api.anthropic.com",
		BaseURLPlaceholder: "https://api.anthropic.com",
		DefaultAuthMode:    "oauth",
		AuthModes: []provider.AccountAuthMode{
			{Value: "oauth", Label: "OAuth"},
			{Value: "api_key", Label: "API Key"},
		},
		APIKeyField: provider.CapabilityField{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "textarea",
			Required:    true,
			Placeholder: "sk-ant-...",
			Storage:     "credentials",
		},
	}
}

// DefaultOAuthRedirectURI 返回默认的 OAuth 回调地址
func (p *Provider) DefaultOAuthRedirectURI(map[string]string) string {
	return redirectURI
}

// BuildOAuthAuthorizationURL 构建 Claude OAuth 授权 URL
// 使用 PKCE (S256) 流程，包含 code_challenge
func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	return fmt.Sprintf("%s?code=true&client_id=%s&response_type=code&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		authorizeURL,
		url.QueryEscape(p.clientID),
		url.QueryEscape(input.RedirectURI),
		strings.ReplaceAll(url.QueryEscape(oauthScopeValue), "%20", "+"), // 用 + 替代 %20
		url.QueryEscape(provider.PKCEChallenge(input.CodeVerifier)),
		url.QueryEscape(input.State),
	), nil
}

// ExchangeOAuthCode 使用授权码换取 Token
// 通过 JSON 请求发送授权码和 PKCE code_verifier
func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	payload := map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     p.clientID,
		"code":          input.Code,
		"redirect_uri":  input.RedirectURI,
		"code_verifier": input.CodeVerifier, // PKCE 验证
	}
	body, err := provider.JSONRequest(ctx, client, tokenURL, payload)
	if err != nil {
		return nil, err
	}
	// 解析响应，提取 Token 和账号信息
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Account      struct {
			UUID         string `json:"uuid"`
			EmailAddress string `json:"email_address"`
		} `json:"account"`
		Organization struct {
			UUID string `json:"uuid"`
		} `json:"organization"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	// 构建凭证
	cred := &provider.AccountCredentials{
		AccessToken:    result.AccessToken,
		RefreshToken:   result.RefreshToken,
		ClientID:       p.clientID,
		TokenURL:       tokenURL,
		RedirectURI:    input.RedirectURI,
		Email:          result.Account.EmailAddress, // 邮箱
		OrganizationID: result.Organization.UUID,    // 组织 ID
		AccountID:      result.Account.UUID,         // 账号 ID
	}
	// 设置过期时间
	if result.ExpiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

// RefreshOAuthToken 使用刷新令牌获取新的访问令牌
func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials // 复制凭证
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", defaultString(cred.ClientID, p.clientID))
	// 如果有客户端密钥，添加到请求中
	if cred.ClientSecret != "" {
		form.Set("client_secret", cred.ClientSecret)
	}
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	// 更新凭证
	cred.TokenURL = tokenURL
	cred.AccessToken = resp["access_token"]
	// 如果响应包含新的刷新令牌，更新之
	if token := resp["refresh_token"]; token != "" {
		cred.RefreshToken = token
	}
	// 更新过期时间
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return &cred, nil
}

// defaultString 返回第一个非空字符串，否则返回 fallback
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// extractClaudeCacheUsage 递归提取缓存使用量
// 支持 cache_creation_input_tokens/cache_read_input_tokens 和备用字段名
func extractClaudeCacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				// 查找缓存创建 Token
				create := int64Value(usage["cache_creation_input_tokens"])
				if create == 0 {
					create = int64Value(usage["cache_creation_tokens"]) // 备用字段名
				}
				// 查找缓存读取 Token
				read := int64Value(usage["cache_read_input_tokens"])
				if read == 0 {
					read = int64Value(usage["cache_read_tokens"]) // 备用字段名
				}
				if create > 0 || read > 0 {
					return create, read, true
				}
			}
		}
		// 递归搜索子元素
		for _, child := range value {
			if create, read, ok := extractClaudeCacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		for _, child := range value {
			if create, read, ok := extractClaudeCacheUsage(child); ok {
				return create, read, true
			}
		}
	}
	return 0, 0, false
}

// int64Value 安全地将任意类型转换为 int64
// 支持 float64、int64、int 类型
func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n) // JSON 数字默认解析为 float64
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
