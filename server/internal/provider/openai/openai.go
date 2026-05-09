package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// Provider OpenAI API Provider实现
// 支持OpenAI兼容的API接口
type Provider struct {
	clientID string
}

// 编译期断言：OpenAI provider 必须持续满足核心契约。
var (
	_ provider.Provider                  = (*Provider)(nil)
	_ provider.ContractProvider          = (*Provider)(nil)
	_ provider.AccountCapabilityProvider = (*Provider)(nil)
	_ provider.CacheUsageParser          = (*Provider)(nil)
	_ provider.StreamUsageParser         = (*Provider)(nil)
	_ provider.OAuthStarter              = (*Provider)(nil)
	_ provider.OAuthExchanger            = (*Provider)(nil)
	_ provider.OAuthRefresher            = (*Provider)(nil)
)

// New 创建OpenAI Provider实例
func New(cfg config.Config) *Provider {
	return &Provider{clientID: cfg.OpenAI.ClientID}
}

const (
	authorizeURL    = "https://auth.openai.com/oauth/authorize"
	tokenURL        = "https://auth.openai.com/oauth/token"
	defaultRedirect = "http://localhost:1455/auth/callback"
	scopes          = "openid profile email offline_access"
	refreshScopes   = "openid profile email"
	chatGPTBaseURL  = "https://chatgpt.com"
	platformBaseURL = "https://api.openai.com"
)

// Name 返回Provider名称
func (p *Provider) Name() string {
	return "openai"
}

// Contract 返回 OpenAI provider 的能力契约。
// OpenAI 原生就是公开协议主实现，因此不需要插件内额外响应协议回写。
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: false,
		RequiresStreamUsageParser:      true,
		RequiresCacheUsageParser:       true,
		SupportsOAuth:                  true,
	}
}

// NormalizeGateway 归一化网关请求
// 设置 Provider 名称、上游方法、内部路径、模型名和流式标志
// 对于 OAuth 认证的账号，将 /v1/responses 路径重写为 /backend-api/codex/responses
func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	req.Provider = p.Name()
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost // 默认 POST 方法
	}
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	// 从请求路径和请求体中提取模型名和流式标志
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	// OAuth 认证账号：将 /v1/responses 重写为 /backend-api/codex/responses
	if strings.TrimSpace(account.AuthType) == "oauth" && strings.HasPrefix(req.InternalPath, "/v1/responses") {
		req.InternalPath = strings.Replace(req.InternalPath, "/v1/responses", "/backend-api/codex/responses", 1)
	}
	if strings.TrimSpace(account.AuthType) == "oauth" && req.InternalPath == "/backend-api/codex/responses" {
		_ = cred // 保留 cred 引用，后续可能需要
	}
	// 模型列表接口使用 GET 方法
	if req.InternalPath == "/v1/models" {
		req.UpstreamMethod = http.MethodGet
	}
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath // 兜底：使用公开路径
	}
	return req, nil
}

// BuildUpstreamURL 构建OpenAI API的完整URL
// 如果账号配置了自定义BaseURL则使用，否则使用默认的api.openai.com
// OAuth 认证账号使用 chatgpt.com 作为基础 URL
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	// 根据认证类型选择基础 URL
	if strings.TrimSpace(account.AuthType) == "oauth" {
		base = chatGPTBaseURL // OAuth 使用 chatgpt.com
	} else if base == "" {
		base = platformBaseURL // 默认使用 api.openai.com
	}
	// Responses 接口需要特殊处理基础 URL
	if strings.Contains(path, "/responses") {
		base = buildResponsesBaseURL(base)
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
	if strings.TrimSpace(account.AuthType) == "oauth" {
		req.Host = "chatgpt.com"
	}
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
	case "/v1/chat/completions", "/v1/responses", "/v1/embeddings", "/v1/models", "/backend-api/codex/responses", "/v1/images/generations", "/v1/images/edits":
		return true
	default:
		return strings.HasPrefix(path, "/v1/responses/") || strings.HasPrefix(path, "/backend-api/codex/responses/")
	}
}

// buildResponsesBaseURL 构建 Responses 接口的基础 URL
// Responses 接口的 URL 结构与 Chat Completions 不同，需要去掉 /v1 后缀
func buildResponsesBaseURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	// 已经是 codex 路径，直接返回
	if strings.HasSuffix(normalized, "/backend-api/codex") {
		return normalized
	}
	// 去掉多余的 /responses 后缀
	if strings.HasSuffix(normalized, "/backend-api/codex/responses") {
		return strings.TrimSuffix(normalized, "/responses")
	}
	if strings.HasSuffix(normalized, "/responses") {
		return strings.TrimSuffix(normalized, "/responses")
	}
	// 去掉 /v1 后缀，Responses 接口不需要 /v1 前缀
	if strings.HasSuffix(normalized, "/v1") {
		return strings.TrimSuffix(normalized, "/v1")
	}
	return normalized
}

// ParseStreamUsage 解析流式响应中的使用量
// 遍历流式数据块，找出 usage 信息，取最大值
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
		// 跳过空行和结束标记
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) {
			continue
		}
		// 解析 JSON
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		// 递归提取使用量
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

// extractOpenAIUsage 从任意 JSON 结构中递归提取 usage 信息
// 支持 prompt_tokens/completion_tokens 和 input_tokens/output_tokens 两种字段名
func extractOpenAIUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 检查是否有 usage 字段
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				// 优先使用 prompt_tokens/completion_tokens（旧版字段名）
				in := int64Value(usage["prompt_tokens"])
				out := int64Value(usage["completion_tokens"])
				// 备用 input_tokens/output_tokens（新版字段名）
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
		// 递归检查数组元素
		for _, child := range value {
			if in, out, ok := extractOpenAIUsage(child); ok {
				return in, out, true
			}
		}
	}
	return 0, 0, false
}

// ParseCacheUsage 解析缓存相关 Token（缓存创建和缓存读取）
// 递归搜索响应中的 cache_creation_input_tokens 和 cache_read_input_tokens
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractOpenAICacheUsage(payload)
}

// AccountCapability 返回 OpenAI 账号能力描述
// 支持 OAuth 和 API Key 两种认证模式
func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "OpenAI",
		Notice:             "OAuth 账号用于官方授权；API Key 账号适合兼容 OpenAI 协议的上游。",
		DefaultBaseURL:     "https://api.openai.com",
		BaseURLPlaceholder: "https://api.openai.com",
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
			Placeholder: "sk-...",
			Storage:     "credentials",
		},
	}
}

// DefaultOAuthRedirectURI 返回默认的 OAuth 回调地址
func (p *Provider) DefaultOAuthRedirectURI(map[string]string) string {
	return defaultRedirect
}

// BuildOAuthAuthorizationURL 构建 OAuth 授权 URL
// 使用 PKCE (S256) 流程，包含 code_challenge
func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", input.RedirectURI)
	params.Set("scope", scopes)
	params.Set("state", input.State)
	// PKCE 参数
	params.Set("code_challenge", provider.PKCEChallenge(input.CodeVerifier))
	params.Set("code_challenge_method", "S256")
	// OpenAI 特有参数
	params.Set("id_token_add_organizations", "true") // 在 ID Token 中包含组织信息
	params.Set("codex_cli_simplified_flow", "true")  // 使用简化流程
	return authorizeURL + "?" + params.Encode(), nil
}

// ExchangeOAuthCode 使用授权码换取 Token
// 通过表单请求发送授权码和 PKCE code_verifier
func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", p.clientID)
	form.Set("code", input.Code)
	form.Set("redirect_uri", input.RedirectURI)
	form.Set("code_verifier", input.CodeVerifier) // PKCE 验证
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	// 构建凭证
	cred := &provider.AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     p.clientID,
		TokenURL:     tokenURL,
		RedirectURI:  input.RedirectURI,
	}
	// 设置过期时间
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	// 从 ID Token 中提取用户信息
	populateIDToken(cred, resp["id_token"])
	return cred, nil
}

// RefreshOAuthToken 使用刷新令牌获取新的访问令牌
func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials // 复制凭证
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", defaultString(cred.ClientID, p.clientID))
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("scope", refreshScopes) // 刷新时使用精简的 scope
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
	// 从 ID Token 中提取用户信息
	populateIDToken(&cred, resp["id_token"])
	return &cred, nil
}

// populateIDToken 从 ID Token (JWT) 中提取用户信息
// 解析 JWT 的 payload 部分，提取邮箱、账号 ID、套餐类型和组织 ID
func populateIDToken(cred *provider.AccountCredentials, idToken string) {
	if idToken == "" {
		return
	}
	// JWT 格式：header.payload.signature
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return
	}
	payload := parts[1]
	// Base64URL 补齐填充
	switch len(payload) % 4 {
	case 2:
		payload += "=="
	case 3:
		payload += "="
	}
	raw, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return
	}
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return
	}
	// 提取邮箱
	if email, _ := claims["email"].(string); email != "" {
		cred.Email = email
	}
	// 提取 OpenAI 认证信息
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if id, _ := authClaims["chatgpt_account_id"].(string); id != "" {
			cred.AccountID = id // ChatGPT 账号 ID
		}
		if plan, _ := authClaims["chatgpt_plan_type"].(string); plan != "" {
			cred.PlanType = plan // 套餐类型
		}
		if oid, _ := authClaims["poid"].(string); oid != "" {
			cred.OrganizationID = oid // 组织 ID
		}
	}
}

// defaultString 返回第一个非空字符串，否则返回 fallback
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// extractOpenAICacheUsage 递归提取缓存使用量
// 支持 cache_creation_input_tokens/cache_read_input_tokens 和 input_tokens_details 两种格式
func extractOpenAICacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				// 直接在 usage 层级查找缓存字段
				create := int64Value(usage["cache_creation_input_tokens"])
				if create == 0 {
					create = int64Value(usage["cache_creation_tokens"]) // 备用字段名
				}
				read := int64Value(usage["cache_read_input_tokens"])
				if read == 0 {
					read = int64Value(usage["cache_read_tokens"]) // 备用字段名
				}
				if create > 0 || read > 0 {
					return create, read, true
				}
				// 在 input_tokens_details 中查找缓存字段
				if details, ok := usage["input_tokens_details"].(map[string]any); ok {
					create = int64Value(details["cache_creation_input_tokens"])
					read = int64Value(details["cache_read_input_tokens"])
					if create > 0 || read > 0 {
						return create, read, true
					}
				}
			}
		}
		// 递归搜索子元素
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
