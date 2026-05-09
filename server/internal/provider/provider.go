package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"sub2api/server/internal/model"
)

// Provider AI服务Provider接口
// 所有AI服务提供商需要实现此接口
type Provider interface {
	// Name 返回Provider名称
	Name() string
	// NormalizeGateway 将外部公开网关请求归一化为该 provider 的网关语义
	NormalizeGateway(account model.Account, cred *AccountCredentials, req GatewayRequest) (GatewayRequest, error)
	// BuildUpstreamURL 构建上游API的完整URL
	BuildUpstreamURL(account model.Account, path, rawQuery string) string
	// ApplyRequest 对请求进行必要的处理（如添加认证头）
	ApplyRequest(req *http.Request, account model.Account, token string) error
	// ParseUsage 从响应体解析token使用量
	ParseUsage(body []byte) (int64, int64)
	// SupportsPath 判断该Provider是否支持指定的API路径
	SupportsPath(path string) bool
}

// Contract 描述一个 provider 的能力契约。
// 这不是给前端展示的元数据，而是给后端注册阶段做强校验的约束。
//
// 设计目标：
// 1. 新增 provider 时，先把“需要承担哪些职责”写清楚。
// 2. 启动时统一校验，避免出现“代码能编译，但能力缺半截”的半成品 provider。
// 3. 让后续新增 provider 时直接照模板实现，不再靠口头约定。
type Contract struct {
	// Name 必须与 Provider.Name() 一致。
	Name string
	// RequiresGatewayResponseAdapter 表示该 provider 需要在插件内完成
	// OpenAI 公开协议 <-> 上游原生协议 的响应/流式回写适配。
	RequiresGatewayResponseAdapter bool
	// RequiresStreamUsageParser 表示该 provider 必须能从流式返回中提取 usage。
	RequiresStreamUsageParser bool
	// RequiresCacheUsageParser 表示该 provider 必须能提取缓存命中/写入 token。
	RequiresCacheUsageParser bool
	// SupportsOAuth 表示该 provider 需要实现完整 OAuth 生命周期。
	SupportsOAuth bool
}

// ContractProvider 要求每个 provider 显式声明自己的能力契约。
// 这样后续新增 provider 时，“哪些能力必须实现”就不会再模糊。
type ContractProvider interface {
	Contract() Contract
}

// GatewayResponseAdapter 允许 provider 在插件内完成公共协议响应回写。
// core 仅负责调用，不感知具体协议细节。
type GatewayResponseAdapter interface {
	AdaptGatewayResponse(req GatewayRequest, resp *http.Response, body []byte) (*http.Response, []byte, error)
	AdaptGatewayStream(req GatewayRequest, resp *http.Response) (*http.Response, error)
}

type GatewayRequest struct {
	Method         string
	Provider       string
	PublicPath     string
	InternalPath   string
	RawQuery       string
	Body           []byte
	Model          string
	Stream         bool
	IncludeUsage   bool
	UpstreamStream bool
	UsageEndpoint  string
	UpstreamMethod string
	LocalStatus    int
	LocalHeader    http.Header
	LocalBody      []byte
}

type GatewayPathMeta struct {
	Provider      string
	PublicPath    string
	InternalPath  string
	UsageEndpoint string
	Method        string
	AllowBody     bool
}

func ParseGatewayPath(path string) (GatewayPathMeta, error) {
	trimmed := strings.TrimSpace(path)
	switch {
	case trimmed == "/v1/models":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodGet}, nil
	case trimmed == "/v1/messages":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/messages/count_tokens":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/messages/batches":
		return GatewayPathMeta{Provider: "claude", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/v1/messages/batches/"):
		return GatewayPathMeta{Provider: "claude", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/messages/batches", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/chat/completions":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/chat/completions":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1/chat/completions", UsageEndpoint: "/v1/chat/completions", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/responses":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/v1/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/responses", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/responses":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1/responses", UsageEndpoint: "/v1/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1" + trimmed, UsageEndpoint: "/v1/responses", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/backend-api/codex/responses":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/backend-api/codex/responses", UsageEndpoint: "/backend-api/codex/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/backend-api/codex/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/backend-api/codex/responses", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/embeddings":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/images/generations":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/images/edits":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/images/generations":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: "/v1/images/generations", UsageEndpoint: "/v1/images/generations", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/images/edits":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: "/v1/images/edits", UsageEndpoint: "/v1/images/edits", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1beta/models":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodGet}, nil
	case strings.HasPrefix(trimmed, "/v1beta/models/") && !strings.Contains(trimmed, ":"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1beta/models", Method: http.MethodGet}, nil
	case strings.HasPrefix(trimmed, "/v1beta/models/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1beta/models", Method: http.MethodPost, AllowBody: true}, nil
	default:
		return GatewayPathMeta{}, fmt.Errorf("unsupported gateway path %s", trimmed)
	}
}

func ExtractModelAndStream(path string, body []byte) (modelName string, stream bool) {
	switch {
	case strings.Contains(path, "/messages"):
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if json.Unmarshal(body, &payload) == nil {
			return strings.TrimSpace(payload.Model), payload.Stream
		}
	case strings.Contains(path, "/chat/completions"), strings.Contains(path, "/responses"), strings.Contains(path, "/embeddings"), strings.Contains(path, "/images/"):
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if json.Unmarshal(body, &payload) == nil {
			return strings.TrimSpace(payload.Model), payload.Stream
		}
	case strings.Contains(path, "/models/"):
		parts := strings.Split(path, "/")
		for i := range parts {
			if parts[i] == "models" && i+1 < len(parts) {
				modelName = strings.Split(parts[i+1], ":")[0]
				break
			}
		}
		stream = strings.Contains(path, ":streamGenerateContent")
	}
	return modelName, stream
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

// Register 注册一个Provider。
// 注册阶段会做强校验，避免把不完整的 provider 放进运行时。
func (r *Registry) Register(p Provider) error {
	if err := ValidateProvider(p); err != nil {
		return err
	}
	r.items[p.Name()] = p
	return nil
}

// Get 根据名称获取Provider
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", name)
	}
	return p, nil
}

// ValidateProvider 校验 provider 是否满足本项目约定的最小实现规范。
func ValidateProvider(p Provider) error {
	name := strings.TrimSpace(p.Name())
	if name == "" {
		return fmt.Errorf("provider name is empty")
	}

	contractProvider, ok := p.(ContractProvider)
	if !ok {
		return fmt.Errorf("provider %s must implement ContractProvider", name)
	}
	contract := contractProvider.Contract()
	if strings.TrimSpace(contract.Name) == "" {
		return fmt.Errorf("provider %s contract name is empty", name)
	}
	if strings.TrimSpace(contract.Name) != name {
		return fmt.Errorf("provider %s contract name mismatch: %s", name, contract.Name)
	}

	if _, ok := p.(AccountCapabilityProvider); !ok {
		return fmt.Errorf("provider %s must implement AccountCapabilityProvider", name)
	}
	if contract.RequiresGatewayResponseAdapter {
		if _, ok := p.(GatewayResponseAdapter); !ok {
			return fmt.Errorf("provider %s must implement GatewayResponseAdapter", name)
		}
	}
	if contract.RequiresStreamUsageParser {
		if _, ok := p.(StreamUsageParser); !ok {
			return fmt.Errorf("provider %s must implement StreamUsageParser", name)
		}
	}
	if contract.RequiresCacheUsageParser {
		if _, ok := p.(CacheUsageParser); !ok {
			return fmt.Errorf("provider %s must implement CacheUsageParser", name)
		}
	}
	if contract.SupportsOAuth {
		if _, ok := p.(OAuthStarter); !ok {
			return fmt.Errorf("provider %s must implement OAuthStarter", name)
		}
		if _, ok := p.(OAuthExchanger); !ok {
			return fmt.Errorf("provider %s must implement OAuthExchanger", name)
		}
		if _, ok := p.(OAuthRefresher); !ok {
			return fmt.Errorf("provider %s must implement OAuthRefresher", name)
		}
	}
	return nil
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
