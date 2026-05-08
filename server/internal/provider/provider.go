package provider

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"sub2api/server/internal/model"
)

// Provider AI服务Provider接口
// 所有AI服务提供商需要实现此接口
type Provider interface {
	// Name 返回Provider名称
	Name() string
	// BuildUpstreamURL 构建上游API的完整URL
	BuildUpstreamURL(account model.Account, path, rawQuery string) string
	// ApplyRequest 对请求进行必要的处理（如添加认证头）
	ApplyRequest(req *http.Request, account model.Account, token string) error
	// ParseUsage 从响应体解析token使用量
	ParseUsage(body []byte) (int64, int64)
	// SupportsPath 判断该Provider是否支持指定的API路径
	SupportsPath(path string) bool
}

// CacheUsageParser 缓存使用量解析器接口
// 用于解析缓存创建和读取 token
type CacheUsageParser interface {
	ParseCacheUsage(body []byte) (int64, int64, bool)
}

// StreamUsageParser 流式响应使用量解析器接口
// 对于支持流式输出的Provider，需要实现此接口来解析流式响应中的使用量
type StreamUsageParser interface {
	// ParseStreamUsage 解析流式响应中的使用量
	// 返回: 输入token数, 输出token数, 是否成功解析
	ParseStreamUsage(body []byte) (int64, int64, bool)
}

// AccountCredentials 账号凭证
// 供 service 持久化，也供 provider 认证流程读写。
type AccountCredentials struct {
	APIKey            string `json:"api_key,omitempty"`
	AccessToken       string `json:"access_token,omitempty"`
	RefreshToken      string `json:"refresh_token,omitempty"`
	TokenURL          string `json:"token_url,omitempty"`
	ClientID          string `json:"client_id,omitempty"`
	ClientSecret      string `json:"client_secret,omitempty"`
	RedirectURI       string `json:"redirect_uri,omitempty"`
	CodeVerifier      string `json:"code_verifier,omitempty"`
	ExpiresAtMS       int64  `json:"expires_at_ms,omitempty"`
	ProjectID         string `json:"project_id,omitempty"`
	OAuthType         string `json:"oauth_type,omitempty"`
	Email             string `json:"email,omitempty"`
	OrganizationID    string `json:"organization_id,omitempty"`
	AccountID         string `json:"account_id,omitempty"`
	BaseURL           string `json:"base_url,omitempty"`
	UserAgent         string `json:"user_agent,omitempty"`
	SetupToken        string `json:"setup_token,omitempty"`
	TierID            string `json:"tier_id,omitempty"`
	PlanType          string `json:"plan_type,omitempty"`
	SubscriptionUntil string `json:"subscription_expires_at,omitempty"`
}

// AccountCapabilityProvider 暴露账号创建能力
type AccountCapabilityProvider interface {
	AccountCapability() AccountCapability
}

// OAuthStarter 负责 OAuth 授权入口
type OAuthStarter interface {
	DefaultOAuthRedirectURI(meta map[string]string) string
	BuildOAuthAuthorizationURL(input OAuthAuthorizationInput) (string, error)
}

// OAuthExchanger 负责 OAuth 授权码换 token
type OAuthExchanger interface {
	ExchangeOAuthCode(ctx context.Context, client *http.Client, input OAuthExchangeInput) (*AccountCredentials, error)
}

// OAuthRefresher 负责刷新 OAuth token
type OAuthRefresher interface {
	RefreshOAuthToken(ctx context.Context, client *http.Client, input OAuthRefreshInput) (*AccountCredentials, error)
}

type OAuthAuthorizationInput struct {
	State        string
	CodeVerifier string
	RedirectURI  string
	Meta         map[string]string
}

type OAuthExchangeInput struct {
	Code         string
	CodeVerifier string
	RedirectURI  string
	Meta         map[string]string
}

type OAuthRefreshInput struct {
	Account     model.Account
	Credentials *AccountCredentials
}

type AccountCapability struct {
	Name               string            `json:"name"`
	Label              string            `json:"label"`
	Notice             string            `json:"notice"`
	DefaultBaseURL     string            `json:"default_base_url"`
	BaseURLPlaceholder string            `json:"base_url_placeholder"`
	DefaultAuthMode    string            `json:"default_auth_mode"`
	AuthModes          []AccountAuthMode `json:"auth_modes"`
	APIKeyField        CapabilityField   `json:"api_key_field"`
	AccountFields      []CapabilityField `json:"account_fields,omitempty"`
	OAuthFields        []CapabilityField `json:"oauth_fields,omitempty"`
}

type AccountAuthMode struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type CapabilityField struct {
	Key          string              `json:"key"`
	Label        string              `json:"label"`
	Type         string              `json:"type"`
	Required     bool                `json:"required"`
	Placeholder  string              `json:"placeholder,omitempty"`
	DefaultValue string              `json:"default_value,omitempty"`
	Help         string              `json:"help,omitempty"`
	Storage      string              `json:"storage,omitempty"`
	VisibleWhen  map[string][]string `json:"visible_when,omitempty"`
	Options      []CapabilityOption  `json:"options,omitempty"`
}

type CapabilityOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// Registry Provider注册表
// 用于管理所有可用的AI服务Provider
type Registry struct {
	items map[string]Provider // 以Provider名称为键的映射
}

// NewRegistry 创建新的Provider注册表
func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

// Register 注册一个Provider
func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

// Get 根据名称获取Provider
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", name)
	}
	return p, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.items))
	for name := range r.items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Capabilities() []AccountCapability {
	names := r.Names()
	items := make([]AccountCapability, 0, len(names))
	for _, name := range names {
		item := r.items[name]
		capabilityProvider, ok := item.(AccountCapabilityProvider)
		if !ok {
			continue
		}
		items = append(items, capabilityProvider.AccountCapability())
	}
	return items
}
