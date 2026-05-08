package antigravity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// Provider Antigravity Provider实现
// 支持Antigravity内部API（基于Gemini的修改版）
type Provider struct {
	clientSecret string
}

// New 创建Antigravity Provider实例
func New(cfg config.Config) *Provider {
	return &Provider{clientSecret: cfg.Antigravity.ClientSecret}
}

const (
	authorizeURL     = "https://accounts.google.com/o/oauth2/v2/auth"
	tokenURL         = "https://oauth2.googleapis.com/token"
	userInfoURL      = "https://www.googleapis.com/oauth2/v2/userinfo"
	redirectURI      = "http://localhost:8085/callback"
	oauthScopes      = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/cclog https://www.googleapis.com/auth/experimentsandconfigs"
	clientID         = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
	defaultUserAgent = "antigravity/1.21.9 windows/amd64"
)

// Name 返回Provider名称
func (p *Provider) Name() string {
	return "antigravity"
}

// BuildUpstreamURL 构建Antigravity API的完整URL
// 使用cloudcode-pa.googleapis.com作为默认端点
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

// ApplyRequest 为请求添加Antigravity所需的认证头
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(account.AuthType) == "oauth" {
		req.Header.Set("Authorization", "Bearer "+token)
	} else {
		req.Header.Del("Authorization")
		query := req.URL.Query()
		query.Set("key", token)
		req.URL.RawQuery = query.Encode()
	}
	req.Header.Set("User-Agent", "antigravity/1.21.9 windows/amd64")
	return nil
}

// ParseUsage 从响应体解析token使用量
// Antigravity的响应结构与Gemini略有不同，使用response.usageMetadata
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
	// 如果输出token为0但有总数，计算输出
	if out == 0 && payload.Response.UsageMetadata.TotalTokenCount > in {
		out = payload.Response.UsageMetadata.TotalTokenCount - in
	}
	return in, out
}

// SupportsPath 判断Antigravity Provider支持的API路径
func (p *Provider) SupportsPath(path string) bool {
	return strings.HasPrefix(path, "/v1internal:")
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

// extractAntigravityUsage 从任意JSON结构中递归提取usageMetadata信息
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

// ParseCacheUsage 解析缓存相关token
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractAntigravityCacheUsage(payload)
}

func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "Antigravity",
		Notice:             "Antigravity 既支持 OAuth，也支持 query key 形式的上游兼容接入。",
		DefaultBaseURL:     "https://cloudcode-pa.googleapis.com",
		BaseURLPlaceholder: "https://cloudcode-pa.googleapis.com",
		DefaultAuthMode:    "oauth",
		AuthModes: []provider.AccountAuthMode{
			{Value: "oauth", Label: "OAuth"},
			{Value: "api_key", Label: "Upstream API Key"},
		},
		APIKeyField: provider.CapabilityField{
			Key:         "api_key",
			Label:       "Upstream API Key",
			Type:        "textarea",
			Required:    true,
			Placeholder: "upstream key",
			Storage:     "credentials",
		},
	}
}

func (p *Provider) DefaultOAuthRedirectURI(map[string]string) string {
	return redirectURI
}

func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("scope", oauthScopes)
	params.Set("state", input.State)
	params.Set("code_challenge", provider.PKCEChallenge(input.CodeVerifier))
	params.Set("code_challenge_method", "S256")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	return authorizeURL + "?" + params.Encode(), nil
}

func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", input.Code)
	form.Set("redirect_uri", redirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", input.CodeVerifier)
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	cred := &provider.AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     clientID,
		ClientSecret: p.clientSecret,
		TokenURL:     tokenURL,
		RedirectURI:  redirectURI,
		UserAgent:    defaultUserAgent,
	}
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
		if userResp, doErr := client.Do(req); doErr == nil {
			defer userResp.Body.Close()
			if body, readErr := io.ReadAll(userResp.Body); readErr == nil {
				var info struct {
					Email string `json:"email"`
				}
				if json.Unmarshal(body, &info) == nil {
					cred.Email = info.Email
				}
			}
		}
	}
	return cred, nil
}

func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("refresh_token", cred.RefreshToken)
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
	cred.ClientID = clientID
	cred.ClientSecret = p.clientSecret
	cred.RedirectURI = redirectURI
	cred.UserAgent = defaultUserAgent
	return &cred, nil
}

func extractAntigravityCacheUsage(v any) (int64, int64, bool) {
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
			if create, read, ok := extractAntigravityCacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		for _, child := range value {
			if create, read, ok := extractAntigravityCacheUsage(child); ok {
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
