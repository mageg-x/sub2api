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

// ApplyRequest 为请求添加Claude所需的认证头
// 使用x-api-key头和anthropic-version头
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	switch strings.TrimSpace(account.AuthType) {
	case "oauth":
		req.Header.Del("x-api-key")
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		req.Header.Set("x-api-key", token)
		req.Header.Del("Authorization")
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
	case "/v1/messages", "/v1/messages/count_tokens":
		return true
	default:
		return false
	}
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
		if len(line) == 0 {
			continue
		}
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		in, out, ok := extractClaudeUsage(payload)
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

// extractClaudeUsage 从任意JSON结构中递归提取usage信息
func extractClaudeUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		if usageAny, ok := value["usage"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["input_tokens"])
				out := int64Value(usage["output_tokens"])
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
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

// ParseCacheUsage 解析缓存相关token
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractClaudeCacheUsage(payload)
}

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

func (p *Provider) DefaultOAuthRedirectURI(map[string]string) string {
	return redirectURI
}

func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	return fmt.Sprintf("%s?code=true&client_id=%s&response_type=code&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
		authorizeURL,
		url.QueryEscape(p.clientID),
		url.QueryEscape(input.RedirectURI),
		strings.ReplaceAll(url.QueryEscape(oauthScopeValue), "%20", "+"),
		url.QueryEscape(provider.PKCEChallenge(input.CodeVerifier)),
		url.QueryEscape(input.State),
	), nil
}

func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	payload := map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     p.clientID,
		"code":          input.Code,
		"redirect_uri":  input.RedirectURI,
		"code_verifier": input.CodeVerifier,
	}
	body, err := provider.JSONRequest(ctx, client, tokenURL, payload)
	if err != nil {
		return nil, err
	}
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
	cred := &provider.AccountCredentials{
		AccessToken:    result.AccessToken,
		RefreshToken:   result.RefreshToken,
		ClientID:       p.clientID,
		TokenURL:       tokenURL,
		RedirectURI:    input.RedirectURI,
		Email:          result.Account.EmailAddress,
		OrganizationID: result.Organization.UUID,
		AccountID:      result.Account.UUID,
	}
	if result.ExpiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	cred := *input.Credentials
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", cred.RefreshToken)
	form.Set("client_id", defaultString(cred.ClientID, p.clientID))
	if cred.ClientSecret != "" {
		form.Set("client_secret", cred.ClientSecret)
	}
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
	return &cred, nil
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func extractClaudeCacheUsage(v any) (int64, int64, bool) {
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
			}
		}
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
