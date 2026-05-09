// Package gemini 实现了 Google Gemini API 的 Provider 适配器。
// 负责将外部请求（OpenAI 兼容格式）转换为 Gemini 原生协议，
// 并将 Gemini 的响应转换回 OpenAI 兼容格式返回给客户端。
// 同时支持 OAuth 认证流程（Code Assist / Google One / AI Studio 三种类型）。
package gemini

import (
	"bytes"         // 用于流式响应体按行分割
	"context"       // 用于 OAuth 交换和刷新的上下文控制
	"encoding/json" // JSON 序列化/反序列化
	"net/http"      // HTTP 请求构建与响应处理
	"net/url"       // URL 解析与查询参数构建
	"strings"       // 字符串工具函数
	"time"          // 用于 OAuth Token 过期时间计算

	"sub2api/server/internal/config"   // 全局配置，包含 Gemini 专属配置
	"sub2api/server/internal/model"    // 数据模型，包含 Account 等
	"sub2api/server/internal/provider" // Provider 接口定义与通用工具
)

// Provider Gemini API Provider 实现
// 持有 Gemini 专属配置，实现 provider 包中定义的多个接口：
// - Provider: 基础 Provider 接口
// - ContractProvider: 能力契约声明
// - AccountCapabilityProvider: 账号能力描述（认证方式、字段定义等）
// - GatewayResponseAdapter: 网关响应适配（Gemini→OpenAI 格式转换）
// - CacheUsageParser: 缓存 Token 用量解析
// - StreamUsageParser: 流式 Token 用量解析
// - OAuthStarter/OAuthExchanger/OAuthRefresher: OAuth 认证三步流程
type Provider struct {
	cfg config.GeminiConfig // Gemini 专属配置（客户端ID、密钥等）
}

// 编译期接口断言：确保 Provider 实现了所有必需的接口。
// 如果缺少某个接口方法，编译时会直接报错，避免运行时才发现问题。
var (
	_ provider.Provider                  = (*Provider)(nil) // 基础 Provider 接口
	_ provider.ContractProvider          = (*Provider)(nil) // 能力契约接口
	_ provider.AccountCapabilityProvider = (*Provider)(nil) // 账号能力接口
	_ provider.GatewayResponseAdapter    = (*Provider)(nil) // 网关响应适配接口
	_ provider.CacheUsageParser          = (*Provider)(nil) // 缓存用量解析接口
	_ provider.StreamUsageParser         = (*Provider)(nil) // 流式用量解析接口
	_ provider.OAuthStarter              = (*Provider)(nil) // OAuth 启动接口
	_ provider.OAuthExchanger            = (*Provider)(nil) // OAuth 代码交换接口
	_ provider.OAuthRefresher            = (*Provider)(nil) // OAuth Token 刷新接口
)

// New 创建 Gemini Provider 实例
// 从全局配置中提取 Gemini 专属配置项，初始化 Provider
func New(cfg config.Config) *Provider {
	return &Provider{cfg: cfg.Gemini}
}

// OAuth 和 API 相关常量定义
const (
	authorizeURL     = "https://accounts.google.com/o/oauth2/v2/auth"                                                                                                   // Google OAuth 授权端点
	tokenURL         = "https://oauth2.googleapis.com/token"                                                                                                            // Google OAuth Token 交换/刷新端点
	codeAssistScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile" // Code Assist 类型 OAuth 所需的权限范围
	aiStudioScopes   = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever"                                   // AI Studio 类型 OAuth 所需的权限范围
	aiRedirectURI    = "http://localhost:1455/auth/callback"                                                                                                            // AI Studio 类型 OAuth 的回调地址
	cliRedirectURI   = "https://codeassist.google.com/authcode"                                                                                                         // Code Assist / Google One 类型 OAuth 的回调地址
	builtinClientID  = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"                                                                       // 内置的 Google OAuth 客户端 ID（用于无自定义客户端时的默认值）
)

// Name 返回 Provider 的唯一标识名称 "gemini"
// 用于在系统中区分不同的 API Provider
func (p *Provider) Name() string {
	return "gemini"
}

// Contract 返回 Gemini Provider 的能力契约
// 声明此 Provider 需要哪些适配器能力以及支持哪些特性：
// - RequiresGatewayResponseAdapter: 需要将 Gemini 原生响应转换为 OpenAI 格式
// - RequiresStreamUsageParser: 需要从流式响应中解析 Token 用量
// - RequiresCacheUsageParser: 需要解析缓存相关的 Token 用量
// - SupportsOAuth: 支持 OAuth 认证流程
func (p *Provider) Contract() provider.Contract {
	return provider.Contract{
		Name:                           p.Name(),
		RequiresGatewayResponseAdapter: true, // Gemini 响应需要格式转换
		RequiresStreamUsageParser:      true, // 流式响应中需要提取用量
		RequiresCacheUsageParser:       true, // 需要解析缓存 Token
		SupportsOAuth:                  true, // 支持 OAuth 认证
	}
}

// NormalizeGateway 对网关请求进行标准化处理
// 主要职责：
//  1. 设置 Provider 名称和上游 HTTP 方法（默认 POST）
//  2. 从请求路径和 Body 中提取模型名称和流式标志
//  3. 对于 OpenAI 兼容路径（/v1/chat/completions、/v1/responses 等），
//     调用 convertGeminiOpenAIRequest 将请求体从 OpenAI 格式转换为 Gemini 原生格式
//  4. 根据是否流式，构建 Gemini API 的内部路径（generateContent 或 streamGenerateContent）
func (p *Provider) NormalizeGateway(account model.Account, cred *provider.AccountCredentials, req provider.GatewayRequest) (provider.GatewayRequest, error) {
	// 标记请求所属的 Provider
	req.Provider = p.Name()
	// 设置上游 HTTP 方法，默认为 POST
	req.UpstreamMethod = req.Method
	if req.UpstreamMethod == "" {
		req.UpstreamMethod = http.MethodPost
	}
	// 清理内部路径的空白字符
	req.InternalPath = strings.TrimSpace(req.InternalPath)
	// 从公共路径和请求体中提取模型名称和是否流式
	req.Model, req.Stream = provider.ExtractModelAndStream(req.PublicPath, req.Body)
	_ = account
	_ = cred
	// 如果内部路径为空，则使用公共路径作为内部路径
	if req.InternalPath == "" {
		req.InternalPath = req.PublicPath
	}
	// 根据标准化后的路径判断是否需要 OpenAI→Gemini 格式转换
	switch normalizeGeminiOpenAIPath(req.PublicPath) {
	case "/v1/chat/completions", "/v1/responses", "/backend-api/codex/responses":
		// 将 OpenAI 格式的请求体转换为 Gemini 原生格式
		converted, modelName, stream, includeUsage, err := convertGeminiOpenAIRequest(req.PublicPath, req.Body)
		if err != nil {
			return req, err
		}
		// 更新转换后的请求体、模型名称、流式标志和是否包含用量信息
		req.Body = converted
		req.Model = modelName
		req.Stream = stream
		req.IncludeUsage = includeUsage
		// 根据是否流式选择 Gemini API 的 action
		action := "generateContent"
		if stream {
			action = "streamGenerateContent"
			req.UpstreamStream = true
		}
		// 构建 Gemini API 的内部路径：/v1beta/models/{model}:{action}
		req.InternalPath = "/v1beta/models/" + req.Model + ":" + action
		req.UpstreamMethod = http.MethodPost
	}
	return req, nil
}

// BuildUpstreamURL 构建 Gemini API 的完整上游请求 URL
// 逻辑说明：
// 1. 使用账号配置的 BaseURL，若为空则默认使用 generativelanguage.googleapis.com
// 2. 拼接路径部分
// 3. 处理查询参数：对于流式请求（streamGenerateContent），自动添加 alt=sse 参数以启用 SSE 流式传输
func (p *Provider) BuildUpstreamURL(account model.Account, path, rawQuery string) string {
	// 获取账号的 BaseURL，去除末尾斜杠
	base := strings.TrimRight(account.BaseURL, "/")
	// 若未配置 BaseURL，使用 Gemini 默认 API 地址
	if base == "" {
		base = "https://generativelanguage.googleapis.com"
	}
	u := base + path
	// 解析原始查询参数
	query, _ := url.ParseQuery(rawQuery)
	// 流式请求需要设置 alt=sse 以获取 SSE 格式的流式响应
	if strings.Contains(path, ":streamGenerateContent") && query.Get("alt") == "" {
		query.Set("alt", "sse")
	}
	// 将查询参数编码后拼接到 URL
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	return u
}

// ApplyRequest 为出站请求添加 Gemini 所需的认证信息
// 两种认证方式：
// - OAuth 模式：在请求头中设置 Authorization: Bearer {token}
// - API Key 模式：在 URL 查询参数中添加 key={token}
func (p *Provider) ApplyRequest(req *http.Request, account model.Account, token string) error {
	// Gemini API 统一使用 JSON 格式
	req.Header.Set("Content-Type", "application/json")
	// OAuth 认证模式：使用 Bearer Token 放在请求头中
	if strings.TrimSpace(account.AuthType) == "oauth" {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
	// API Key 认证模式：移除 Authorization 头，将 key 放在 URL 查询参数中
	req.Header.Del("Authorization")
	query := req.URL.Query()
	query.Set("key", token)
	req.URL.RawQuery = query.Encode()
	return nil
}

// ParseUsage 从非流式响应体中解析 Token 使用量
// 解析 Gemini API 响应中的 usageMetadata 字段，提取输入和输出 Token 数
// 返回值：(输入Token数, 输出Token数)
func (p *Provider) ParseUsage(body []byte) (int64, int64) {
	// 定义与 Gemini 响应 usageMetadata 对应的结构体
	var payload struct {
		UsageMetadata struct {
			PromptTokenCount     int64 `json:"promptTokenCount"`     // 输入 Token 数
			CandidatesTokenCount int64 `json:"candidatesTokenCount"` // 输出 Token 数（候选）
			TotalTokenCount      int64 `json:"totalTokenCount"`      // 总 Token 数
		} `json:"usageMetadata"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0
	}
	in := payload.UsageMetadata.PromptTokenCount
	out := payload.UsageMetadata.CandidatesTokenCount
	// 如果输出 Token 为 0 但有总数，通过总数减去输入来推算输出
	if out == 0 && payload.UsageMetadata.TotalTokenCount > in {
		out = payload.UsageMetadata.TotalTokenCount - in
	}
	return in, out
}

// SupportsPath 判断 Gemini Provider 是否支持给定的 API 路径
// 支持的路径包括：
// - /v1beta/models：模型列表
// - /v1/chat/completions、/chat/completions：OpenAI 兼容的聊天补全
// - /v1/responses、/responses：OpenAI 兼容的 Responses API
// - /backend-api/codex/responses：Codex 后端 API
// - /v1beta/models/、/v1/models/：Gemini 原生模型操作路径
// - /v1/responses/、/responses/：Responses 子路径
func (p *Provider) SupportsPath(path string) bool {
	return path == "/v1beta/models" ||
		path == "/v1/chat/completions" ||
		path == "/chat/completions" ||
		path == "/v1/responses" ||
		path == "/responses" ||
		path == "/backend-api/codex/responses" ||
		strings.HasPrefix(path, "/v1beta/models/") || // Gemini 原生模型路径
		strings.HasPrefix(path, "/v1/models/") || // Gemini v1 模型路径
		strings.HasPrefix(path, "/v1/responses/") || // Responses 子路径
		strings.HasPrefix(path, "/responses/") || // Responses 子路径（无版本前缀）
		strings.HasPrefix(path, "/backend-api/codex/responses/") // Codex 子路径
}

// ParseStreamUsage 从流式响应体中解析 Token 使用量
// 流式响应可能包含多个 SSE data 行，每行可能携带 usageMetadata。
// 此方法遍历所有 SSE 行，提取每行的用量信息，取最大值作为最终结果。
// 返回值：(输入Token数, 输出Token数, 是否找到用量信息)
func (p *Provider) ParseStreamUsage(body []byte) (int64, int64, bool) {
	var inMax int64  // 输入 Token 最大值
	var outMax int64 // 输出 Token 最大值
	var found bool   // 是否找到用量信息
	// 按换行符分割流式响应体
	for _, line := range bytes.Split(body, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		// 只处理 "data:" 开头的 SSE 数据行
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		// 去除 "data:" 前缀并清理空白
		line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		// 跳过空行和 [DONE] 结束标记
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) {
			continue
		}
		// 解析 JSON 数据
		var payload any
		if err := json.Unmarshal(line, &payload); err != nil {
			continue
		}
		// 递归提取 usageMetadata 中的 Token 用量
		in, out, ok := extractGeminiUsage(payload)
		if !ok {
			continue
		}
		found = true
		// 保留各 SSE 行中的最大值（因为流式响应可能多次更新用量）
		if in > inMax {
			inMax = in
		}
		if out > outMax {
			outMax = out
		}
	}
	return inMax, outMax, found
}

// extractGeminiUsage 从任意 JSON 结构中递归提取 usageMetadata 信息
// 因为 Gemini 的流式响应结构可能嵌套较深，所以需要递归查找。
// 返回值：(输入Token数, 输出Token数, 是否找到)
func extractGeminiUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 在当前对象中查找 usageMetadata 字段
		if usageAny, ok := value["usageMetadata"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				in := int64Value(usage["promptTokenCount"])
				out := int64Value(usage["candidatesTokenCount"])
				total := int64Value(usage["totalTokenCount"])
				// 如果输出为 0 但总数大于输入，推算输出值
				if out == 0 && total > in {
					out = total - in
				}
				if in > 0 || out > 0 {
					return in, out, true
				}
			}
		}
		// 当前对象未找到，递归搜索所有子字段
		for _, child := range value {
			if in, out, ok := extractGeminiUsage(child); ok {
				return in, out, true
			}
		}
	case []any:
		// 对数组中的每个元素递归搜索
		for _, child := range value {
			if in, out, ok := extractGeminiUsage(child); ok {
				return in, out, true
			}
		}
	}
	return 0, 0, false
}

// ParseCacheUsage 解析缓存相关的 Token 用量
// 从响应体中提取缓存创建和缓存读取的 Token 数量
// 返回值：(缓存创建Token数, 缓存读取Token数, 是否找到缓存用量)
func (p *Provider) ParseCacheUsage(body []byte) (int64, int64, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, 0, false
	}
	return extractGeminiCacheUsage(payload)
}

// AccountCapability 返回 Gemini Provider 的账号能力描述
// 定义了 Gemini 账号支持的认证方式、表单字段、OAuth 配置等信息
// 这些信息用于前端 UI 渲染账号配置表单
func (p *Provider) AccountCapability() provider.AccountCapability {
	return provider.AccountCapability{
		Name:               p.Name(),
		Label:              "Gemini",
		Notice:             "Gemini 账号区分 OAuth 与 API Key，且不同 OAuth 类型对应不同套餐与回调。",
		DefaultBaseURL:     "https://generativelanguage.googleapis.com", // 默认 API 地址
		BaseURLPlaceholder: "https://generativelanguage.googleapis.com",
		DefaultAuthMode:    "oauth", // 默认使用 OAuth 认证
		// 支持的认证模式列表
		AuthModes: []provider.AccountAuthMode{
			{Value: "oauth", Label: "OAuth"},     // OAuth 认证
			{Value: "api_key", Label: "API Key"}, // API Key 认证
		},
		// API Key 认证时的字段定义
		APIKeyField: provider.CapabilityField{
			Key:         "api_key",
			Label:       "API Key",
			Type:        "textarea", // 多行文本输入
			Required:    true,
			Placeholder: "AIza...",
			Storage:     "credentials", // 存储在凭证中
		},
		// API Key 认证模式下的额外账号字段（套餐选择）
		AccountFields: []provider.CapabilityField{
			{
				Key:          "tier_id",
				Label:        "Tier",
				Type:         "select",       // 下拉选择
				DefaultValue: "gcp_standard", // 默认套餐
				Storage:      "credentials",
				Options: []provider.CapabilityOption{
					{Value: "gcp_standard", Label: "GCP Standard"},       // GCP 标准套餐
					{Value: "gcp_enterprise", Label: "GCP Enterprise"},   // GCP 企业套餐
					{Value: "google_one_free", Label: "Google One Free"}, // Google One 免费版
					{Value: "google_ai_pro", Label: "Google AI Pro"},     // Google AI Pro 版
					{Value: "google_ai_ultra", Label: "Google AI Ultra"}, // Google AI Ultra 版
					{Value: "aistudio_free", Label: "AI Studio Free"},    // AI Studio 免费版
					{Value: "aistudio_paid", Label: "AI Studio Paid"},    // AI Studio 付费版
				},
			},
		},
		// OAuth 认证模式下的字段定义
		OAuthFields: []provider.CapabilityField{
			{
				Key:          "oauth_type",
				Label:        "OAuth Type",
				Type:         "select", // 下拉选择 OAuth 类型
				Required:     true,
				DefaultValue: "code_assist", // 默认 Code Assist 类型
				Storage:      "meta",        // 存储在元数据中
				Options: []provider.CapabilityOption{
					{Value: "code_assist", Label: "Code Assist"}, // Code Assist 类型
					{Value: "google_one", Label: "Google One"},   // Google One 类型
					{Value: "ai_studio", Label: "AI Studio"},     // AI Studio 类型
				},
			},
			{
				Key:         "project_id",
				Label:       "Project ID",
				Type:        "text",
				Placeholder: "optional-gcp-project", // 可选的 GCP 项目 ID
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

// DefaultOAuthRedirectURI 返回指定 OAuth 类型的默认回调地址
// 根据 oauth_type 选择对应的 redirect URI
func (p *Provider) DefaultOAuthRedirectURI(meta map[string]string) string {
	_, redirectURI, _ := p.oauthConfig(meta["oauth_type"])
	return redirectURI
}

// BuildOAuthAuthorizationURL 构建 OAuth 授权跳转 URL
// 用户在浏览器中访问此 URL 完成 Google 账号授权
// 流程说明：
// 1. 根据 oauth_type 获取对应的 OAuth 配置（客户端ID、回调地址、权限范围）
// 2. 如果调用方未指定回调地址，使用默认回调地址
// 3. 构建包含 PKCE 挑战码的授权 URL
func (p *Provider) BuildOAuthAuthorizationURL(input provider.OAuthAuthorizationInput) (string, error) {
	// 获取 OAuth 配置：客户端信息、有效回调地址、权限范围
	cfg, effectiveRedirect, scopes := p.oauthConfig(input.Meta["oauth_type"])
	// 使用调用方指定的回调地址，若为空则使用默认值
	redirectURI := input.RedirectURI
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = effectiveRedirect
	}
	// 构建授权 URL 的查询参数
	params := url.Values{}
	params.Set("response_type", "code")                                      // 授权码模式
	params.Set("client_id", cfg.ClientID)                                    // 客户端 ID
	params.Set("redirect_uri", redirectURI)                                  // 回调地址
	params.Set("scope", scopes)                                              // 权限范围
	params.Set("state", input.State)                                         // 防 CSRF 的状态参数
	params.Set("code_challenge", provider.PKCEChallenge(input.CodeVerifier)) // PKCE 挑战码
	params.Set("code_challenge_method", "S256")                              // PKCE 使用 S256 算法
	// 如果提供了 GCP 项目 ID，添加到参数中
	if projectID := strings.TrimSpace(input.Meta["project_id"]); projectID != "" {
		params.Set("project_id", projectID)
	}
	return authorizeURL + "?" + params.Encode(), nil
}

// ExchangeOAuthCode 使用授权码交换 OAuth Token
// 在用户完成 Google 授权后，用回调返回的授权码向 Google Token 端点换取 Access Token 和 Refresh Token
// 返回包含完整凭证信息的 AccountCredentials 结构体
func (p *Provider) ExchangeOAuthCode(ctx context.Context, client *http.Client, input provider.OAuthExchangeInput) (*provider.AccountCredentials, error) {
	// 获取 OAuth 配置
	cfg, _, _ := p.oauthConfig(input.Meta["oauth_type"])
	// 构建表单形式的请求参数
	form := url.Values{}
	form.Set("grant_type", "authorization_code") // 授权码模式
	form.Set("client_id", cfg.ClientID)
	// 如果有客户端密钥，添加到请求中
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	form.Set("code", input.Code)             // 授权码
	form.Set("redirect_uri", input.RedirectURI) // 回调地址
	form.Set("code_verifier", input.CodeVerifier) // PKCE 验证码
	// 向 Google Token 端点发送请求
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	// 构建凭证结构体，保存所有 OAuth 相关信息
	cred := &provider.AccountCredentials{
		AccessToken:  resp["access_token"],   // 访问令牌
		RefreshToken: resp["refresh_token"],  // 刷新令牌
		ClientID:     cfg.ClientID,           // 客户端 ID
		ClientSecret: cfg.ClientSecret,       // 客户端密钥
		TokenURL:     tokenURL,               // Token 端点
		RedirectURI:  input.RedirectURI,      // 回调地址
		OAuthType:    input.Meta["oauth_type"], // OAuth 类型
		ProjectID:    input.Meta["project_id"], // GCP 项目 ID
		TierID:       input.Meta["tier_id"],    // 套餐 ID
	}
	// 计算并设置 Token 过期时间（毫秒级时间戳）
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

// RefreshOAuthToken 使用 Refresh Token 刷新 OAuth 凭证
// 当 Access Token 过期时，使用此方法获取新的 Access Token
// 同时更新凭证中的其他信息（如客户端配置、回调地址等）
func (p *Provider) RefreshOAuthToken(ctx context.Context, client *http.Client, input provider.OAuthRefreshInput) (*provider.AccountCredentials, error) {
	// 复制现有凭证，避免修改原始数据
	cred := *input.Credentials
	// 获取 OAuth 配置
	cfg, redirectURI, scopes := p.oauthConfig(cred.OAuthType)
	// 构建刷新请求的表单参数
	form := url.Values{}
	form.Set("grant_type", "refresh_token") // 刷新令牌模式
	form.Set("client_id", cfg.ClientID)
	// 如果有客户端密钥，添加到请求中
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	form.Set("refresh_token", cred.RefreshToken) // 刷新令牌
	form.Set("scope", scopes)                    // 权限范围
	// 向 Google Token 端点发送刷新请求
	resp, err := provider.FormRequest(ctx, client, tokenURL, form)
	if err != nil {
		return nil, err
	}
	// 更新凭证中的配置信息
	cred.TokenURL = tokenURL
	cred.RedirectURI = defaultString(cred.RedirectURI, redirectURI) // 优先使用已有回调地址
	cred.ClientID = cfg.ClientID
	cred.ClientSecret = cfg.ClientSecret
	cred.AccessToken = resp["access_token"]  // 更新访问令牌
	// 如果响应中包含新的刷新令牌，更新之（某些 OAuth 实现会轮换刷新令牌）
	if token := resp["refresh_token"]; token != "" {
		cred.RefreshToken = token
	}
	// 计算并更新 Token 过期时间
	if expiresIn := provider.ParseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return &cred, nil
}

// oauthConfig 根据 OAuth 类型返回对应的 OAuth 配置
// 返回值：(Gemini配置, 回调地址, 权限范围)
// 逻辑说明：
// 1. 优先使用用户自定义的 ClientID/ClientSecret
// 2. 若未配置自定义客户端，使用内置客户端 ID 和密钥
// 3. 根据 oauth_type 选择对应的回调地址和权限范围
func (p *Provider) oauthConfig(oauthType string) (config.GeminiConfig, string, string) {
	cfg := p.cfg
	// 构建有效的配置，去除空白字符
	effective := config.GeminiConfig{
		ClientID:            strings.TrimSpace(cfg.ClientID),
		ClientSecret:        strings.TrimSpace(cfg.ClientSecret),
		BuiltinClientSecret: strings.TrimSpace(cfg.BuiltinClientSecret),
	}
	// 默认 OAuth 类型为 code_assist
	oauthType = strings.TrimSpace(oauthType)
	if oauthType == "" {
		oauthType = "code_assist"
	}
	// 如果未配置自定义客户端，使用内置客户端
	isBuiltin := false
	if effective.ClientID == "" && effective.ClientSecret == "" {
		effective.ClientID = builtinClientID
		effective.ClientSecret = effective.BuiltinClientSecret
		isBuiltin = true
	}
	// 根据 OAuth 类型设置默认回调地址和权限范围
	redirectURI := aiRedirectURI  // 默认使用 AI Studio 的本地回调地址
	scopes := codeAssistScopes    // 默认使用 Code Assist 的权限范围
	switch oauthType {
	case "ai_studio":
		// AI Studio 类型：自定义客户端使用 AI Studio 权限范围，内置客户端保持默认
		if !isBuiltin {
			scopes = aiStudioScopes
		}
	case "google_one", "code_assist":
		// Google One 和 Code Assist 类型使用 CLI 回调地址
		redirectURI = cliRedirectURI
	default:
		// 未知类型默认使用 CLI 回调地址
		redirectURI = cliRedirectURI
	}
	// 内置客户端统一使用 CLI 回调地址
	if isBuiltin {
		redirectURI = cliRedirectURI
	}
	return effective, redirectURI, scopes
}

// defaultString 字符串默认值工具函数
// 如果 value 为空白字符串，返回 fallback，否则返回 value
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// extractGeminiCacheUsage 从任意 JSON 结构中递归提取缓存相关的 Token 用量
// 返回值：(缓存创建Token数, 缓存读取Token数, 是否找到缓存用量)
// 注意：Gemini API 的缓存字段名可能因版本不同而变化，
// 因此同时检查 cachedContentTokenCount/cacheCreationTokenCount 和
// cacheReadTokenCount/cachedInputTokenCount 两组字段名
func extractGeminiCacheUsage(v any) (int64, int64, bool) {
	switch value := v.(type) {
	case map[string]any:
		// 在当前对象中查找 usageMetadata 字段
		if usageAny, ok := value["usageMetadata"]; ok {
			if usage, ok := usageAny.(map[string]any); ok {
				// 提取缓存创建 Token 数（兼容两种字段名）
				create := int64Value(usage["cachedContentTokenCount"])
				if create == 0 {
					create = int64Value(usage["cacheCreationTokenCount"])
				}
				// 提取缓存读取 Token 数（兼容两种字段名）
				read := int64Value(usage["cacheReadTokenCount"])
				if read == 0 {
					read = int64Value(usage["cachedInputTokenCount"])
				}
				if create > 0 || read > 0 {
					return create, read, true
				}
			}
		}
		// 当前对象未找到，递归搜索所有子字段
		for _, child := range value {
			if create, read, ok := extractGeminiCacheUsage(child); ok {
				return create, read, true
			}
		}
	case []any:
		// 对数组中的每个元素递归搜索
		for _, child := range value {
			if create, read, ok := extractGeminiCacheUsage(child); ok {
				return create, read, true
			}
		}
	}
	return 0, 0, false
}

// int64Value 安全地将任意类型转换为 int64
// 支持 float64、int64、int 三种常见 JSON 反序列化后的数值类型
// 其他类型返回 0
func int64Value(v any) int64 {
	switch n := v.(type) {
	case float64: // JSON 数字反序列化后最常见的类型
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
