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

func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	req.Provider = p.Name()
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost
	}
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	if strings.TrimSpace(account.AuthType) == "oauth" && strings.HasPrefix(req.InternalPath, "/v1/responses") {
		req.InternalPath = strings.Replace(req.InternalPath, "/v1/responses", "/backend-api/codex/responses", 1)
	}
	if strings.TrimSpace(account.AuthType) == "oauth" && req.InternalPath == "/backend-api/codex/responses" {
		_ = cred
	}
	if req.InternalPath == "/v1/models" {
		req.UpstreamMethod = http.MethodGet
	}
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath
	}
	return req, nil
}

// BuildUpstreamURL 构建OpenAI API的完整URL
// 如果账号配置了自定义BaseURL则使用，否则使用默认的api.openai.com
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	base := strings.TrimRight(strings.TrimSpace(account.BaseURL), "/")
	if strings.TrimSpace(account.AuthType) == "oauth" {
		base = chatGPTBaseURL
	} else if base == "" {
		base = platformBaseURL
	}
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

func buildResponsesBaseURL(base string) string {
	normalized := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(normalized, "/backend-api/codex") {
		return normalized
	}
	if strings.HasSuffix(normalized, "/backend-api/codex/responses") {
		return strings.TrimSuffix(normalized, "/responses")
	}
	if strings.HasSuffix(normalized, "/responses") {
		return strings.TrimSuffix(normalized, "/responses")
	}
	if strings.HasSuffix(normalized, "/v1") {
		return strings.TrimSuffix(normalized, "/v1")
	}
	return normalized
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

func (p *Provider) DefaultOAuthRedirectURI(map[string]string) string {
	return defaultRedirect
}

func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", p.clientID)
	params.Set("redirect_uri", input.RedirectURI)
	params.Set("scope", scopes)
	params.Set("state", input.State)
	params.Set("code_challenge", provider.PKCEChallenge(input.CodeVerifier))
	params.Set("code_challenge_method", "S256")
	params.Set("id_token_add_organizations", "true")
	params.Set("codex_cli_simplified_flow", "true")
	return authorizeURL + "?" + params.Encode(), nil
}

func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", p.clientID)
	form.Set("code", input.Code)
	form.Set("redirect_uri", input.RedirectURI)
	form.Set("code_verifier", input.CodeVerifier)
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	cred := &provider.AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     p.clientID,
		TokenURL:     tokenURL,
		RedirectURI:  input.RedirectURI,
	}
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	populateIDToken(cred, resp["id_token"])
	return cred, nil
}

func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", defaultString(cred.ClientID, p.clientID))
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("scope", refreshScopes)
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	cred.TokenURL = tokenURL
	cred.AccessToken = resp["access_token"]
	if token := resp["refresh_token"]; token != "" {
		cred.RefreshToken = token
	}
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	populateIDToken(&cred, resp["id_token"])
	return &cred, nil
}

func populateIDToken(cred *provider.AccountCredentials, idToken string) {
	if idToken == "" {
		return
	}
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return
	}
	payload := parts[1]
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
	if email, _ := claims["email"].(string); email != "" {
		cred.Email = email
	}
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if id, _ := authClaims["chatgpt_account_id"].(string); id != "" {
			cred.AccountID = id
		}
		if plan, _ := authClaims["chatgpt_plan_type"].(string); plan != "" {
			cred.PlanType = plan
		}
		if oid, _ := authClaims["poid"].(string); oid != "" {
			cred.OrganizationID = oid
		}
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
