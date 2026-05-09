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

// GatewayRequest 网关请求结构
// 封装了从客户端到上游 API 的完整请求信息
type GatewayRequest struct {
	Method         string      // HTTP 方法
	Provider       string      // 目标 Provider 名称
	PublicPath     string      // 公开路径（客户端请求的路径）
	InternalPath   string      // 内部路径（归一化后的路径）
	RawQuery       string      // 原始查询参数
	Body           []byte      // 请求体
	Model          string      // 请求的模型名称
	Stream         bool        // 是否流式请求
	IncludeUsage   bool        // 是否包含用量信息
	UpstreamStream bool        // 上游是否使用流式响应
	UsageEndpoint  string      // 用量统计端点
	UpstreamMethod string      // 上游请求方法
	LocalStatus    int         // 本地响应状态码
	LocalHeader    http.Header // 本地响应头
	LocalBody      []byte      // 本地响应体
}

// GatewayPathMeta 网关路径元数据
// 描述一个 API 路径的属性，包括归属 Provider、路径映射、HTTP 方法等
type GatewayPathMeta struct {
	Provider      string // 归属的 Provider（空表示通用路径）
	PublicPath    string // 公开路径
	InternalPath  string // 内部路径
	UsageEndpoint string // 用量统计端点
	Method        string // HTTP 方法
	AllowBody     bool   // 是否允许请求体
}

// ParseGatewayPath 解析网关路径，返回路径元数据
// 根据路径格式判断 API 类型（Chat/Responses/Messages/Embeddings/Images/Gemini 等）
// 并确定对应的 Provider、内部路径、HTTP 方法和是否允许请求体
func ParseGatewayPath(path string) (GatewayPathMeta, error) {
	trimmed := strings.TrimSpace(path)
	switch {
	// OpenAI 模型列表接口
	case trimmed == "/v1/models":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodGet}, nil
	// Claude Messages 接口
	case trimmed == "/v1/messages":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	// Claude Token 计数接口
	case trimmed == "/v1/messages/count_tokens":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	// Claude Batches 接口
	case trimmed == "/v1/messages/batches":
		return GatewayPathMeta{Provider: "claude", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/v1/messages/batches/"):
		return GatewayPathMeta{Provider: "claude", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/messages/batches", Method: http.MethodPost, AllowBody: true}, nil
	// OpenAI Chat Completions 接口
	case trimmed == "/v1/chat/completions":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/chat/completions":
		// 无前缀版本，归一化为 /v1/chat/completions
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1/chat/completions", UsageEndpoint: "/v1/chat/completions", Method: http.MethodPost, AllowBody: true}, nil
	// OpenAI Responses 接口
	case trimmed == "/v1/responses":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/v1/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1/responses", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/responses":
		// 无前缀版本，归一化为 /v1/responses
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1/responses", UsageEndpoint: "/v1/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/v1" + trimmed, UsageEndpoint: "/v1/responses", Method: http.MethodPost, AllowBody: true}, nil
	// Codex Responses 接口
	case trimmed == "/backend-api/codex/responses":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: "/backend-api/codex/responses", UsageEndpoint: "/backend-api/codex/responses", AllowBody: true}, nil
	case strings.HasPrefix(trimmed, "/backend-api/codex/responses/"):
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/backend-api/codex/responses", Method: http.MethodPost, AllowBody: true}, nil
	// OpenAI Embeddings 接口
	case trimmed == "/v1/embeddings":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	// OpenAI Images 接口
	case trimmed == "/v1/images/generations":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/v1/images/edits":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/images/generations":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: "/v1/images/generations", UsageEndpoint: "/v1/images/generations", Method: http.MethodPost, AllowBody: true}, nil
	case trimmed == "/images/edits":
		return GatewayPathMeta{Provider: "openai", PublicPath: trimmed, InternalPath: "/v1/images/edits", UsageEndpoint: "/v1/images/edits", Method: http.MethodPost, AllowBody: true}, nil
	// Gemini 原生接口
	case trimmed == "/v1beta/models":
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: trimmed, Method: http.MethodGet}, nil
	case strings.HasPrefix(trimmed, "/v1beta/models/") && !strings.Contains(trimmed, ":"):
		// Gemini 模型详情（不含冒号，如 /v1beta/models/gemini-pro）
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1beta/models", Method: http.MethodGet}, nil
	case strings.HasPrefix(trimmed, "/v1beta/models/"):
		// Gemini 模型操作（含冒号，如 /v1beta/models/gemini-pro:generateContent）
		return GatewayPathMeta{PublicPath: trimmed, InternalPath: trimmed, UsageEndpoint: "/v1beta/models", Method: http.MethodPost, AllowBody: true}, nil
	default:
		return GatewayPathMeta{}, fmt.Errorf("unsupported gateway path %s", trimmed)
	}
}

// ExtractModelAndStream 从请求路径和请求体中提取模型名称和是否流式
// 不同 API 路径使用不同的提取方式：
// - Messages/Chat/Responses/Embeddings/Images：从请求体 JSON 解析
// - Gemini 原生接口：从 URL 路径解析
func ExtractModelAndStream(path string, body []byte) (modelName string, stream bool) {
	switch {
	// Claude Messages 接口：从请求体解析
	case strings.Contains(path, "/messages"):
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if json.Unmarshal(body, &payload) == nil {
			return strings.TrimSpace(payload.Model), payload.Stream
		}
	// OpenAI 兼容接口：从请求体解析
	case strings.Contains(path, "/chat/completions"), strings.Contains(path, "/responses"), strings.Contains(path, "/embeddings"), strings.Contains(path, "/images/"):
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if json.Unmarshal(body, &payload) == nil {
			return strings.TrimSpace(payload.Model), payload.Stream
		}
	// Gemini 原生接口：从 URL 路径解析模型名
	case strings.Contains(path, "/models/"):
		parts := strings.Split(path, "/")
		for i := range parts {
			if parts[i] == "models" && i+1 < len(parts) {
				// 提取模型名（去掉冒号后的操作名，如 gemini-pro:streamGenerateContent → gemini-pro）
				modelName = strings.Split(parts[i+1], ":")[0]
				break
			}
		}
		// 判断是否流式（路径包含 :streamGenerateContent）
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
	APIKey            string `json:"api_key,omitempty"`                 // API 密钥
	AccessToken       string `json:"access_token,omitempty"`            // OAuth 访问令牌
	RefreshToken      string `json:"refresh_token,omitempty"`           // OAuth 刷新令牌
	TokenURL          string `json:"token_url,omitempty"`               // Token 刷新地址
	ClientID          string `json:"client_id,omitempty"`               // OAuth 客户端 ID
	ClientSecret      string `json:"client_secret,omitempty"`           // OAuth 客户端密钥
	RedirectURI       string `json:"redirect_uri,omitempty"`            // OAuth 回调地址
	CodeVerifier      string `json:"code_verifier,omitempty"`           // PKCE code_verifier
	ExpiresAtMS       int64  `json:"expires_at_ms,omitempty"`           // Token 过期时间（毫秒时间戳）
	ProjectID         string `json:"project_id,omitempty"`              // 项目 ID
	OAuthType         string `json:"oauth_type,omitempty"`              // OAuth 类型（code_assist/google_one/ai_studio）
	Email             string `json:"email,omitempty"`                   // 关联邮箱
	OrganizationID    string `json:"organization_id,omitempty"`         // 组织 ID
	AccountID         string `json:"account_id,omitempty"`              // 账号 ID
	BaseURL           string `json:"base_url,omitempty"`                // 自定义基础 URL
	UserAgent         string `json:"user_agent,omitempty"`              // 自定义 User-Agent
	SetupToken        string `json:"setup_token,omitempty"`             // 设置令牌
	TierID            string `json:"tier_id,omitempty"`                 // 套餐层级 ID
	PlanType          string `json:"plan_type,omitempty"`               // 套餐类型
	SubscriptionUntil string `json:"subscription_expires_at,omitempty"` // 订阅到期时间
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

// OAuthAuthorizationInput OAuth 授权输入参数
type OAuthAuthorizationInput struct {
	State        string            // 防 CSRF 的状态参数
	CodeVerifier string            // PKCE code_verifier
	RedirectURI  string            // 回调地址
	Meta         map[string]string // 额外元数据
}

// OAuthExchangeInput OAuth 授权码换 Token 的输入参数
type OAuthExchangeInput struct {
	Code         string            // 授权码
	CodeVerifier string            // PKCE code_verifier
	RedirectURI  string            // 回调地址
	Meta         map[string]string // 额外元数据
}

// OAuthRefreshInput OAuth 刷新 Token 的输入参数
type OAuthRefreshInput struct {
	Account     model.Account       // 账号信息
	Credentials *AccountCredentials // 当前凭证
}

// AccountCapability 账号能力描述
// 定义了前端创建账号时需要的字段和选项
type AccountCapability struct {
	Name               string            `json:"name"`                     // Provider 名称
	Label              string            `json:"label"`                    // 显示标签
	Notice             string            `json:"notice"`                   // 提示信息
	DefaultBaseURL     string            `json:"default_base_url"`         // 默认基础 URL
	BaseURLPlaceholder string            `json:"base_url_placeholder"`     // 基础 URL 输入框占位符
	DefaultAuthMode    string            `json:"default_auth_mode"`        // 默认认证模式
	AuthModes          []AccountAuthMode `json:"auth_modes"`               // 可选认证模式列表
	APIKeyField        CapabilityField   `json:"api_key_field"`            // API Key 输入字段
	AccountFields      []CapabilityField `json:"account_fields,omitempty"` // 账号额外字段
	OAuthFields        []CapabilityField `json:"oauth_fields,omitempty"`   // OAuth 额外字段
}

// AccountAuthMode 认证模式选项
type AccountAuthMode struct {
	Value string `json:"value"` // 模式值（如 api_key、oauth）
	Label string `json:"label"` // 显示标签
}

// CapabilityField 能力字段定义
// 描述前端表单中的一个输入字段
type CapabilityField struct {
	Key          string              `json:"key"`                     // 字段键名
	Label        string              `json:"label"`                   // 显示标签
	Type         string              `json:"type"`                    // 字段类型（text/select 等）
	Required     bool                `json:"required"`                // 是否必填
	Placeholder  string              `json:"placeholder,omitempty"`   // 输入框占位符
	DefaultValue string              `json:"default_value,omitempty"` // 默认值
	Help         string              `json:"help,omitempty"`          // 帮助文本
	Storage      string              `json:"storage,omitempty"`       // 存储位置（credentials/account）
	VisibleWhen  map[string][]string `json:"visible_when,omitempty"`  // 条件显示（当某字段为指定值时显示）
	Options      []CapabilityOption  `json:"options,omitempty"`       // 下拉选项（type=select 时）
}

// CapabilityOption 下拉选项
type CapabilityOption struct {
	Value string `json:"value"` // 选项值
	Label string `json:"label"` // 显示标签
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
// 根据契约（Contract）检查 provider 是否实现了所有声明的必需接口
func ValidateProvider(p Provider) error {
	name := strings.TrimSpace(p.Name())
	// 检查名称非空
	if name == "" {
		return fmt.Errorf("provider name is empty")
	}

	// 必须实现 ContractProvider 接口
	contractProvider, ok := p.(ContractProvider)
	if !ok {
		return fmt.Errorf("provider %s must implement ContractProvider", name)
	}
	contract := contractProvider.Contract()
	// 检查契约名称非空
	if strings.TrimSpace(contract.Name) == "" {
		return fmt.Errorf("provider %s contract name is empty", name)
	}
	// 检查契约名称与 Provider 名称一致
	if strings.TrimSpace(contract.Name) != name {
		return fmt.Errorf("provider %s contract name mismatch: %s", name, contract.Name)
	}

	// 必须实现 AccountCapabilityProvider 接口
	if _, ok := p.(AccountCapabilityProvider); !ok {
		return fmt.Errorf("provider %s must implement AccountCapabilityProvider", name)
	}
	// 根据契约检查可选接口的实现
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
	// 支持 OAuth 的 Provider 必须实现完整的 OAuth 生命周期接口
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

// Names 返回所有已注册 Provider 的名称列表（按字母排序）
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.items))
	for name := range r.items {
		names = append(names, name)
	}
	sort.Strings(names) // 按字母排序，保证输出稳定
	return names
}

// Capabilities 返回所有已注册 Provider 的账号能力列表
// 按名称排序，跳过未实现 AccountCapabilityProvider 的 Provider
func (r *Registry) Capabilities() []AccountCapability {
	names := r.Names()
	items := make([]AccountCapability, 0, len(names))
	for _, name := range names {
		item := r.items[name]
		capabilityProvider, ok := item.(AccountCapabilityProvider)
		if !ok {
			continue // 跳过未实现接口的 Provider
		}
		items = append(items, capabilityProvider.AccountCapability())
	}
	return items
}
