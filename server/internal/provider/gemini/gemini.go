package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// Provider Gemini API Provider实现
// 支持Google Gemini API接口
type Provider struct {
	cfg config.GeminiConfig
}

// New 创建Gemini Provider实例
func New(cfg config.Config) *Provider {
	return &Provider{cfg: cfg.Gemini}
}

const (
	authorizeURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL         = "https://oauth2.googleapis.com/token"
	codeAssistScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	aiStudioScopes   = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever"
	aiRedirectURI    = "http://localhost:1455/auth/callback"
	cliRedirectURI   = "https://codeassist.google.com/authcode"
	builtinClientID  = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
)

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
	if strings.TrimSpace(account.AuthType) == "oauth" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	req.Header.Del("Authorization")
	query := req.URL.Query()
	query.Set("key", token)
	req.URL.RawQuery = query.Encode()
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

func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "Gemini",
		Notice:             "Gemini 账号区分 OAuth 与 API Key，且不同 OAuth 类型对应不同套餐与回调。",
		DefaultBaseURL:     "https://generativelanguage.googleapis.com",
		BaseURLPlaceholder: "https://generativelanguage.googleapis.com",
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
			Placeholder: "AIza...",
			Storage:     "credentials",
		},
		AccountFields: []provider.CapabilityField{
			{
				Key:          "tier_id",
				Label:        "Tier",
				Type:         "select",
				DefaultValue: "gcp_standard",
				Storage:      "credentials",
				Options: []provider.CapabilityOption{
					{Value: "gcp_standard", Label: "GCP Standard"},
					{Value: "gcp_enterprise", Label: "GCP Enterprise"},
					{Value: "google_one_free", Label: "Google One Free"},
					{Value: "google_ai_pro", Label: "Google AI Pro"},
					{Value: "google_ai_ultra", Label: "Google AI Ultra"},
					{Value: "aistudio_free", Label: "AI Studio Free"},
					{Value: "aistudio_paid", Label: "AI Studio Paid"},
				},
			},
		},
		OAuthFields: []provider.CapabilityField{
			{
				Key:          "oauth_type",
				Label:        "OAuth Type",
				Type:         "select",
				Required:     true,
				DefaultValue: "code_assist",
				Storage:      "meta",
				Options: []provider.CapabilityOption{
					{Value: "code_assist", Label: "Code Assist"},
					{Value: "google_one", Label: "Google One"},
					{Value: "ai_studio", Label: "AI Studio"},
				},
			},
			{
				Key:         "project_id",
				Label:       "Project ID",
				Type:        "text",
				Placeholder: "optional-gcp-project",
				Storage:     "meta",
			},
			{
				Key:          "tier_id",
				Label:        "Tier",
				Type:         "select",
				Required:     true,
				DefaultValue: "gcp_standard",
				Storage:      "meta",
				Options: []provider.CapabilityOption{
					{Value: "gcp_standard", Label: "GCP Standard"},
					{Value: "gcp_enterprise", Label: "GCP Enterprise"},
					{Value: "google_one_free", Label: "Google One Free"},
					{Value: "google_ai_pro", Label: "Google AI Pro"},
					{Value: "google_ai_ultra", Label: "Google AI Ultra"},
					{Value: "aistudio_free", Label: "AI Studio Free"},
					{Value: "aistudio_paid", Label: "AI Studio Paid"},
				},
			},
		},
	}
}

func (p *Provider) DefaultOAuthRedirectURI(meta map[string]string) string {
	_, redirectURI, _ := p.oauthConfig(meta["oauth_type"])
	return redirectURI
}

func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	cfg, effectiveRedirect, scopes := p.oauthConfig(input.Meta["oauth_type"])
	redirectURI := input.RedirectURI
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = effectiveRedirect
	}
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", scopes)
	params.Set("state", input.State)
	params.Set("code_challenge", provider.PKCEChallenge(input.CodeVerifier))
	params.Set("code_challenge_method", "S256")
	if projectID := strings.TrimSpace(input.Meta["project_id"]); projectID != "" {
		params.Set("project_id", projectID)
	}
	return authorizeURL + "?" + params.Encode(), nil
}

func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	cfg, _, _ := p.oauthConfig(input.Meta["oauth_type"])
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
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
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     tokenURL,
		RedirectURI:  input.RedirectURI,
		OAuthType:    input.Meta["oauth_type"],
		ProjectID:    input.Meta["project_id"],
		TierID:       input.Meta["tier_id"],
	}
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials
	cfg, redirectURI, scopes := p.oauthConfig(cred.OAuthType)
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", cfg.ClientID)
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("scope", scopes)
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	cred.TokenURL = tokenURL
	cred.RedirectURI = defaultString(cred.RedirectURI, redirectURI)
	cred.ClientID = cfg.ClientID
	cred.ClientSecret = cfg.ClientSecret
	cred.AccessToken = resp["access_token"]
	if token := resp["refresh_token"]; token != "" {
		cred.RefreshToken = token
	}
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return &cred, nil
}

func (p *Provider) oauthConfig(oauthType string) (config.GeminiConfig, string, string) {
	cfg := p.cfg
	effective := config.GeminiConfig{
		ClientID:            strings.TrimSpace(cfg.ClientID),
		ClientSecret:        strings.TrimSpace(cfg.ClientSecret),
		BuiltinClientSecret: strings.TrimSpace(cfg.BuiltinClientSecret),
	}
	oauthType = strings.TrimSpace(oauthType)
	if oauthType == "" {
		oauthType = "code_assist"
	}
	isBuiltin := false
	if effective.ClientID == "" && effective.ClientSecret == "" {
		effective.ClientID = builtinClientID
		effective.ClientSecret = effective.BuiltinClientSecret
		isBuiltin = true
	}
	redirectURI := aiRedirectURI
	scopes := codeAssistScopes
	switch oauthType {
	case "ai_studio":
		if !isBuiltin {
			scopes = aiStudioScopes
		}
	case "google_one", "code_assist":
		redirectURI = cliRedirectURI
	default:
		redirectURI = cliRedirectURI
	}
	if isBuiltin {
		redirectURI = cliRedirectURI
	}
	return effective, redirectURI, scopes
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
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
