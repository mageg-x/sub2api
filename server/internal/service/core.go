// Package service 提供核心业务逻辑服务
// 包含用户管理、API密钥管理、AI账号管理、OAuth认证、代理转发、支付处理等功能
package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/cryptoext"
	"sub2api/server/internal/model"
	"sub2api/server/internal/payment"
	"sub2api/server/internal/provider"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// OpenAI OAuth授权URL
	openAIAuthorizeURL = "https://auth.openai.com/oauth/authorize"
	// OpenAI OAuth令牌URL
	openAITokenURL = "https://auth.openai.com/oauth/token"
	// OpenAI默认回调地址
	openAIDefaultRedirect = "http://localhost:1455/auth/callback"
	// OpenAI授权范围
	openAIScopes = "openid profile email offline_access"
	// OpenAI刷新令牌范围
	openAIRefreshScopes = "openid profile email"
	// Claude OAuth授权URL
	claudeAuthorizeURL = "https://claude.ai/oauth/authorize"
	// Claude OAuth令牌URL
	claudeTokenURL = "https://platform.claude.com/v1/oauth/token"
	// Claude回调地址
	claudeRedirectURI = "https://platform.claude.com/oauth/code/callback"
	// Claude授权范围
	claudeScopeOAuth = "org:create_api_key user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
	// Gemini OAuth授权URL
	geminiAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	// Gemini OAuth令牌URL
	geminiTokenURL = "https://oauth2.googleapis.com/token"
	// Gemini Code Assist授权范围
	geminiCodeAssistScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	// Gemini AI Studio授权范围
	geminiAIStudioScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever"
	// Gemini AI重定向URI
	geminiAIRedirectURI = "http://localhost:1455/auth/callback"
	// Gemini CLI重定向URI
	geminiCLIRedirectURI = "https://codeassist.google.com/authcode"
	// Gemini内置客户端ID
	geminiBuiltinClientID = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	// Antigravity OAuth授权URL
	antigravityAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	// Antigravity OAuth令牌URL
	antigravityTokenURL = "https://oauth2.googleapis.com/token"
	// Antigravity用户信息URL
	antigravityUserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	// Antigravity回调地址
	antigravityRedirectURI = "http://localhost:8085/callback"
	// Antigravity授权范围
	antigravityScopes = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/cclog https://www.googleapis.com/auth/experimentsandconfigs"
	// Antigravity客户端ID
	antigravityClientID = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
)

// Core 核心服务结构体
// 包含所有业务逻辑：用户管理、API密钥管理、AI账号管理、OAuth认证、代理转发、支付处理等
type Core struct {
	cfg            config.Config                  // 应用配置
	db             *gorm.DB                       // 数据库连接
	providers      *provider.Registry             // AI Provider注册表
	payments       *payment.Registry              // 支付Provider注册表
	httpClient     *http.Client                   // HTTP客户端
	accountLoads   sync.Map                       // account_id -> *atomic.Int64
	refreshMu      sync.Map                       // OAuth刷新锁（每个账号一个）
	cacheMu        sync.RWMutex                   // 缓存的读写锁
	cacheItems     map[string]cachedProxyResponse // 代理响应缓存
	registerMu     sync.Mutex                     // 注册速率限制锁
	registerIPs    map[string]int64               // IP -> 最近注册时间戳(ms)
	accountCache   map[string][]model.Account     // provider -> 活跃账号缓存
	accountCacheMu sync.RWMutex                   // 账号缓存读写锁
	priceCache     map[string]model.ModelPrice    // "provider:model" -> 价格缓存
	priceCacheMu   sync.RWMutex                   // 价格缓存读写锁
	pendingKeyMu   sync.Mutex                     // 批量key更新锁
	pendingKeys    map[uint64]int64               // key_id -> last_used_at_ms 待刷新
	pendingUserMu  sync.Mutex                     // 批量user更新锁
	pendingUsers   map[uint64]int64               // user_id -> last_used_at_ms 待刷新
	usageLogMu     sync.Mutex
	usageLogQueue  []model.UsageLog
	dashboardMu    sync.RWMutex
	dashboardCache map[uint64]cachedUserDashboard
	statsMu        sync.RWMutex
	statsCache     map[string]cachedStats
	tokenSignKey   []byte
}

type cachedUserDashboard struct {
	data      *UserDashboardData
	expiresAt time.Time
}

type cachedStats struct {
	data      map[string]any
	expiresAt time.Time
}

// cachedProxyResponse 缓存的代理响应
type cachedProxyResponse struct {
	StatusCode int         // HTTP状态码
	Header     http.Header // 响应头
	Body       []byte      // 响应体
	ExpiresAt  time.Time   // 过期时间
}

// usageTrackingReadCloser 用于跟踪API使用量的ReadCloser包装器
// 在读取响应体的同时记录数据，用于后续使用量统计
type usageTrackingReadCloser struct {
	src          io.ReadCloser // 原始的ReadCloser
	buf          bytes.Buffer  // 缓冲区，用于存储已读取的数据
	finalizeOnce sync.Once     // 确保finalize只执行一次
	finalize     func([]byte)  // 最终回调函数，用于处理已读取的数据
}

func (r *usageTrackingReadCloser) Read(p []byte) (int, error) {
	// Read 从源读取数据，同时复制到缓冲区
	n, err := r.src.Read(p)
	if n > 0 {
		_, _ = r.buf.Write(p[:n])
	}
	// 读取完成后调用finish
	if err != nil {
		r.finish()
	}
	return n, err
}

func (r *usageTrackingReadCloser) Close() error {
	// Close 关闭源并调用finish
	err := r.src.Close()
	r.finish()
	return err
}

func (r *usageTrackingReadCloser) finish() {
	// finish 确保finalize只执行一次
	r.finalizeOnce.Do(func() {
		if r.finalize != nil {
			r.finalize(append([]byte(nil), r.buf.Bytes()...))
		}
	})
}

// userTokenClaims 用户令牌声明结构
type userTokenClaims struct {
	UserID       uint64 `json:"user_id"`       // 用户ID
	TokenVersion int64  `json:"token_version"` // 令牌版本
	ExpiresAtMS  int64  `json:"expires_at_ms"` // 过期时间（毫秒）
	Kind         string `json:"kind"`          // 令牌类型
}

// AccountCredentials AI账号凭证结构
// 存储各种OAuth和API密钥信息
type AccountCredentials struct {
	APIKey            string `json:"api_key,omitempty"`                 // API密钥
	AccessToken       string `json:"access_token,omitempty"`            // 访问令牌
	RefreshToken      string `json:"refresh_token,omitempty"`           // 刷新令牌
	TokenURL          string `json:"token_url,omitempty"`               // 令牌URL
	ClientID          string `json:"client_id,omitempty"`               // 客户端ID
	ClientSecret      string `json:"client_secret,omitempty"`           // 客户端密钥
	RedirectURI       string `json:"redirect_uri,omitempty"`            // 回调URI
	CodeVerifier      string `json:"code_verifier,omitempty"`           // PKCE代码验证器
	ExpiresAtMS       int64  `json:"expires_at_ms,omitempty"`           // 过期时间（毫秒）
	ProjectID         string `json:"project_id,omitempty"`              // 项目ID
	OAuthType         string `json:"oauth_type,omitempty"`              // OAuth类型
	Email             string `json:"email,omitempty"`                   // 邮箱
	OrganizationID    string `json:"organization_id,omitempty"`         // 组织ID
	AccountID         string `json:"account_id,omitempty"`              // 账号ID
	BaseURL           string `json:"base_url,omitempty"`                // 基础URL
	UserAgent         string `json:"user_agent,omitempty"`              // 用户代理
	SetupToken        string `json:"setup_token,omitempty"`             // 设置令牌
	TierID            string `json:"tier_id,omitempty"`                 // 套餐ID
	PlanType          string `json:"plan_type,omitempty"`               // 套餐类型
	SubscriptionUntil string `json:"subscription_expires_at,omitempty"` // 订阅截止时间
}

// ProxyAuth 代理认证结构
// 用于API代理请求的身份验证
type ProxyAuth struct {
	User           model.User   // 用户信息
	APIKey         model.APIKey // API密钥信息
	UserAllowedSet map[string]struct{}
}

// UserAuth 用户认证结构
// 包含用户信息和认证令牌
type UserAuth struct {
	User         model.User `json:"user"`          // 用户信息
	AccessToken  string     `json:"access_token"`  // 访问令牌
	RefreshToken string     `json:"refresh_token"` // 刷新令牌
	TokenType    string     `json:"token_type"`    // 令牌类型
	ExpiresIn    int64      `json:"expires_in"`    // 过期时间（秒）
}

type CreateUserInput struct {
	Email         string          `json:"email"`
	Name          string          `json:"name"`
	Password      string          `json:"password"`
	Role          string          `json:"role"`
	Balance       int64           `json:"balance"`
	RatePercent   int             `json:"rate_percent"`
	AllowedModels []string        `json:"allowed_models"`
	Metadata      json.RawMessage `json:"metadata"`
}

type UpdateUserInput struct {
	Name          string   `json:"name"`
	Status        string   `json:"status"`
	Role          string   `json:"role"`
	Password      string   `json:"password"`
	Balance       *int64   `json:"balance"`
	RatePercent   *int     `json:"rate_percent"`
	AllowedModels []string `json:"allowed_models"`
}

type CreateAPIKeyInput struct {
	UserID      uint64 `json:"user_id"`
	Provider    string `json:"provider"`
	Name        string `json:"name"`
	ExpiresAtMS int64  `json:"expires_at_ms"`
}

type CreateAccountInput struct {
	Provider         string          `json:"provider"`
	Name             string          `json:"name"`
	AuthType         string          `json:"auth_type"`
	BaseURL          string          `json:"base_url"`
	ModelScope       []string        `json:"model_scope"`
	Credentials      json.RawMessage `json:"credentials"`
	Priority         int             `json:"priority"`
	ConcurrencyLimit int             `json:"concurrency_limit"`
	Metadata         json.RawMessage `json:"metadata"`
}

type UpdateAccountInput struct {
	Status           *string `json:"status"`
	Priority         *int    `json:"priority"`
	ConcurrencyLimit *int    `json:"concurrency_limit"`
	CredentialsJSON  *string `json:"credentials_json"`
	BaseURL          *string `json:"base_url"`
	ModelScopeJSON   *string `json:"model_scope_json"`
}

type CreateModelPriceInput struct {
	Provider         string `json:"provider"`
	Model            string `json:"model"`
	InputPrice       int64  `json:"input_price"`
	OutputPrice      int64  `json:"output_price"`
	CacheCreatePrice int64  `json:"cache_create_price"`
	CacheReadPrice   int64  `json:"cache_read_price"`
	Status           string `json:"status"`
}

type CreateAnnouncementInput struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	Status        string `json:"status"`
	PublishedAtMS int64  `json:"published_at_ms"`
}

type CreateCouponInput struct {
	Code        string `json:"code"`
	Kind        string `json:"kind"`
	Amount      int64  `json:"amount"`
	MaxUses     int    `json:"max_uses"`
	ExpiresAtMS int64  `json:"expires_at_ms"`
}

type CreatePaymentOrderInput struct {
	UserID    uint64 `json:"user_id"`
	Amount    int64  `json:"amount"`
	Subject   string `json:"subject"`
	ReturnURL string `json:"return_url"`
}

// RegisterInput 用户注册输入结构
type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	ClientIP string `json:"-"`
}

// LoginInput 用户登录输入结构
type LoginInput struct {
	Email    string `json:"email"`    // 邮箱
	Password string `json:"password"` // 密码
}

// RefreshTokenInput 刷新令牌输入结构
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"` // 刷新令牌
}

// UpdateProfileInput 更新用户资料输入结构
type UpdateProfileInput struct {
	Name string `json:"name"` // 用户名
}

// ChangePasswordInput 修改密码输入结构
type ChangePasswordInput struct {
	OldPassword string `json:"old_password"` // 旧密码
	NewPassword string `json:"new_password"` // 新密码
}

// OAuthStartInput OAuth授权开始输入结构
type OAuthStartInput struct {
	Provider    string `json:"provider"`     // OAuth提供商
	RedirectURI string `json:"redirect_uri"` // 回调URI
	OAuthType   string `json:"oauth_type"`   // OAuth类型
	ProjectID   string `json:"project_id"`   // 项目ID
	TierID      string `json:"tier_id"`      // 套餐ID
}

// OAuthStartResult OAuth授权开始结果结构
type OAuthStartResult struct {
	Provider  string `json:"provider"`   // OAuth提供商
	SessionID string `json:"session_id"` // 会话ID
	State     string `json:"state"`      // 状态
	AuthURL   string `json:"auth_url"`   // 授权URL
}

// OAuthExchangeInput OAuth令牌交换输入结构
type OAuthExchangeInput struct {
	SessionID string `json:"session_id"` // 会话ID
	State     string `json:"state"`      // 状态
	Code      string `json:"code"`       // 授权码
}

// OAuthExchangeResult OAuth令牌交换结果结构
type OAuthExchangeResult struct {
	AccountCredentials // 账号凭证
}

// DashboardData 仪表盘数据结构
type DashboardData struct {
	Users         []model.User         `json:"users"`         // 用户列表
	Accounts      []AccountView        `json:"accounts"`      // 账号列表
	Prices        []model.ModelPrice   `json:"prices"`        // 价格列表
	Orders        []model.PaymentOrder `json:"orders"`        // 订单列表
	Announcements []model.Announcement `json:"announcements"` // 公告列表
	Coupons       []model.Coupon       `json:"coupons"`       // 优惠券列表
	Stats         map[string]any       `json:"stats"`         // 统计数据
}

// UserDashboardData 用户首页仪表盘数据结构
type UserDashboardData struct {
	Balance             int64                `json:"balance"`
	RatePercent         int                  `json:"rate_percent"`
	TotalCost           int64                `json:"total_cost"`
	TotalRecharge       int64                `json:"total_recharge"`
	RequestCount        int64                `json:"request_count"`
	RecentUsageCount    int64                `json:"recent_usage_count"`
	TotalInputTokens    int64                `json:"total_input_tokens"`
	TotalOutputTokens   int64                `json:"total_output_tokens"`
	TotalTokens         int64                `json:"total_tokens"`
	AvgRPM              string               `json:"avg_rpm"`
	AvgTPM              string               `json:"avg_tpm"`
	TopModel            string               `json:"top_model"`
	TopProvider         string               `json:"top_provider"`
	LastUsageTimeMS     int64                `json:"last_usage_time_ms"`
	KeyCount            int64                `json:"key_count"`
	UsageTimeline       []int64              `json:"usage_timeline"`
	CostTimeline        []int64              `json:"cost_timeline"`
	TokenTimeline       []int64              `json:"token_timeline"`
	RecentUsageLogs     []model.UsageLog     `json:"recent_usage_logs"`
	RecentPaymentOrders []model.PaymentOrder `json:"recent_payment_orders"`
}

// AccountView 账号视图结构（包含脱敏后的凭证）
type AccountView struct {
	model.Account
	Credentials map[string]any `json:"credentials,omitempty"` // 凭证信息
}

// New 创建Core核心服务实例
// 参数：
//   - cfg: 应用配置
//   - db: 数据库连接
//   - providers: AI Provider注册表
//   - payments: 支付Provider注册表
func New(cfg config.Config, db *gorm.DB, providers *provider.Registry, payments *payment.Registry) *Core {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   128,
		MaxConnsPerHost:       0,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}
	c := &Core{
		cfg:       cfg,
		db:        db,
		providers: providers,
		payments:  payments,
		tokenSignKey: deriveScopedKey(cfg.AESKey, "user-token-signing"),
		httpClient: &http.Client{
			Timeout:   120 * time.Second,
			Transport: transport,
		},
		cacheItems:     map[string]cachedProxyResponse{},
		registerIPs:    map[string]int64{},
		accountCache:   map[string][]model.Account{},
		priceCache:     map[string]model.ModelPrice{},
		pendingKeys:    map[uint64]int64{},
		pendingUsers:   map[uint64]int64{},
		dashboardCache: map[uint64]cachedUserDashboard{},
		statsCache:     map[string]cachedStats{},
	}
	c.reloadAccountCache()
	c.reloadPriceCache()
	return c
}

// Start 启动核心服务
// 启动后台任务：OAuth刷新循环和指标快照循环
func (c *Core) Start(ctx context.Context) {
	go c.refreshOAuthLoop(ctx)
	go c.snapshotMetricsLoop(ctx)
	go c.cacheCleanupLoop(ctx)
	go c.oauthSessionCleanupLoop(ctx)
	go c.cacheRefreshLoop(ctx)
	go c.dataCleanupLoop(ctx)
	go c.flushKeyUpdatesLoop(ctx)
	go c.flushUserUpdatesLoop(ctx)
	go c.flushUsageLogsLoop(ctx)
}

// CheckAdminToken 检查管理员令牌是否有效
// 参数：
//   - token: 待验证的令牌
//
// 返回：令牌是否有效
func (c *Core) CheckAdminToken(token string) bool {
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.cfg.AdminToken)) == 1
}

// BootstrapAdmin 引导创建管理员账号
// 如果系统中不存在管理员，则创建第一个管理员
// 参数：
//   - name: 管理员名称
//   - email: 管理员邮箱
//   - password: 管理员密码
//
// 返回：用户认证信息和错误
func (c *Core) BootstrapAdmin(name, email, password string) (*UserAuth, error) {
	var count int64
	if err := c.db.Model(&model.User{}).Where("role = ?", "admin").Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, fmt.Errorf("admin already exists")
	}
	return c.createUserAuth(CreateUserInput{
		Email:       email,
		Name:        name,
		Password:    password,
		Role:        "admin",
		RatePercent: 100,
	})
}

// CreateUser 创建用户
// 参数：
//   - in: 创建用户输入参数
//
// 返回：创建的用户和错误
func (c *Core) CreateUser(in CreateUserInput) (*model.User, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email != "" && !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email format")
	}
	allowed, _ := json.Marshal(normalizeStrings(in.AllowedModels))
	salt, hash, err := hashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Email:             email,
		Name:              strings.TrimSpace(in.Name),
		PasswordSalt:      salt,
		PasswordHash:      hash,
		Role:              defaultString(strings.TrimSpace(in.Role), "user"),
		Status:            "active",
		TokenVersion:      1,
		Balance:           in.Balance,
		RatePercent:       defaultInt(in.RatePercent, 100),
		AllowedModelsJSON: string(allowed),
		MetadataJSON:      normalizeJSON(in.Metadata, "{}"),
	}
	return user, c.db.Create(user).Error
}

// ListUsers 获取所有用户列表
// 返回：用户列表和错误
func (c *Core) ListUsers() ([]model.User, error) {
	var items []model.User
	err := c.db.Order("id asc").Limit(500).Find(&items).Error
	return items, err
}

func (c *Core) UpdateUser(id uint64, in UpdateUserInput) (*model.User, error) {
	updates := map[string]any{}
	if name := strings.TrimSpace(in.Name); name != "" {
		updates["name"] = name
	}
	if status := strings.TrimSpace(in.Status); status != "" {
		updates["status"] = status
	}
	if role := strings.TrimSpace(in.Role); role != "" {
		updates["role"] = role
	}
	if password := strings.TrimSpace(in.Password); password != "" {
		salt, hash, err := hashPassword(password)
		if err != nil {
			return nil, err
		}
		updates["password_salt"] = salt
		updates["password_hash"] = hash
	}
	if in.Balance != nil {
		updates["balance"] = *in.Balance
	}
	if in.RatePercent != nil {
		updates["rate_percent"] = *in.RatePercent
	}
	if in.AllowedModels != nil {
		allowed, _ := json.Marshal(normalizeStrings(in.AllowedModels))
		updates["allowed_models_json"] = string(allowed)
	}
	if len(updates) == 0 {
		var user model.User
		if err := c.db.First(&user, id).Error; err != nil {
			return nil, err
		}
		return &user, nil
	}
	updates["updated_at_ms"] = time.Now().UnixMilli()
	if err := c.db.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	var user model.User
	if err := c.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	c.invalidateStats()
	c.invalidateUserDashboard(id)
	return &user, nil
}

// GetUserByID 根据ID获取用户
// 参数：
//   - id: 用户ID
//
// 返回：用户信息和错误
func (c *Core) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := c.db.Where("id = ? AND status = ?", id, "active").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Register 用户注册
// 参数：
//   - in: 注册输入参数
//
// 返回：用户认证信息和错误
const registerCooldownMS int64 = 60_000
const maxRegisterIPEntries = 10000
const maxUsageLogQueueSize = 20000

func (c *Core) Register(in RegisterInput) (*UserAuth, error) {
	if ip := strings.TrimSpace(in.ClientIP); ip != "" {
		c.registerMu.Lock()
		now := time.Now().UnixMilli()
		cutoff := now - registerCooldownMS
		for k, v := range c.registerIPs {
			if v < cutoff {
				delete(c.registerIPs, k)
			}
		}
		if len(c.registerIPs) >= maxRegisterIPEntries {
			oldestIP := ""
			var oldestAt int64
			for k, v := range c.registerIPs {
				if oldestIP == "" || v < oldestAt {
					oldestIP = k
					oldestAt = v
				}
			}
			if oldestIP != "" {
				delete(c.registerIPs, oldestIP)
			}
		}
		if last, ok := c.registerIPs[ip]; ok && now-last < registerCooldownMS {
			c.registerMu.Unlock()
			return nil, fmt.Errorf("registration too frequent, please try again later")
		}
		c.registerIPs[ip] = now
		c.registerMu.Unlock()
	}
	auth, err := c.createUserAuth(CreateUserInput{
		Email:       in.Email,
		Name:        in.Name,
		Password:    in.Password,
		Role:        "user",
		RatePercent: 100,
	})
	if err == nil {
		c.invalidateStats()
	}
	return auth, err
}

func (c *Core) createUserAuth(in CreateUserInput) (*UserAuth, error) {
	user, err := c.CreateUser(in)
	if err != nil {
		return nil, err
	}
	return c.issueUserAuth(user)
}

// Login 用户登录
// 参数：
//   - in: 登录输入参数
//
// 返回：用户认证信息和错误
func (c *Core) Login(in LoginInput) (*UserAuth, error) {
	var user model.User
	if err := c.db.Where("email = ?", strings.TrimSpace(strings.ToLower(in.Email))).First(&user).Error; err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}
	if user.Status != "active" {
		return nil, fmt.Errorf("user is not active")
	}
	if !verifyPassword(user.PasswordSalt, user.PasswordHash, in.Password) {
		return nil, fmt.Errorf("invalid email or password")
	}
	now := time.Now().UnixMilli()
	_ = c.db.Model(&user).Updates(map[string]any{
		"last_login_at_ms": now,
		"updated_at_ms":    now,
	}).Error
	user.LastLoginAtMS = now
	return c.issueUserAuth(&user)
}

// RefreshUserToken 刷新用户令牌
// 参数：
//   - in: 刷新令牌输入参数
//
// 返回：新的用户认证信息和错误
func (c *Core) RefreshUserToken(in RefreshTokenInput) (*UserAuth, error) {
	claims, err := c.parseUserToken(strings.TrimSpace(in.RefreshToken), "refresh")
	if err != nil {
		return nil, err
	}
	user, err := c.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not available")
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, fmt.Errorf("refresh token expired")
	}
	return c.issueUserAuth(user)
}

// LogoutUser 用户登出
// 参数：
//   - refreshToken: 刷新令牌
//
// 返回：错误
func (c *Core) LogoutUser(refreshToken string) error {
	if strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	claims, err := c.parseUserToken(strings.TrimSpace(refreshToken), "refresh")
	if err != nil {
		return nil
	}
	return c.db.Model(&model.User{}).Where("id = ?", claims.UserID).Updates(map[string]any{
		"token_version": gorm.Expr("token_version + 1"),
		"updated_at_ms": time.Now().UnixMilli(),
	}).Error
}

// AuthenticateUserToken 验证用户访问令牌
// 参数：
//   - token: 访问令牌
//
// 返回：用户信息和错误
func (c *Core) AuthenticateUserToken(token string) (*model.User, error) {
	claims, err := c.parseUserToken(strings.TrimSpace(token), "access")
	if err != nil {
		return nil, err
	}
	user, err := c.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not available")
	}
	if user.TokenVersion != claims.TokenVersion {
		return nil, fmt.Errorf("token expired")
	}
	return user, nil
}

// UpdateProfile 更新用户资料
// 参数：
//   - userID: 用户ID
//   - in: 更新资料输入参数
//
// 返回：更新后的用户信息和错误
func (c *Core) UpdateProfile(userID uint64, in UpdateProfileInput) (*model.User, error) {
	updates := map[string]any{
		"updated_at_ms": time.Now().UnixMilli(),
	}
	if name := strings.TrimSpace(in.Name); name != "" {
		updates["name"] = name
	}
	if err := c.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		return nil, err
	}
	return c.GetUserByID(userID)
}

// ChangePassword 修改用户密码
// 参数：
//   - userID: 用户ID
//   - in: 修改密码输入参数
//
// 返回：错误
func (c *Core) ChangePassword(userID uint64, in ChangePasswordInput) error {
	var user model.User
	if err := c.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("user not found")
	}
	if !verifyPassword(user.PasswordSalt, user.PasswordHash, in.OldPassword) {
		return fmt.Errorf("old password is invalid")
	}
	salt, hash, err := hashPassword(in.NewPassword)
	if err != nil {
		return err
	}
	return c.db.Model(&user).Updates(map[string]any{
		"password_salt": salt,
		"password_hash": hash,
		"token_version": gorm.Expr("token_version + 1"),
		"updated_at_ms": time.Now().UnixMilli(),
	}).Error
}

// ListUserAPIKeys 获取用户的所有API密钥
// 参数：
//   - userID: 用户ID
//
// 返回：API密钥列表和错误
func (c *Core) ListUserAPIKeys(userID uint64) ([]model.APIKey, error) {
	var items []model.APIKey
	err := c.db.Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

// CreateUserAPIKey 为用户创建API密钥
// 参数：
//   - userID: 用户ID
//   - provider: 绑定的供应商
//   - name: 密钥名称
//   - expiresAtMS: 过期时间（毫秒）
//
// 返回：创建的API密钥和错误
func (c *Core) CreateUserAPIKey(userID uint64, provider, name string, expiresAtMS int64) (*model.APIKey, error) {
	return c.CreateAPIKey(CreateAPIKeyInput{
		UserID:      userID,
		Provider:    provider,
		Name:        name,
		ExpiresAtMS: expiresAtMS,
	})
}

// ListUserUsage 获取用户的使用记录
// 参数：
//   - userID: 用户ID
//   - limit: 返回记录数量限制
//
// 返回：使用记录列表和错误
func (c *Core) ListUserUsage(userID uint64, limit int) ([]model.UsageLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []model.UsageLog
	err := c.db.Where("user_id = ?", userID).Order("created_at_ms desc").Limit(limit).Find(&items).Error
	return items, err
}

// ListUsage 获取所有使用记录
// 参数：
//   - limit: 返回记录数量限制
//
// 返回：使用记录列表和错误
func (c *Core) ListUsage(limit int) ([]model.UsageLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var items []model.UsageLog
	err := c.db.Order("created_at_ms desc").Limit(limit).Find(&items).Error
	return items, err
}

// ListUserPaymentOrders 获取用户的支付订单列表
// 参数：
//   - userID: 用户ID
//
// 返回：支付订单列表和错误
func (c *Core) ListUserPaymentOrders(userID uint64) ([]model.PaymentOrder, error) {
	var items []model.PaymentOrder
	err := c.db.Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

// UserDashboard 获取用户首页聚合数据
func (c *Core) UserDashboard(userID uint64) (*UserDashboardData, error) {
	if cached, ok := c.getCachedUserDashboard(userID); ok {
		return cached, nil
	}
	user, err := c.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	oneHourAgo := now - int64(time.Hour/time.Millisecond)
	sevenDaysAgo := now - int64(7*24*time.Hour/time.Millisecond)

	var agg struct {
		TotalCost   int64
		TotalInput  int64
		TotalOutput int64
		Count       int64
	}
	c.db.Model(&model.UserUsageDay{}).Where("user_id = ?", userID).Select(
		"COALESCE(SUM(cost),0) as total_cost, COALESCE(SUM(input_tokens),0) as total_input, " +
			"COALESCE(SUM(output_tokens),0) as total_output, COALESCE(SUM(request_count),0) as count",
	).Scan(&agg)

	var recentUsageCount int64
	c.db.Model(&model.UserUsageHour{}).Where("user_id = ? AND bucket_start_ms >= ?", userID, alignHourStart(sevenDaysAgo)).Select(
		"COALESCE(SUM(request_count),0)",
	).Scan(&recentUsageCount)

	var hourAgg struct {
		Count  int64
		Tokens int64
	}
	c.db.Model(&model.UserUsageMinute{}).Where("user_id = ? AND bucket_start_ms >= ?", userID, alignMinuteStart(oneHourAgo)).Select(
		"COALESCE(SUM(request_count),0) as count, COALESCE(SUM(input_tokens+output_tokens),0) as tokens",
	).Scan(&hourAgg)

	var topModelResult struct {
		Model string
		Cnt   int64
	}
	c.db.Model(&model.UserUsageDayDimension{}).Where("user_id = ?", userID).Select(
		"model, COALESCE(SUM(request_count),0) as cnt").Group("model").Order("cnt desc").Limit(1).Scan(&topModelResult)
	topModel := "-"
	if topModelResult.Model != "" {
		topModel = topModelResult.Model
	}

	var topProviderResult struct {
		Provider string
		Cnt      int64
	}
	c.db.Model(&model.UserUsageDayDimension{}).Where("user_id = ?", userID).Select(
		"provider, COALESCE(SUM(request_count),0) as cnt").Group("provider").Order("cnt desc").Limit(1).Scan(&topProviderResult)
	topProvider := "-"
	if topProviderResult.Provider != "" {
		topProvider = topProviderResult.Provider
	}

	var rechargeAgg struct {
		Total int64
	}
	c.db.Model(&model.PaymentOrder{}).Where("user_id = ? AND status = ?", userID, "paid").Select(
		"COALESCE(SUM(amount),0) as total").Scan(&rechargeAgg)

	var keyCount int64
	c.db.Model(&model.APIKey{}).Where("user_id = ?", userID).Count(&keyCount)

	var recentUsage []struct {
		BucketStartMS int64
		DayCost       int64
		DayRequests   int64
		DayTokens     int64
	}
	c.db.Model(&model.UserUsageDay{}).Where("user_id = ? AND bucket_start_ms >= ?", userID, alignDayStart(sevenDaysAgo)).Select(
		"bucket_start_ms, COALESCE(SUM(cost),0) as day_cost, COALESCE(SUM(request_count),0) as day_requests, COALESCE(SUM(input_tokens+output_tokens),0) as day_tokens",
	).Group("bucket_start_ms").Scan(&recentUsage)

	dayBuckets := 7
	dailyRequests := make([]int64, dayBuckets)
	dailyCost := make([]int64, dayBuckets)
	dailyTokens := make([]int64, dayBuckets)
	for _, item := range recentUsage {
		idx := int((alignDayStart(now) - item.BucketStartMS) / int64(24*time.Hour/time.Millisecond))
		if idx >= 0 && idx < dayBuckets {
			dailyRequests[dayBuckets-1-idx] = item.DayRequests
			dailyCost[dayBuckets-1-idx] = item.DayCost
			dailyTokens[dayBuckets-1-idx] = item.DayTokens
		}
	}

	var recentUsageLogs []model.UsageLog
	c.db.Where("user_id = ?", userID).Order("created_at_ms desc").Limit(20).Find(&recentUsageLogs)

	var recentPaymentOrders []model.PaymentOrder
	c.db.Where("user_id = ?", userID).Order("created_at_ms desc").Limit(10).Find(&recentPaymentOrders)

	avgRPM := "0"
	avgTPM := "0"
	if hourAgg.Count > 0 {
		avgRPM = fmt.Sprintf("%.3f", float64(hourAgg.Count)/60.0)
		avgTPM = fmt.Sprintf("%.3f", float64(hourAgg.Tokens)/60.0)
	}

	result := &UserDashboardData{
		Balance:             user.Balance,
		RatePercent:         user.RatePercent,
		TotalCost:           agg.TotalCost,
		TotalRecharge:       rechargeAgg.Total,
		RequestCount:        agg.Count,
		RecentUsageCount:    recentUsageCount,
		TotalInputTokens:    agg.TotalInput,
		TotalOutputTokens:   agg.TotalOutput,
		TotalTokens:         agg.TotalInput + agg.TotalOutput,
		AvgRPM:              avgRPM,
		AvgTPM:              avgTPM,
		TopModel:            topModel,
		TopProvider:         topProvider,
		LastUsageTimeMS:     user.LastUsedAtMS,
		KeyCount:            keyCount,
		UsageTimeline:       dailyRequests,
		CostTimeline:        dailyCost,
		TokenTimeline:       dailyTokens,
		RecentUsageLogs:     recentUsageLogs,
		RecentPaymentOrders: recentPaymentOrders,
	}
	c.setCachedUserDashboard(userID, result)
	return result, nil
}

// GetUserPaymentOrder 获取用户的指定支付订单
// 参数：
//   - userID: 用户ID
//   - orderID: 订单ID
//
// 返回：支付订单和错误
func (c *Core) GetUserPaymentOrder(userID, orderID uint64) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	if err := c.db.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

// RedeemCoupon 兑换优惠券
// 参数：
//   - userID: 用户ID
//   - code: 优惠券代码
//
// 返回：优惠券信息和错误
func (c *Core) RedeemCoupon(userID uint64, code string) (*model.Coupon, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("coupon code is required")
	}
	var coupon model.Coupon
	now := time.Now().UnixMilli()
	var userUpdates map[string]any
	err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ? AND status = ?", code, "active").First(&coupon).Error; err != nil {
			return fmt.Errorf("coupon not found")
		}
		if coupon.ExpiresAtMS > 0 && coupon.ExpiresAtMS < now {
			return fmt.Errorf("coupon expired")
		}
		if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
			return fmt.Errorf("coupon exhausted")
		}
		var usageLog []map[string]any
		_ = json.Unmarshal([]byte(coupon.UsageLogJSON), &usageLog)
		for _, entry := range usageLog {
			if uid, ok := entry["user_id"]; ok {
				if fmt.Sprintf("%v", uid) == fmt.Sprintf("%v", userID) {
					return fmt.Errorf("coupon already redeemed by this user")
				}
			}
		}
		usageLog = append(usageLog, map[string]any{
			"user_id":        userID,
			"redeemed_at_ms": now,
		})
		usageLogRaw, _ := json.Marshal(usageLog)
		if err := tx.Model(&coupon).Updates(map[string]any{
			"used_count":     gorm.Expr("used_count + 1"),
			"updated_at_ms":  now,
			"usage_log_json": string(usageLogRaw),
		}).Error; err != nil {
			return err
		}
		switch strings.TrimSpace(strings.ToLower(coupon.Kind)) {
		case "", "balance":
			userUpdates = map[string]any{
				"balance":       gorm.Expr("balance + ?", coupon.Amount),
				"updated_at_ms": now,
			}
		case "percent":
			if coupon.Amount <= 0 {
				return fmt.Errorf("invalid percent coupon amount")
			}
			userUpdates = map[string]any{
				"rate_percent":  coupon.Amount,
				"updated_at_ms": now,
			}
		default:
			return fmt.Errorf("unsupported coupon kind")
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Updates(userUpdates).Error
	})
	if err != nil {
		return nil, err
	}
	if err := c.db.First(&coupon, coupon.ID).Error; err != nil {
		return nil, err
	}
	return &coupon, nil
}

// CreateAPIKey 创建API密钥
// 参数：
//   - in: 创建API密钥输入参数
//
// 返回：创建的API密钥和错误
func (c *Core) CreateAPIKey(in CreateAPIKeyInput) (*model.APIKey, error) {
	providerName := strings.TrimSpace(in.Provider)
	if providerName == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if _, err := c.providers.Get(providerName); err != nil {
		return nil, err
	}
	key := &model.APIKey{
		UserID:      in.UserID,
		Provider:    providerName,
		Name:        strings.TrimSpace(in.Name),
		Secret:      "sk-" + randomHex(24),
		Status:      "active",
		ExpiresAtMS: in.ExpiresAtMS,
	}
	if err := c.db.Create(key).Error; err != nil {
		return nil, err
	}
	c.invalidateUserDashboard(in.UserID)
	c.invalidateStats()
	return key, nil
}

// DeleteUserAPIKey 删除用户的API密钥
// 参数：
//   - userID: 用户ID
//   - keyID: 密钥ID
//
// 返回：错误
func (c *Core) DeleteUserAPIKey(userID, keyID uint64) error {
	if err := c.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&model.APIKey{}).Error; err != nil {
		return err
	}
	c.invalidateUserDashboard(userID)
	c.invalidateStats()
	return nil
}

// CreateAccount 创建AI账号
// 参数：
//   - in: 创建账号输入参数
//
// 返回：创建的账号和错误
func (c *Core) CreateAccount(in CreateAccountInput) (*model.Account, error) {
	models, _ := json.Marshal(normalizeStrings(in.ModelScope))
	encrypted, err := cryptoext.Encrypt(c.cfg.AESKey, normalizeJSON(in.Credentials, "{}"))
	if err != nil {
		return nil, err
	}
	account := &model.Account{
		Provider:             strings.TrimSpace(in.Provider),
		Name:                 strings.TrimSpace(in.Name),
		AuthType:             strings.TrimSpace(in.AuthType),
		BaseURL:              strings.TrimSpace(in.BaseURL),
		ModelScopeJSON:       string(models),
		CredentialsEncrypted: encrypted,
		Status:               "active",
		Priority:             defaultInt(in.Priority, 100),
		ConcurrencyLimit:     in.ConcurrencyLimit,
		MetadataJSON:         normalizeJSON(in.Metadata, "{}"),
	}
	if err := c.db.Create(account).Error; err != nil {
		return nil, err
	}
	c.reloadAccountCache()
	c.invalidateStats()
	return account, nil
}

func (c *Core) DeleteAccount(id uint64) error {
	err := c.db.Where("id = ?", id).Delete(&model.Account{}).Error
	if err == nil {
		c.accountLoads.Delete(id)
		c.refreshMu.Delete(id)
		c.reloadAccountCache()
		c.invalidateStats()
	}
	return err
}

// UpdateAccount 更新AI账号
// 参数：
//   - id: 账号ID
//   - in: 更新账号输入参数
//
// 返回：错误
func (c *Core) UpdateAccount(id uint64, in UpdateAccountInput) error {
	updates := map[string]any{}
	if in.Status != nil {
		if status := strings.TrimSpace(*in.Status); status != "" {
			updates["status"] = status
		}
	}
	if in.Priority != nil && *in.Priority > 0 {
		updates["priority"] = *in.Priority
	}
	if in.ConcurrencyLimit != nil && *in.ConcurrencyLimit >= 0 {
		updates["concurrency_limit"] = *in.ConcurrencyLimit
	}
	if in.CredentialsJSON != nil {
		if creds := strings.TrimSpace(*in.CredentialsJSON); creds != "" {
			encrypted, err := cryptoext.Encrypt(c.cfg.AESKey, creds)
			if err != nil {
				return err
			}
			updates["credentials_encrypted"] = encrypted
		}
	}
	if in.BaseURL != nil {
		if baseURL := strings.TrimSpace(*in.BaseURL); baseURL != "" {
			updates["base_url"] = baseURL
		}
	}
	if in.ModelScopeJSON != nil {
		if scope := strings.TrimSpace(*in.ModelScopeJSON); scope != "" {
			updates["model_scope_json"] = scope
		}
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at_ms"] = time.Now().UnixMilli()
	if err := c.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	c.reloadAccountCache()
	c.invalidateStats()
	return nil
}

// 参数：
//   - ctx: 上下文
//   - id: 账号ID
//
// 返回：账号视图和错误
func (c *Core) RefreshAccountOAuth(ctx context.Context, id uint64) (*AccountView, error) {
	var account model.Account
	if err := c.db.First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("account not found")
	}
	if account.AuthType != "oauth" {
		return nil, fmt.Errorf("account is not oauth")
	}
	cred, err := c.accountCredentials(&account)
	if err != nil {
		return nil, err
	}
	if _, err := c.refreshOAuthToken(ctx, &account, cred); err != nil {
		c.recordError("account.refresh", "manual oauth refresh failed", err.Error())
		return nil, err
	}
	c.reloadAccountCache()
	cred, _ = c.accountCredentials(&account)
	return &AccountView{
		Account:     account,
		Credentials: redactCredentialsForView(&account, cred),
	}, nil
}

// ListAccounts 获取所有AI账号列表
// 返回：账号列表和错误
func (c *Core) ListAccounts() ([]model.Account, error) {
	var items []model.Account
	err := c.db.Order("priority desc, id asc").Limit(200).Find(&items).Error
	return items, err
}

// ListAccountViews 获取所有AI账号视图列表（包含脱敏后的凭证）
// 返回：账号视图列表和错误
func (c *Core) ListAccountViews() ([]AccountView, error) {
	accounts, err := c.ListAccounts()
	if err != nil {
		return nil, err
	}
	views := make([]AccountView, 0, len(accounts))
	for i := range accounts {
		cred, _ := c.accountCredentials(&accounts[i])
		views = append(views, AccountView{
			Account:     accounts[i],
			Credentials: redactCredentialsForView(&accounts[i], cred),
		})
	}
	return views, nil
}

// CreateModelPrice 创建模型价格
// 参数：
//   - in: 创建价格输入参数
//
// 返回：创建的价格和错误
func (c *Core) CreateModelPrice(in CreateModelPriceInput) (*model.ModelPrice, error) {
	item := &model.ModelPrice{
		Provider:         strings.TrimSpace(in.Provider),
		Model:            strings.TrimSpace(in.Model),
		InputPrice:       in.InputPrice,
		OutputPrice:      in.OutputPrice,
		CacheCreatePrice: in.CacheCreatePrice,
		CacheReadPrice:   in.CacheReadPrice,
		Currency:         "CNY_1E4",
		Status:           defaultString(strings.TrimSpace(in.Status), "active"),
	}
	if err := c.db.Create(item).Error; err != nil {
		return nil, err
	}
	c.reloadPriceCache()
	c.invalidateStats()
	return item, nil
}

func (c *Core) reloadAccountCache() {
	var accounts []model.Account
	c.db.Where("status = ?", "active").Order("priority desc, id asc").Find(&accounts)
	m := map[string][]model.Account{}
	for _, a := range accounts {
		m[a.Provider] = append(m[a.Provider], a)
	}
	c.accountCacheMu.Lock()
	c.accountCache = m
	c.accountCacheMu.Unlock()
}

func (c *Core) reloadPriceCache() {
	var prices []model.ModelPrice
	c.db.Where("status = ?", "active").Find(&prices)
	m := map[string]model.ModelPrice{}
	for _, p := range prices {
		m[p.Provider+":"+p.Model] = p
	}
	c.priceCacheMu.Lock()
	c.priceCache = m
	c.priceCacheMu.Unlock()
}

// ListModelPrices 获取所有模型价格列表
// 返回：价格列表和错误
func (c *Core) ListModelPrices() ([]model.ModelPrice, error) {
	var items []model.ModelPrice
	err := c.db.Order("provider asc, model asc").Limit(500).Find(&items).Error
	return items, err
}

// CreateAnnouncement 创建公告
// 参数：
//   - in: 创建公告输入参数
//
// 返回：创建的公告和错误
func (c *Core) CreateAnnouncement(in CreateAnnouncementInput) (*model.Announcement, error) {
	item := &model.Announcement{
		Title:         strings.TrimSpace(in.Title),
		Content:       in.Content,
		Status:        defaultString(in.Status, "published"),
		PublishedAtMS: defaultInt64(in.PublishedAtMS, time.Now().UnixMilli()),
	}
	if err := c.db.Create(item).Error; err != nil {
		return nil, err
	}
	c.invalidateStats()
	return item, nil
}

// ListAnnouncements 获取所有公告列表
// 返回：公告列表和错误
func (c *Core) ListAnnouncements() ([]model.Announcement, error) {
	var items []model.Announcement
	err := c.db.Order("published_at_ms desc, id desc").Limit(100).Find(&items).Error
	return items, err
}

// CreateCoupon 创建优惠券
// 参数：
//   - in: 创建优惠券输入参数
//
// 返回：创建的优惠券和错误
func (c *Core) CreateCoupon(in CreateCouponInput) (*model.Coupon, error) {
	kind := strings.TrimSpace(strings.ToLower(in.Kind))
	if kind == "" {
		kind = "balance"
	}
	switch kind {
	case "balance":
		if in.Amount <= 0 {
			return nil, fmt.Errorf("coupon amount must be greater than 0")
		}
	case "percent":
		if in.Amount <= 0 || in.Amount > 1000 {
			return nil, fmt.Errorf("percent coupon amount must be between 1 and 1000")
		}
	default:
		return nil, fmt.Errorf("unsupported coupon kind")
	}
	item := &model.Coupon{
		Code:         strings.TrimSpace(in.Code),
		Kind:         kind,
		Amount:       in.Amount,
		MaxUses:      defaultInt(in.MaxUses, 1),
		ExpiresAtMS:  in.ExpiresAtMS,
		Status:       "active",
		UsageLogJSON: "[]",
	}
	if err := c.db.Create(item).Error; err != nil {
		return nil, err
	}
	c.invalidateStats()
	return item, nil
}

// ListCoupons 获取所有优惠券列表
// 返回：优惠券列表和错误
func (c *Core) ListCoupons() ([]model.Coupon, error) {
	var items []model.Coupon
	err := c.db.Order("id desc").Limit(100).Find(&items).Error
	return items, err
}

// ListErrorLogs 获取错误日志列表
// 参数：
//   - limit: 返回记录数量限制
//
// 返回：错误日志列表和错误
func (c *Core) ListErrorLogs(limit int) ([]model.ErrorLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []model.ErrorLog
	err := c.db.Order("last_seen_at_ms desc, id desc").Limit(limit).Find(&items).Error
	return items, err
}

// Dashboard 获取仪表盘数据
// 返回：仪表盘数据结构和错误
func (c *Core) Dashboard() (*DashboardData, error) {
	stats, err := c.Stats()
	if err != nil {
		return nil, err
	}
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	accounts, err := c.ListAccountViews()
	if err != nil {
		return nil, err
	}
	prices, err := c.ListModelPrices()
	if err != nil {
		return nil, err
	}
	orders, err := c.ListPaymentOrders()
	if err != nil {
		return nil, err
	}
	ann, err := c.ListAnnouncements()
	if err != nil {
		return nil, err
	}
	coupons, err := c.ListCoupons()
	if err != nil {
		return nil, err
	}
	return &DashboardData{
		Users:         users,
		Accounts:      accounts,
		Prices:        prices,
		Orders:        orders,
		Announcements: ann,
		Coupons:       coupons,
		Stats:         stats,
	}, nil
}

func (c *Core) CreatePaymentOrder(ctx context.Context, in CreatePaymentOrderInput, clientIP, device, baseURL string) (*model.PaymentOrder, *payment.CreateOrderResponse, error) {
	// 获取支付提供商（GoPay）
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return nil, nil, err
	}
	// 验证用户ID
	if in.UserID == 0 {
		return nil, nil, fmt.Errorf("user_id is required")
	}
	// 验证金额
	if in.Amount <= 0 {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}
	// 设置基础URL
	if baseURL == "" {
		baseURL = c.cfg.PublicBaseURL
	}
	if baseURL == "" {
		return nil, nil, fmt.Errorf("public base url is required for payment notify")
	}
	// 创建支付订单记录
	order := &model.PaymentOrder{
		UserID:         in.UserID,
		Provider:       "gopay",
		OutTradeNo:     "pay_" + randomHex(12), // 生成唯一订单号
		Subject:        defaultString(strings.TrimSpace(in.Subject), "Balance Recharge"),
		Status:         "PENDING", // 初始状态为待支付
		Amount:         in.Amount,
		CreditedAmount: in.Amount,
		MetadataJSON:   "{}",
	}
	// 保存订单到数据库
	if err := c.db.Create(order).Error; err != nil {
		return nil, nil, err
	}
	// 调用支付提供商创建订单
	result, err := providerImpl.CreateOrder(ctx, payment.CreateOrderRequest{
		OutTradeNo: order.OutTradeNo,
		Subject:    order.Subject,
		Amount:     order.Amount,
		NotifyURL:  strings.TrimRight(baseURL, "/") + "/api/payments/notify/gopay",
		ReturnURL:  in.ReturnURL,
		ClientIP:   clientIP,
		Device:     device,
	})
	if err != nil {
		c.recordError("payment.create", "gopay create order failed", err.Error())
		return nil, nil, err
	}
	// 更新订单的交易号
	now := time.Now().UnixMilli()
	if err := c.db.Model(order).Updates(map[string]any{
		"provider_trade_no": result.ProviderTradeNo,
		"updated_at_ms":     now,
	}).Error; err != nil {
		return nil, nil, err
	}
	order.ProviderTradeNo = result.ProviderTradeNo
	order.UpdatedAtMS = now
	c.invalidateUserDashboard(in.UserID)
	c.invalidateStats()
	return order, result, nil
}

func (c *Core) HandlePaymentNotify(r *http.Request) error {
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return err
	}
	// 验证支付通知
	notify, err := providerImpl.VerifyNotify(r)
	if err != nil {
		c.recordError("payment.notify", "gopay notify verify failed", err.Error())
		return err
	}
	// 检查支付是否完成
	if !notify.Paid {
		return fmt.Errorf("payment not completed")
	}
	// 事务处理：更新订单状态和用户余额
	var dashboardUserID uint64
	err = c.db.Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		// 查找订单
		if err := tx.Where("out_trade_no = ?", notify.OutTradeNo).First(&order).Error; err != nil {
			return err
		}
		// 回调金额必须严格匹配订单金额，0 或负数均视为非法
		if notify.Amount <= 0 || notify.Amount != order.Amount {
			return fmt.Errorf("payment amount mismatch: expected %d, got %d", order.Amount, notify.Amount)
		}
		// 更新订单状态为已支付
		now := time.Now().UnixMilli()
		result := tx.Model(&model.PaymentOrder{}).
			Where("id = ? AND status <> ?", order.ID, "PAID").
			Updates(map[string]any{
			"status":            "PAID",
			"provider_trade_no": notify.ProviderTradeNo,
			"notified_at_ms":    now,
			"updated_at_ms":     now,
		})
		if result.Error != nil {
			return result.Error
		}
		// 幂等：并发回调时只有一个事务能从非 PAID 切换到 PAID
		if result.RowsAffected == 0 {
			return nil
		}
		// 增加用户余额
		if err := tx.Model(&model.User{}).Where("id = ?", order.UserID).Updates(map[string]any{
			"balance":       gorm.Expr("balance + ?", order.CreditedAmount),
			"updated_at_ms": now,
		}).Error; err != nil {
			return err
		}
		dashboardUserID = order.UserID
		return nil
	})
	if err == nil && dashboardUserID != 0 {
		c.invalidateUserDashboard(dashboardUserID)
		c.invalidateStats()
	}
	return err
}

// RefundPayment 退款
// 参数：
//   - ctx: 上下文
//   - outTradeNo: 订单号
//   - amount: 退款金额
//
// 返回：错误
func (c *Core) RefundPayment(ctx context.Context, outTradeNo string, amount int64) error {
	// 获取支付提供商
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return err
	}
	// 查找原订单
	var order model.PaymentOrder
	if err := c.db.Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
		return err
	}
	// 检查订单是否已支付
	if order.Status != "PAID" {
		return fmt.Errorf("payment order is not paid")
	}
	// 验证退款金额不超过可退金额
	refundable := order.CreditedAmount - order.RefundedAmount
	if amount <= 0 || amount > refundable {
		return fmt.Errorf("invalid refund amount: %d, refundable: %d", amount, refundable)
	}
	// 查找用户
	var user model.User
	if err := c.db.First(&user, order.UserID).Error; err != nil {
		return err
	}
	// 检查用户余额是否足够
	if user.Balance < amount {
		return fmt.Errorf("user balance is insufficient for refund")
	}
	// 调用支付提供商退款
	if err := providerImpl.Refund(ctx, payment.RefundRequest{
		ProviderTradeNo: order.ProviderTradeNo,
		Amount:          amount,
	}); err != nil {
		c.recordError("payment.refund", "gopay refund failed", err.Error())
		return err
	}
	// 更新订单状态和用户余额
	now := time.Now().UnixMilli()
	nextRefunded := order.RefundedAmount + amount
	nextStatus := "PAID"
	if nextRefunded >= order.CreditedAmount {
		nextStatus = "REFUNDED"
	}
	err = c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Updates(map[string]any{
			"status":          nextStatus,
			"refunded_amount": gorm.Expr("refunded_amount + ?", amount),
			"updated_at_ms":   now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&user).Updates(map[string]any{
			"balance":       gorm.Expr("balance - ?", amount),
			"updated_at_ms": now,
		}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		c.recordError("payment.refund", "refund local transaction failed after upstream refund", err.Error())
	}
	if err == nil {
		c.invalidateUserDashboard(user.ID)
		c.invalidateStats()
	}
	return err
}

// AuthenticateAPIKey 验证API密钥
// 参数：
//   - secret: API密钥
//
// 返回：代理认证信息和错误
func (c *Core) AuthenticateAPIKey(secret string) (*ProxyAuth, error) {
	trimmedSecret := strings.TrimSpace(secret)
	var authRow struct {
		APIKeyID          uint64
		APIKeyUserID      uint64
		APIKeyProvider    string
		APIKeyExpiresAtMS int64
		UserID            uint64
		UserBalance       int64
		UserRatePercent   int
		UserAllowedModels string
	}
	err := c.db.Table("api_keys").
		Select(
			"api_keys.id as api_key_id, api_keys.user_id as api_key_user_id, "+
				"api_keys.provider as api_key_provider, api_keys.expires_at_ms as api_key_expires_at_ms, "+
				"users.id as user_id, users.balance as user_balance, users.rate_percent as user_rate_percent, "+
				"users.allowed_models_json as user_allowed_models",
		).
		Joins("JOIN users ON users.id = api_keys.user_id").
		Where("api_keys.secret = ? AND api_keys.status = ? AND users.status = ?", trimmedSecret, "active", "active").
		Take(&authRow).Error
	if err != nil {
		return nil, fmt.Errorf("invalid api key")
	}
	// 检查密钥是否过期
	if authRow.APIKeyExpiresAtMS > 0 && authRow.APIKeyExpiresAtMS < time.Now().UnixMilli() {
		return nil, fmt.Errorf("api key expired")
	}
	now := time.Now().UnixMilli()
	c.pendingKeyMu.Lock()
	c.pendingKeys[authRow.APIKeyID] = now
	c.pendingKeyMu.Unlock()
	return &ProxyAuth{
		User: model.User{
			ID:                authRow.UserID,
			Balance:           authRow.UserBalance,
			RatePercent:       authRow.UserRatePercent,
			AllowedModelsJSON: authRow.UserAllowedModels,
		},
		APIKey: model.APIKey{
			ID:           authRow.APIKeyID,
			UserID:       authRow.APIKeyUserID,
			Provider:     authRow.APIKeyProvider,
			ExpiresAtMS:  authRow.APIKeyExpiresAtMS,
		},
		UserAllowedSet: parseAllowedModelsSet(authRow.UserAllowedModels),
	}, nil
}

// Proxy 代理AI请求
// 核心转发逻辑：根据API Key查找用户和AI账号，构建上游请求并转发
// 参数：
//   - ctx: 上下文
//   - auth: 代理认证信息
//   - path: 请求路径
//   - rawQuery: 原始查询字符串
//   - hdr: 请求头
//   - body: 请求体
//
// 返回：上游响应、响应体和错误
func (c *Core) Proxy(ctx context.Context, auth *ProxyAuth, path, rawQuery string, hdr http.Header, body []byte) (*http.Response, []byte, error) {
	// 检测请求路由：模型名、是否流式、Provider名称
	modelName, stream, providerName, err := detectRoute(path, body)
	if err != nil {
		return nil, nil, err
	}
	// API Key 只允许在自身绑定的 provider 下使用
	if auth.APIKey.Provider == "" || auth.APIKey.Provider != providerName {
		return nil, nil, fmt.Errorf("api key provider mismatch")
	}
	// 用户级模型权限仍然生效
	if modelName != "" && !isModelAllowedSet(auth.UserAllowedSet, modelName) {
		return nil, nil, fmt.Errorf("model is not allowed")
	}
	// 生成缓存键，检查是否可缓存
	cacheKey, cacheable := c.cacheKey(auth.User.ID, providerName, path, rawQuery, body, stream)
	if cacheable {
		// 尝试从缓存获取响应
		if cached, ok := c.getCachedResponse(cacheKey); ok {
			respBody, err := io.ReadAll(cached.Body)
			if err != nil {
				return nil, nil, err
			}
			cached.Body = io.NopCloser(bytes.NewReader(respBody))
			return cached, respBody, nil
		}
	}
	// 选择一个可用的AI账号
	account, err := c.pickAccount(providerName, modelName, path)
	if err != nil {
		return nil, nil, err
	}
	// 请求完成后释放账号
	defer c.releaseAccount(account.ID)
	// 获取账号的访问令牌
	token, err := c.accountToken(ctx, &account)
	if err != nil {
		c.recordError("proxy.token", "resolve upstream token failed", err.Error())
		return nil, nil, err
	}
	// 获取Provider实现
	providerImpl, err := c.providers.Get(account.Provider)
	if err != nil {
		return nil, nil, err
	}
	// 仅在模型已知会收费时做余额预检，避免免费模型被错误拦截
	if !c.isFreeModel(account.Provider, modelName, auth.User.RatePercent) && auth.User.Balance <= 0 {
		return nil, nil, fmt.Errorf("insufficient balance")
	}
	// 构建上游URL
	upstreamURL := providerImpl.BuildUpstreamURL(account, path, rawQuery)
	// 创建上游请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	// 复制请求头
	copyHeaders(req.Header, hdr)
	req.Header.Del("Authorization")
	// 应用Provider特定的请求处理（如签名、认证头等）
	if err := providerImpl.ApplyRequest(req, account, token); err != nil {
		return nil, nil, err
	}
	// 发送上游请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordError("proxy.request", "upstream request failed", err.Error())
		return nil, nil, err
	}
	// 处理流式响应
	if stream || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "event-stream") {
		if resp.StatusCode < 400 && modelName != "" {
			// 使用usageTrackingReadCloser跟踪流式响应的使用量
			resp.Body = &usageTrackingReadCloser{
				src: resp.Body,
				finalize: func(body []byte) {
					// 解析使用量
					inTokens, outTokens := providerImpl.ParseUsage(body)
					cacheCreateTokens, cacheReadTokens := c.parseCacheUsage(providerImpl, body)
					// 尝试从流式响应中解析使用量
					if parser, ok := providerImpl.(provider.StreamUsageParser); ok {
						if in, out, found := parser.ParseStreamUsage(body); found {
							inTokens = in
							outTokens = out
						}
					}
					if cacheCreateTokens == 0 && cacheReadTokens == 0 {
						if parser, ok := providerImpl.(provider.CacheUsageParser); ok {
							cacheCreateTokens, cacheReadTokens, _ = parser.ParseCacheUsage(body)
						}
					}
					// 记录使用量
					if inTokens > 0 || outTokens > 0 || cacheCreateTokens > 0 || cacheReadTokens > 0 {
						if err := c.recordUsage(auth, &account, modelName, path, inTokens, outTokens, cacheCreateTokens, cacheReadTokens); err != nil {
							c.recordError("usage.record", "record stream usage failed", err.Error())
						}
					}
				},
			}
		}
		return resp, nil, nil
	}
	// 处理非流式响应
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	// 缓存成功响应
	if cacheable && resp.StatusCode < 400 {
		c.putCachedResponse(cacheKey, resp.StatusCode, resp.Header, respBody, 30*time.Second)
	}
	// 解析并记录使用量
	if resp.StatusCode < 400 && modelName != "" {
		inTokens, outTokens := providerImpl.ParseUsage(respBody)
		cacheCreateTokens, cacheReadTokens := c.parseCacheUsage(providerImpl, respBody)
		if err := c.recordUsage(auth, &account, modelName, path, inTokens, outTokens, cacheCreateTokens, cacheReadTokens); err != nil {
			c.recordError("usage.record", "record usage failed", err.Error())
		}
	}
	return resp, respBody, nil
}

// OAuthStart 开始OAuth授权流程
// 参数：
//   - in: OAuth授权开始输入参数
//
// 返回：OAuth授权结果和错误
func (c *Core) OAuthStart(in OAuthStartInput) (*OAuthStartResult, error) {
	providerName := normalizeProvider(in.Provider)
	state := randomState()
	codeVerifier := randomCodeVerifier(providerName)
	redirectURI := strings.TrimSpace(in.RedirectURI)
	if redirectURI == "" {
		redirectURI = defaultRedirectURI(providerName, strings.TrimSpace(in.OAuthType))
	}
	meta := map[string]string{
		"provider":   providerName,
		"oauth_type": strings.TrimSpace(in.OAuthType),
		"project_id": strings.TrimSpace(in.ProjectID),
		"tier_id":    strings.TrimSpace(in.TierID),
	}
	metaRaw, _ := json.Marshal(meta)
	session := &model.OAuthSession{
		Provider:     providerName,
		State:        state,
		CodeVerifier: codeVerifier,
		RedirectURI:  redirectURI,
		ExpiresAtMS:  time.Now().Add(30 * time.Minute).UnixMilli(),
		MetadataJSON: string(metaRaw),
	}
	if err := c.db.Create(session).Error; err != nil {
		return nil, err
	}
	authURL, err := c.buildAuthorizationURL(providerName, state, codeVerifier, redirectURI, meta)
	if err != nil {
		return nil, err
	}
	return &OAuthStartResult{
		Provider:  providerName,
		SessionID: fmt.Sprintf("%d", session.ID),
		State:     state,
		AuthURL:   authURL,
	}, nil
}

// OAuthExchange 交换OAuth授权码获取令牌
// 参数：
//   - ctx: 上下文
//   - in: OAuth令牌交换输入参数
//
// 返回：OAuth令牌交换结果和错误
func (c *Core) OAuthExchange(ctx context.Context, in OAuthExchangeInput) (*OAuthExchangeResult, error) {
	sessionID, err := strconv.ParseUint(strings.TrimSpace(in.SessionID), 10, 64)
	if err != nil || sessionID == 0 {
		return nil, fmt.Errorf("invalid oauth session id")
	}
	// 查找OAuth会话
	var session model.OAuthSession
	if err := c.db.Where("id = ?", sessionID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("oauth session not found")
	}
	// 检查会话是否过期
	if session.ExpiresAtMS < time.Now().UnixMilli() {
		return nil, fmt.Errorf("oauth session expired")
	}
	// 验证state参数防止CSRF攻击
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(in.State)), []byte(session.State)) != 1 {
		return nil, fmt.Errorf("invalid oauth state")
	}
	// 解析元数据
	meta := map[string]string{}
	_ = json.Unmarshal([]byte(session.MetadataJSON), &meta)
	// 交换授权码获取令牌
	result, err := c.exchangeOAuthCode(ctx, &session, meta, strings.TrimSpace(in.Code))
	if err != nil {
		return nil, err
	}
	// 删除会话记录
	_ = c.db.Delete(&session).Error
	return &OAuthExchangeResult{AccountCredentials: *result}, nil
}

// CreateAccountFromOAuth 从OAuth凭证创建AI账号
// 参数：
//   - providerName: Provider名称
//   - name: 账号名称
//   - modelScope: 支持的模型列表
//   - creds: OAuth凭证
//   - baseURL: 基础URL
//   - priority: 优先级
//   - limit: 并发限制
//
// 返回：创建的账号和错误
func (c *Core) CreateAccountFromOAuth(providerName, name string, modelScope []string, creds *AccountCredentials, baseURL string, priority, limit int) (*model.Account, error) {
	raw := mustJSON(creds)
	return c.CreateAccount(CreateAccountInput{
		Provider:         normalizeProvider(providerName),
		Name:             strings.TrimSpace(name),
		AuthType:         "oauth",
		BaseURL:          strings.TrimSpace(baseURL),
		ModelScope:       modelScope,
		Credentials:      json.RawMessage(raw),
		Priority:         priority,
		ConcurrencyLimit: limit,
		Metadata:         json.RawMessage(`{}`),
	})
}

// Stats 获取系统统计信息
// 返回：统计数据映射和错误
func (c *Core) Stats() (map[string]any, error) {
	if cached, ok := c.getCachedStats(); ok {
		return cached, nil
	}
	type target struct {
		name  string
		model any
	}
	targets := []target{
		{"users", &model.User{}},
		{"accounts", &model.Account{}},
		{"model_prices", &model.ModelPrice{}},
		{"payment_orders", &model.PaymentOrder{}},
		{"announcements", &model.Announcement{}},
		{"coupons", &model.Coupon{}},
	}
	data := map[string]any{}
	for _, item := range targets {
		var count int64
		if err := c.db.Model(item.model).Count(&count).Error; err != nil {
			return nil, err
		}
		data[item.name] = count
	}
	var activeAccounts int64
	c.db.Model(&model.Account{}).Where("status = ?", "active").Count(&activeAccounts)
	data["active_accounts"] = activeAccounts
	var activeUsers int64
	c.db.Model(&model.User{}).Where("status = ?", "active").Count(&activeUsers)
	data["active_users"] = activeUsers
	var paidOrders int64
	c.db.Model(&model.PaymentOrder{}).Where("status = ?", "PAID").Count(&paidOrders)
	data["paid_orders"] = paidOrders
	var totalRevenue int64
	c.db.Model(&model.PaymentOrder{}).Where("status = ?", "PAID").Select("COALESCE(SUM(credited_amount), 0)").Row().Scan(&totalRevenue)
	data["total_revenue"] = totalRevenue
	var totalUsage int64
	c.db.Model(&model.UsageLog{}).Select("COALESCE(SUM(input_tokens + output_tokens), 0)").Row().Scan(&totalUsage)
	data["total_tokens"] = totalUsage
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	data["memory_alloc_mb"] = ms.Alloc / 1024 / 1024
	data["cache_items"] = len(c.cacheItems)
	data["timestamp_ms"] = time.Now().UnixMilli()
	c.setCachedStats(data)
	return data, nil
}

// ListPaymentOrders 获取所有支付订单列表
// 返回：订单列表和错误
func (c *Core) ListPaymentOrders() ([]model.PaymentOrder, error) {
	var items []model.PaymentOrder
	err := c.db.Order("id desc").Limit(500).Find(&items).Error
	return items, err
}

// snapshotMetricsLoop 指标快照循环
// 每5分钟记录一次系统指标到数据库
// 参数：
//   - ctx: 上下文
func (c *Core) snapshotMetricsLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 获取系统统计
			stats, err := c.Stats()
			if err != nil {
				continue
			}
			_ = c.db.Transaction(func(tx *gorm.DB) error {
				now := time.Now().UnixMilli()
				items := make([]model.SystemMetric, 0, len(stats))
				for k, v := range stats {
					items = append(items, model.SystemMetric{
						MetricKey:    k,
						MetricValue:  fmt.Sprintf("%v", v),
						SnapshotAtMS: now,
						MetadataJSON: "{}",
					})
				}
				return tx.Create(&items).Error
			})
		}
	}
}

// refreshOAuthLoop OAuth令牌刷新循环
// 每分钟检查并刷新即将过期的OAuth令牌
// 参数：
//   - ctx: 上下文
func (c *Core) refreshOAuthLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 查找所有OAuth类型的活跃账号
			var accounts []model.Account
			if err := c.db.Where("auth_type = ? AND status = ?", "oauth", "active").Find(&accounts).Error; err != nil {
				continue
			}
			// 刷新即将过期的令牌（剩余时间少于10分钟）
			for i := range accounts {
				if accounts[i].ExpiresAtMS > 0 && accounts[i].ExpiresAtMS-time.Now().UnixMilli() > 10*60*1000 {
					continue
				}
				_, _ = c.accountToken(ctx, &accounts[i])
			}
		}
	}
}

// accountCredentials 获取账号凭证
// 从加密的凭证中解密并解析出AccountCredentials
// 参数：
//   - account: AI账号
//
// 返回：账号凭证和错误
func (c *Core) accountCredentials(account *model.Account) (*AccountCredentials, error) {
	// 解密凭证
	plain, err := cryptoext.Decrypt(c.cfg.AESKey, account.CredentialsEncrypted)
	if err != nil {
		return nil, err
	}
	// 解析JSON
	var cred AccountCredentials
	if err := json.Unmarshal([]byte(plain), &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// accountToken 获取账号访问令牌
// 根据账号类型返回API密钥或OAuth访问令牌
// 参数：
//   - ctx: 上下文
//   - account: AI账号
//
// 返回：访问令牌和错误
func (c *Core) accountToken(ctx context.Context, account *model.Account) (string, error) {
	// 获取账号凭证
	cred, err := c.accountCredentials(account)
	if err != nil {
		return "", err
	}
	// 根据认证类型返回令牌
	switch account.AuthType {
	case "api_key", "static":
		// 直接返回API密钥或访问令牌
		if cred.APIKey != "" {
			return cred.APIKey, nil
		}
		return cred.AccessToken, nil
	case "oauth":
		// 检查访问令牌是否还有效（剩余超过3分钟）
		if cred.AccessToken != "" && cred.ExpiresAtMS-time.Now().UnixMilli() > 3*60*1000 {
			return cred.AccessToken, nil
		}
		// 刷新OAuth令牌
		return c.refreshOAuthToken(ctx, account, cred)
	default:
		return "", fmt.Errorf("unsupported auth_type %s", account.AuthType)
	}
}

// refreshOAuthToken 刷新OAuth访问令牌
// 使用刷新令牌获取新的访问令牌
// 参数：
//   - ctx: 上下文
//   - account: AI账号
//   - cred: 当前凭证
//
// 返回：新的访问令牌和错误
func (c *Core) refreshOAuthToken(ctx context.Context, account *model.Account, cred *AccountCredentials) (string, error) {
	// 检查是否有刷新令牌
	if cred.RefreshToken == "" {
		return "", fmt.Errorf("oauth refresh token is missing")
	}
	// 使用互斥锁防止并发刷新同一账号
	lockAny, _ := c.refreshMu.LoadOrStore(account.ID, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

	// 构建刷新令牌请求表单
	form := url.Values{}
	switch normalizeProvider(account.Provider) {
	case "openai":
		form.Set("grant_type", "refresh_token")
		form.Set("client_id", defaultString(cred.ClientID, c.cfg.OpenAI.ClientID))
		form.Set("refresh_token", cred.RefreshToken)
		form.Set("scope", openAIRefreshScopes)
		cred.TokenURL = openAITokenURL
	case "claude":
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", cred.RefreshToken)
		form.Set("client_id", defaultString(cred.ClientID, c.cfg.Claude.ClientID))
		if cred.ClientSecret != "" {
			form.Set("client_secret", cred.ClientSecret)
		}
		cred.TokenURL = claudeTokenURL
	case "gemini":
		cfg, redirectURI, scopes := c.geminiOAuthConfig(cred.OAuthType)
		form.Set("grant_type", "refresh_token")
		form.Set("client_id", cfg.ClientID)
		if cfg.ClientSecret != "" {
			form.Set("client_secret", cfg.ClientSecret)
		}
		form.Set("refresh_token", cred.RefreshToken)
		form.Set("scope", scopes)
		cred.TokenURL = geminiTokenURL
		if cred.RedirectURI == "" {
			cred.RedirectURI = redirectURI
		}
		cred.ClientID = cfg.ClientID
		cred.ClientSecret = cfg.ClientSecret
	case "antigravity":
		form.Set("grant_type", "refresh_token")
		form.Set("client_id", antigravityClientID)
		form.Set("client_secret", c.cfg.Antigravity.ClientSecret)
		form.Set("refresh_token", cred.RefreshToken)
		cred.TokenURL = antigravityTokenURL
	default:
		return "", fmt.Errorf("unsupported oauth provider %s", account.Provider)
	}
	// 发送刷新令牌请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cred.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	// 检查响应状态
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("oauth refresh failed: %s", strings.TrimSpace(string(body)))
	}
	// 解析响应
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		IDToken      string `json:"id_token"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("oauth refresh returned empty access token")
	}
	// 更新凭证
	cred.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		cred.RefreshToken = result.RefreshToken
	}
	if result.ExpiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli()
	}
	// 处理ID Token
	if result.IDToken != "" {
		c.populateOpenAIIDToken(cred, result.IDToken)
	}
	// 返回新令牌并保存凭证
	return cred.AccessToken, c.persistCredentials(account, cred)
}

// persistCredentials 持久化账号凭证
// 加密并保存凭证到数据库
// 参数：
//   - account: AI账号
//   - cred: 账号凭证
//
// 返回：错误
func (c *Core) persistCredentials(account *model.Account, cred *AccountCredentials) error {
	// 加密凭证
	encrypted, err := cryptoext.Encrypt(c.cfg.AESKey, mustJSON(cred))
	if err != nil {
		return err
	}
	// 更新数据库
	now := time.Now().UnixMilli()
	if err := c.db.Model(&model.Account{}).Where("id = ?", account.ID).Updates(map[string]any{
		"credentials_encrypted": encrypted,
		"expires_at_ms":         cred.ExpiresAtMS,
		"last_refreshed_at_ms":  now,
		"updated_at_ms":         now,
	}).Error; err != nil {
		return err
	}
	// 更新本地对象
	account.CredentialsEncrypted = encrypted
	account.ExpiresAtMS = cred.ExpiresAtMS
	account.LastRefreshedAtMS = now
	return nil
}

// buildAuthorizationURL 构建OAuth授权URL
// 根据Provider生成授权跳转URL
// 参数：
//   - providerName: Provider名称
//   - state: 状态参数
//   - codeVerifier: PKCE代码验证器
//   - redirectURI: 回调URI
//   - meta: 元数据
//
// 返回：授权URL和错误
func (c *Core) buildAuthorizationURL(providerName, state, codeVerifier, redirectURI string, meta map[string]string) (string, error) {
	// 生成code challenge
	challenge := pkceChallenge(codeVerifier)
	switch providerName {
	case "openai":
		params := url.Values{}
		params.Set("response_type", "code")
		params.Set("client_id", c.cfg.OpenAI.ClientID)
		params.Set("redirect_uri", redirectURI)
		params.Set("scope", openAIScopes)
		params.Set("state", state)
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		params.Set("id_token_add_organizations", "true")
		params.Set("codex_cli_simplified_flow", "true")
		return openAIAuthorizeURL + "?" + params.Encode(), nil
	case "claude":
		return fmt.Sprintf("%s?code=true&client_id=%s&response_type=code&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
			claudeAuthorizeURL,
			url.QueryEscape(c.cfg.Claude.ClientID),
			url.QueryEscape(redirectURI),
			strings.ReplaceAll(url.QueryEscape(claudeScopeOAuth), "%20", "+"),
			url.QueryEscape(challenge),
			url.QueryEscape(state),
		), nil
	case "gemini":
		cfg, effectiveRedirect, scopes := c.geminiOAuthConfig(meta["oauth_type"])
		if redirectURI == "" {
			redirectURI = effectiveRedirect
		}
		params := url.Values{}
		params.Set("response_type", "code")
		params.Set("client_id", cfg.ClientID)
		params.Set("redirect_uri", redirectURI)
		params.Set("scope", scopes)
		params.Set("state", state)
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		if projectID := strings.TrimSpace(meta["project_id"]); projectID != "" {
			params.Set("project_id", projectID)
		}
		return geminiAuthorizeURL + "?" + params.Encode(), nil
	case "antigravity":
		params := url.Values{}
		params.Set("response_type", "code")
		params.Set("client_id", antigravityClientID)
		params.Set("redirect_uri", antigravityRedirectURI)
		params.Set("scope", antigravityScopes)
		params.Set("state", state)
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
		params.Set("access_type", "offline")
		params.Set("prompt", "consent")
		return antigravityAuthorizeURL + "?" + params.Encode(), nil
	default:
		return "", fmt.Errorf("unsupported oauth provider %s", providerName)
	}
}

func (c *Core) exchangeOAuthCode(ctx context.Context, session *model.OAuthSession, meta map[string]string, code string) (*AccountCredentials, error) {
	// 根据Provider类型分发处理
	providerName := normalizeProvider(session.Provider)
	switch providerName {
	case "openai":
		return c.exchangeOpenAI(ctx, session, code)
	case "claude":
		return c.exchangeClaude(ctx, session, code)
	case "gemini":
		return c.exchangeGemini(ctx, session, meta, code)
	case "antigravity":
		return c.exchangeAntigravity(ctx, session, code)
	default:
		return nil, fmt.Errorf("unsupported oauth provider %s", providerName)
	}
}

// exchangeOpenAI 交换OpenAI授权码
// 使用授权码换取访问令牌
func (c *Core) exchangeOpenAI(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
	// 构建令牌请求
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.cfg.OpenAI.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", session.RedirectURI)
	form.Set("code_verifier", session.CodeVerifier)
	// 发送请求
	resp, err := c.oauthFormRequest(ctx, openAITokenURL, form)
	if err != nil {
		return nil, err
	}
	// 构建凭证
	cred := &AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     c.cfg.OpenAI.ClientID,
		TokenURL:     openAITokenURL,
		RedirectURI:  session.RedirectURI,
	}
	// 设置过期时间
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	// 处理ID Token
	c.populateOpenAIIDToken(cred, resp["id_token"])
	return cred, nil
}

// exchangeClaude 交换Claude授权码
// 使用授权码换取访问令牌
func (c *Core) exchangeClaude(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
	// 构建请求载荷
	payload := map[string]any{
		"grant_type":    "authorization_code",
		"client_id":     c.cfg.Claude.ClientID,
		"code":          code,
		"redirect_uri":  session.RedirectURI,
		"code_verifier": session.CodeVerifier,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeTokenURL, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("claude oauth exchange failed: %s", strings.TrimSpace(string(body)))
	}
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		Scope        string `json:"scope"`
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
	cred := &AccountCredentials{
		AccessToken:    result.AccessToken,
		RefreshToken:   result.RefreshToken,
		ClientID:       c.cfg.Claude.ClientID,
		TokenURL:       claudeTokenURL,
		RedirectURI:    session.RedirectURI,
		Email:          result.Account.EmailAddress,
		OrganizationID: result.Organization.UUID,
		AccountID:      result.Account.UUID,
	}
	if result.ExpiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

func (c *Core) exchangeGemini(ctx context.Context, session *model.OAuthSession, meta map[string]string, code string) (*AccountCredentials, error) {
	cfg, _, scopes := c.geminiOAuthConfig(meta["oauth_type"])
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", cfg.ClientID)
	if cfg.ClientSecret != "" {
		form.Set("client_secret", cfg.ClientSecret)
	}
	form.Set("code", code)
	form.Set("redirect_uri", session.RedirectURI)
	form.Set("code_verifier", session.CodeVerifier)
	resp, err := c.oauthFormRequest(ctx, geminiTokenURL, form)
	if err != nil {
		return nil, err
	}
	// 构建凭证
	cred := &AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     geminiTokenURL,
		RedirectURI:  session.RedirectURI,
		OAuthType:    meta["oauth_type"],
		ProjectID:    meta["project_id"],
		TierID:       meta["tier_id"],
	}
	_ = scopes
	// 设置过期时间
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

// exchangeAntigravity 交换Antigravity授权码
// 使用授权码换取访问令牌
func (c *Core) exchangeAntigravity(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
	// 构建令牌请求
	form := url.Values{}
	form.Set("client_id", antigravityClientID)
	form.Set("client_secret", c.cfg.Antigravity.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", antigravityRedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", session.CodeVerifier)
	// 发送请求
	resp, err := c.oauthFormRequest(ctx, antigravityTokenURL, form)
	if err != nil {
		return nil, err
	}
	// 构建凭证
	cred := &AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     antigravityClientID,
		ClientSecret: c.cfg.Antigravity.ClientSecret,
		TokenURL:     antigravityTokenURL,
		RedirectURI:  antigravityRedirectURI,
		UserAgent:    "antigravity/1.21.9 windows/amd64",
	}
	// 设置过期时间
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	// 获取用户信息
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, antigravityUserInfoURL, nil)
	if err == nil {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
		if userResp, doErr := c.httpClient.Do(req); doErr == nil {
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

// oauthFormRequest 发送OAuth表单请求
// 通用方法：发送表单编码的请求并解析JSON响应
// 参数：
//   - ctx: 上下文
//   - endpoint: 请求端点
//   - form: 表单数据
//
// 返回：响应映射和错误
func (c *Core) oauthFormRequest(ctx context.Context, endpoint string, form url.Values) (map[string]string, error) {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("oauth request failed: %s", strings.TrimSpace(string(body)))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for k, v := range data {
		switch val := v.(type) {
		case string:
			result[k] = val
		case float64:
			result[k] = strconv.FormatFloat(val, 'f', -1, 64)
		default:
			result[k] = fmt.Sprintf("%v", v)
		}
	}
	return result, nil
}

// populateOpenAIIDToken 解析OpenAI ID Token
// 从JWT中提取用户信息和组织信息
// 参数：
//   - cred: 账号凭证
//   - idToken: ID Token字符串
func (c *Core) populateOpenAIIDToken(cred *AccountCredentials, idToken string) {
	if idToken == "" {
		return
	}
	// 解析JWT（header.payload.signature）
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return
	}
	// 解码payload
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
	// 解析JSON
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return
	}
	// 提取邮箱
	if email, _ := claims["email"].(string); email != "" {
		cred.Email = email
	}
	// 提取API认证声明
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		// 提取ChatGPT账号ID
		if id, _ := authClaims["chatgpt_account_id"].(string); id != "" {
			cred.AccountID = id
		}
		// 提取套餐类型
		if plan, _ := authClaims["chatgpt_plan_type"].(string); plan != "" {
			cred.PlanType = plan
		}
		// 提取组织ID
		if oid, _ := authClaims["poid"].(string); oid != "" {
			cred.OrganizationID = oid
		}
	}
}

// geminiOAuthConfig 获取Gemini OAuth配置
// 根据OAuth类型返回对应的客户端配置
// 参数：
//   - oauthType: OAuth类型
//
// 返回：Gemini配置、回调URI和授权范围
func (c *Core) geminiOAuthConfig(oauthType string) (config.GeminiConfig, string, string) {
	effective := config.GeminiConfig{
		ClientID:            strings.TrimSpace(c.cfg.Gemini.ClientID),
		ClientSecret:        strings.TrimSpace(c.cfg.Gemini.ClientSecret),
		BuiltinClientSecret: strings.TrimSpace(c.cfg.Gemini.BuiltinClientSecret),
	}
	oauthType = strings.TrimSpace(oauthType)
	if oauthType == "" {
		oauthType = "code_assist"
	}
	isBuiltin := false
	if effective.ClientID == "" && effective.ClientSecret == "" {
		effective.ClientID = geminiBuiltinClientID
		effective.ClientSecret = effective.BuiltinClientSecret
		isBuiltin = true
	}
	redirectURI := geminiAIRedirectURI
	scopes := geminiCodeAssistScopes
	switch oauthType {
	case "ai_studio":
		if !isBuiltin {
			scopes = geminiAIStudioScopes
		}
	case "google_one", "code_assist":
		redirectURI = geminiCLIRedirectURI
	default:
		redirectURI = geminiCLIRedirectURI
	}
	if isBuiltin {
		redirectURI = geminiCLIRedirectURI
	}
	return effective, redirectURI, scopes
}

// pickAccount 选择一个可用的AI账号
// 使用负载均衡策略：优先选择优先级高、负载低的账号
// 参数：
//   - providerName: Provider名称
//   - modelName: 模型名称
//   - path: 请求路径
//
// 返回：选中的账号和错误
func (c *Core) pickAccount(providerName, modelName, path string) (model.Account, error) {
	c.accountCacheMu.RLock()
	accounts := c.accountCache[providerName]
	c.accountCacheMu.RUnlock()

	if len(accounts) == 0 {
		return model.Account{}, fmt.Errorf("no active %s account available", providerName)
	}
	// 获取Provider实现
	providerImpl, err := c.providers.Get(providerName)
	if err != nil {
		return model.Account{}, err
	}
	if !providerImpl.SupportsPath(path) {
		return model.Account{}, fmt.Errorf("provider %s does not support path %s", providerName, path)
	}
	candidates := make([]int, 0, len(accounts))
	for i := range accounts {
		if modelName != "" && !isModelAllowed(accounts[i].ModelScopeJSON, modelName) {
			continue
		}
		candidates = append(candidates, i)
	}
	if len(candidates) == 0 {
		return model.Account{}, fmt.Errorf("no active %s account available", providerName)
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := accounts[candidates[i]]
		right := accounts[candidates[j]]
		leftLoad := c.accountLoadCounter(left.ID).Load()
		rightLoad := c.accountLoadCounter(right.ID).Load()
		if leftLoad != rightLoad {
			return leftLoad < rightLoad
		}
		if left.Priority != right.Priority {
			return left.Priority > right.Priority
		}
		return left.ID < right.ID
	})
	for _, idx := range candidates {
		account := accounts[idx]
		counter := c.accountLoadCounter(account.ID)
		limit := account.ConcurrencyLimit
		if limit <= 0 {
			limit = 1
		}
		for {
			current := counter.Load()
			if current >= int64(limit) {
				break
			}
			if counter.CompareAndSwap(current, current+1) {
				return account, nil
			}
		}
	}
	return model.Account{}, fmt.Errorf("no active %s account available", providerName)
}

// releaseAccount 释放账号（减少负载计数）
// 参数：
//   - accountID: 账号ID
func (c *Core) releaseAccount(accountID uint64) {
	counter := c.accountLoadCounter(accountID)
	for {
		current := counter.Load()
		if current <= 0 {
			return
		}
		if counter.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func (c *Core) accountLoadCounter(accountID uint64) *atomic.Int64 {
	if counter, ok := c.accountLoads.Load(accountID); ok {
		return counter.(*atomic.Int64)
	}
	counter := &atomic.Int64{}
	actual, _ := c.accountLoads.LoadOrStore(accountID, counter)
	return actual.(*atomic.Int64)
}

// recordUsage 记录API使用量
// 参数：
//   - auth: 代理认证信息
//   - account: AI账号
//   - modelName: 模型名称
//   - endpoint: 端点
//   - inTokens: 输入token数
//   - outTokens: 输出token数
//
// 返回：错误
func (c *Core) recordUsage(auth *ProxyAuth, account *model.Account, modelName, endpoint string, inTokens, outTokens, cacheCreateTokens, cacheReadTokens int64) error {
	// 计算费用
	cost, err := c.calculateCost(account.Provider, modelName, auth.User.RatePercent, inTokens, outTokens, cacheCreateTokens, cacheReadTokens)
	if err != nil {
		return err
	}
	// 高频路径上只在需要扣费时同步写余额，避免把免费请求也变成用户行热点写
	now := time.Now().UnixMilli()
	entry := model.UsageLog{
		UserID:            auth.User.ID,
		APIKeyID:          auth.APIKey.ID,
		AccountID:         account.ID,
		Provider:          account.Provider,
		Model:             modelName,
		Endpoint:          endpoint,
		InputTokens:       inTokens,
		OutputTokens:      outTokens,
		CacheCreateTokens: cacheCreateTokens,
		CacheReadTokens:   cacheReadTokens,
		Cost:              cost,
		CreatedAtMS:       now,
	}
	if cost > 0 {
		result := c.db.Model(&model.User{}).
			Where("id = ? AND balance >= ?", auth.User.ID, cost).
			Updates(map[string]any{
				"balance":       gorm.Expr("balance - ?", cost),
				"updated_at_ms": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("insufficient balance")
		}
	}
	c.enqueueUserLastUsed(auth.User.ID, now)
	c.enqueueUsageLog(entry)
	return nil
}

func (c *Core) getCachedUserDashboard(userID uint64) (*UserDashboardData, bool) {
	c.dashboardMu.RLock()
	entry, ok := c.dashboardCache[userID]
	c.dashboardMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) || entry.data == nil {
		return nil, false
	}
	data := *entry.data
	return &data, true
}

func (c *Core) setCachedUserDashboard(userID uint64, data *UserDashboardData) {
	if data == nil {
		return
	}
	copyData := *data
	c.dashboardMu.Lock()
	c.dashboardCache[userID] = cachedUserDashboard{
		data:      &copyData,
		expiresAt: time.Now().Add(15 * time.Second),
	}
	c.dashboardMu.Unlock()
}

func (c *Core) invalidateUserDashboard(userID uint64) {
	c.dashboardMu.Lock()
	delete(c.dashboardCache, userID)
	c.dashboardMu.Unlock()
}

func (c *Core) getCachedStats() (map[string]any, bool) {
	c.statsMu.RLock()
	entry, ok := c.statsCache["global"]
	c.statsMu.RUnlock()
	if !ok || time.Now().After(entry.expiresAt) || entry.data == nil {
		return nil, false
	}
	copyData := make(map[string]any, len(entry.data))
	for k, v := range entry.data {
		copyData[k] = v
	}
	return copyData, true
}

func (c *Core) setCachedStats(data map[string]any) {
	if data == nil {
		return
	}
	copyData := make(map[string]any, len(data))
	for k, v := range data {
		copyData[k] = v
	}
	c.statsMu.Lock()
	c.statsCache["global"] = cachedStats{
		data:      copyData,
		expiresAt: time.Now().Add(10 * time.Second),
	}
	c.statsMu.Unlock()
}

func (c *Core) invalidateStats() {
	c.statsMu.Lock()
	delete(c.statsCache, "global")
	c.statsMu.Unlock()
}

func (c *Core) enqueueUserLastUsed(userID uint64, usedAt int64) {
	c.pendingUserMu.Lock()
	if prev, ok := c.pendingUsers[userID]; !ok || usedAt > prev {
		c.pendingUsers[userID] = usedAt
	}
	c.pendingUserMu.Unlock()
}

func (c *Core) enqueueUsageLog(item model.UsageLog) {
	var dropped int
	c.usageLogMu.Lock()
	if len(c.usageLogQueue) >= maxUsageLogQueueSize {
		dropped = len(c.usageLogQueue) - maxUsageLogQueueSize + 1
		if dropped > len(c.usageLogQueue) {
			dropped = len(c.usageLogQueue)
		}
		c.usageLogQueue = append([]model.UsageLog(nil), c.usageLogQueue[dropped:]...)
	}
	c.usageLogQueue = append(c.usageLogQueue, item)
	c.usageLogMu.Unlock()
	if dropped > 0 {
		c.recordError("usage.queue", "usage log queue overflow", fmt.Sprintf("dropped=%d", dropped))
	}
}

func (c *Core) flushUsageLogsLoop(ctx context.Context) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flushUsageLogsBatch()
			return
		case <-ticker.C:
			c.flushUsageLogsBatch()
		}
	}
}

func (c *Core) flushUsageLogsBatch() {
	c.usageLogMu.Lock()
	if len(c.usageLogQueue) == 0 {
		c.usageLogMu.Unlock()
		return
	}
	batchSize := 200
	if len(c.usageLogQueue) < batchSize {
		batchSize = len(c.usageLogQueue)
	}
	items := append([]model.UsageLog(nil), c.usageLogQueue[:batchSize]...)
	c.usageLogQueue = c.usageLogQueue[batchSize:]
	c.usageLogMu.Unlock()

	if err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.CreateInBatches(items, 100).Error; err != nil {
			return err
		}
		return c.upsertUsageAggregatesTx(tx, items)
	}); err != nil {
		c.recordError("usage.flush", "flush usage logs failed", err.Error())
		c.usageLogMu.Lock()
		c.usageLogQueue = append(items, c.usageLogQueue...)
		c.usageLogMu.Unlock()
	}
}

func (c *Core) upsertUsageAggregates(items []model.UsageLog) error {
	return c.upsertUsageAggregatesTx(c.db, items)
}

func (c *Core) upsertUsageAggregatesTx(tx *gorm.DB, items []model.UsageLog) error {
	if len(items) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	minuteAgg := map[string]*model.UserUsageMinute{}
	hourAgg := map[string]*model.UserUsageHour{}
	dayAgg := map[string]*model.UserUsageDay{}
	dayDimensionAgg := map[string]*model.UserUsageDayDimension{}

	for _, item := range items {
		minuteStart := alignMinuteStart(item.CreatedAtMS)
		hourStart := alignHourStart(item.CreatedAtMS)
		dayStart := alignDayStart(item.CreatedAtMS)

		minuteKey := fmt.Sprintf("%d:%d", item.UserID, minuteStart)
		if minuteAgg[minuteKey] == nil {
			minuteAgg[minuteKey] = &model.UserUsageMinute{UserID: item.UserID, BucketStartMS: minuteStart, CreatedAtMS: now, UpdatedAtMS: now}
		}
		accumulateMinute(minuteAgg[minuteKey], item, now)

		hourKey := fmt.Sprintf("%d:%d", item.UserID, hourStart)
		if hourAgg[hourKey] == nil {
			hourAgg[hourKey] = &model.UserUsageHour{UserID: item.UserID, BucketStartMS: hourStart, CreatedAtMS: now, UpdatedAtMS: now}
		}
		accumulateHour(hourAgg[hourKey], item, now)

		dayKey := fmt.Sprintf("%d:%d", item.UserID, dayStart)
		if dayAgg[dayKey] == nil {
			dayAgg[dayKey] = &model.UserUsageDay{UserID: item.UserID, BucketStartMS: dayStart, CreatedAtMS: now, UpdatedAtMS: now}
		}
		accumulateDay(dayAgg[dayKey], item, now)

		dayDimensionKey := fmt.Sprintf("%d:%d:%s:%s", item.UserID, dayStart, item.Provider, item.Model)
		if dayDimensionAgg[dayDimensionKey] == nil {
			dayDimensionAgg[dayDimensionKey] = &model.UserUsageDayDimension{
				UserID:        item.UserID,
				BucketStartMS: dayStart,
				Provider:      item.Provider,
				Model:         item.Model,
				CreatedAtMS:   now,
				UpdatedAtMS:   now,
			}
		}
		accumulateDayDimension(dayDimensionAgg[dayDimensionKey], item, now)
	}

	if err := upsertMinuteRows(tx, mapValuesMinute(minuteAgg)); err != nil {
		return err
	}
	if err := upsertHourRows(tx, mapValuesHour(hourAgg)); err != nil {
		return err
	}
	if err := upsertDayRows(tx, mapValuesDay(dayAgg)); err != nil {
		return err
	}
	return upsertDayDimensionRows(tx, mapValuesDayDimension(dayDimensionAgg))
}

func (c *Core) upsertMinuteRows(rows []model.UserUsageMinute) error {
	return upsertMinuteRows(c.db, rows)
}

func upsertMinuteRows(tx *gorm.DB, rows []model.UserUsageMinute) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "bucket_start_ms"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":       gorm.Expr("request_count + excluded.request_count"),
			"input_tokens":        gorm.Expr("input_tokens + excluded.input_tokens"),
			"output_tokens":       gorm.Expr("output_tokens + excluded.output_tokens"),
			"cache_create_tokens": gorm.Expr("cache_create_tokens + excluded.cache_create_tokens"),
			"cache_read_tokens":   gorm.Expr("cache_read_tokens + excluded.cache_read_tokens"),
			"cost":                gorm.Expr("cost + excluded.cost"),
			"updated_at_ms":       gorm.Expr("excluded.updated_at_ms"),
		}),
	}).Create(&rows).Error
}

func (c *Core) upsertHourRows(rows []model.UserUsageHour) error {
	return upsertHourRows(c.db, rows)
}

func upsertHourRows(tx *gorm.DB, rows []model.UserUsageHour) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "bucket_start_ms"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":       gorm.Expr("request_count + excluded.request_count"),
			"input_tokens":        gorm.Expr("input_tokens + excluded.input_tokens"),
			"output_tokens":       gorm.Expr("output_tokens + excluded.output_tokens"),
			"cache_create_tokens": gorm.Expr("cache_create_tokens + excluded.cache_create_tokens"),
			"cache_read_tokens":   gorm.Expr("cache_read_tokens + excluded.cache_read_tokens"),
			"cost":                gorm.Expr("cost + excluded.cost"),
			"updated_at_ms":       gorm.Expr("excluded.updated_at_ms"),
		}),
	}).Create(&rows).Error
}

func (c *Core) upsertDayRows(rows []model.UserUsageDay) error {
	return upsertDayRows(c.db, rows)
}

func upsertDayRows(tx *gorm.DB, rows []model.UserUsageDay) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "bucket_start_ms"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":       gorm.Expr("request_count + excluded.request_count"),
			"input_tokens":        gorm.Expr("input_tokens + excluded.input_tokens"),
			"output_tokens":       gorm.Expr("output_tokens + excluded.output_tokens"),
			"cache_create_tokens": gorm.Expr("cache_create_tokens + excluded.cache_create_tokens"),
			"cache_read_tokens":   gorm.Expr("cache_read_tokens + excluded.cache_read_tokens"),
			"cost":                gorm.Expr("cost + excluded.cost"),
			"updated_at_ms":       gorm.Expr("excluded.updated_at_ms"),
		}),
	}).Create(&rows).Error
}

func (c *Core) upsertDayDimensionRows(rows []model.UserUsageDayDimension) error {
	return upsertDayDimensionRows(c.db, rows)
}

func upsertDayDimensionRows(tx *gorm.DB, rows []model.UserUsageDayDimension) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "bucket_start_ms"}, {Name: "provider"}, {Name: "model"}},
		DoUpdates: clause.Assignments(map[string]any{
			"request_count":       gorm.Expr("request_count + excluded.request_count"),
			"input_tokens":        gorm.Expr("input_tokens + excluded.input_tokens"),
			"output_tokens":       gorm.Expr("output_tokens + excluded.output_tokens"),
			"cache_create_tokens": gorm.Expr("cache_create_tokens + excluded.cache_create_tokens"),
			"cache_read_tokens":   gorm.Expr("cache_read_tokens + excluded.cache_read_tokens"),
			"cost":                gorm.Expr("cost + excluded.cost"),
			"updated_at_ms":       gorm.Expr("excluded.updated_at_ms"),
		}),
	}).Create(&rows).Error
}

func alignMinuteStart(ts int64) int64 {
	const minuteMS = int64(time.Minute / time.Millisecond)
	return ts / minuteMS * minuteMS
}

func alignHourStart(ts int64) int64 {
	const hourMS = int64(time.Hour / time.Millisecond)
	return ts / hourMS * hourMS
}

func alignDayStart(ts int64) int64 {
	const dayMS = int64(24 * time.Hour / time.Millisecond)
	return ts / dayMS * dayMS
}

func accumulateMinute(dst *model.UserUsageMinute, item model.UsageLog, now int64) {
	dst.RequestCount++
	dst.InputTokens += item.InputTokens
	dst.OutputTokens += item.OutputTokens
	dst.CacheCreateTokens += item.CacheCreateTokens
	dst.CacheReadTokens += item.CacheReadTokens
	dst.Cost += item.Cost
	dst.UpdatedAtMS = now
}

func accumulateHour(dst *model.UserUsageHour, item model.UsageLog, now int64) {
	dst.RequestCount++
	dst.InputTokens += item.InputTokens
	dst.OutputTokens += item.OutputTokens
	dst.CacheCreateTokens += item.CacheCreateTokens
	dst.CacheReadTokens += item.CacheReadTokens
	dst.Cost += item.Cost
	dst.UpdatedAtMS = now
}

func accumulateDay(dst *model.UserUsageDay, item model.UsageLog, now int64) {
	dst.RequestCount++
	dst.InputTokens += item.InputTokens
	dst.OutputTokens += item.OutputTokens
	dst.CacheCreateTokens += item.CacheCreateTokens
	dst.CacheReadTokens += item.CacheReadTokens
	dst.Cost += item.Cost
	dst.UpdatedAtMS = now
}

func accumulateDayDimension(dst *model.UserUsageDayDimension, item model.UsageLog, now int64) {
	dst.RequestCount++
	dst.InputTokens += item.InputTokens
	dst.OutputTokens += item.OutputTokens
	dst.CacheCreateTokens += item.CacheCreateTokens
	dst.CacheReadTokens += item.CacheReadTokens
	dst.Cost += item.Cost
	dst.UpdatedAtMS = now
}

func mapValuesMinute(src map[string]*model.UserUsageMinute) []model.UserUsageMinute {
	rows := make([]model.UserUsageMinute, 0, len(src))
	for _, item := range src {
		rows = append(rows, *item)
	}
	return rows
}

func mapValuesHour(src map[string]*model.UserUsageHour) []model.UserUsageHour {
	rows := make([]model.UserUsageHour, 0, len(src))
	for _, item := range src {
		rows = append(rows, *item)
	}
	return rows
}

func mapValuesDay(src map[string]*model.UserUsageDay) []model.UserUsageDay {
	rows := make([]model.UserUsageDay, 0, len(src))
	for _, item := range src {
		rows = append(rows, *item)
	}
	return rows
}

func mapValuesDayDimension(src map[string]*model.UserUsageDayDimension) []model.UserUsageDayDimension {
	rows := make([]model.UserUsageDayDimension, 0, len(src))
	for _, item := range src {
		rows = append(rows, *item)
	}
	return rows
}

// calculateCost 计算费用
// 根据模型价格和用户费率计算实际费用
// 参数：
//   - providerName: Provider名称
//   - modelName: 模型名称
//   - ratePercent: 用户费率（百分比）
//   - inTokens: 输入token数
//   - outTokens: 输出token数
//
// 返回：计算出的费用和错误
func (c *Core) calculateCost(providerName, modelName string, ratePercent int, inTokens, outTokens, cacheCreateTokens, cacheReadTokens int64) (int64, error) {
	c.priceCacheMu.RLock()
	price, ok := c.priceCache[providerName+":"+modelName]
	c.priceCacheMu.RUnlock()

	if !ok {
		return 0, nil
	}
	// 计算基础费用：输入价格*输入token + 输出价格*输出token + 缓存价格，结果除以1000（因为价格单位是CNY_1E4，即万分之）
	base := (inTokens*price.InputPrice + outTokens*price.OutputPrice + cacheCreateTokens*price.CacheCreatePrice + cacheReadTokens*price.CacheReadPrice + 999) / 1000
	// 应用用户费率并按分母100做四舍五入，避免持续向下截断。
	return (int64(ratePercent)*base + 50) / 100, nil
}

func (c *Core) parseCacheUsage(providerImpl provider.Provider, body []byte) (int64, int64) {
	if parser, ok := providerImpl.(provider.CacheUsageParser); ok {
		if create, read, ok := parser.ParseCacheUsage(body); ok {
			return create, read
		}
	}
	return 0, 0
}

func (c *Core) recordError(scope, message, detail string) {
	// 记录或更新错误日志
	now := time.Now().UnixMilli()
	var item model.ErrorLog
	err := c.db.Where("scope = ? AND message = ?", scope, message).First(&item).Error
	if err == nil {
		// 已存在，增加计数
		_ = c.db.Model(&item).Updates(map[string]any{
			"count":           gorm.Expr("count + 1"),
			"detail":          detail,
			"last_seen_at_ms": now,
			"updated_at_ms":   now,
		}).Error
		return
	}
	// 创建新记录
	_ = c.db.Create(&model.ErrorLog{
		Scope:        scope,
		Message:      message,
		Detail:       detail,
		Count:        1,
		LastSeenAtMS: now,
	}).Error
}

// normalizeProvider 标准化Provider名称
// 转换为小写并去除空格
func normalizeProvider(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "openai", "claude", "gemini", "antigravity":
		return name
	default:
		return name
	}
}

func deriveScopedKey(base []byte, scope string) []byte {
	mac := hmac.New(sha256.New, base)
	mac.Write([]byte(scope))
	return mac.Sum(nil)
}

// normalizeStrings 标准化字符串数组
// 去除空格和空字符串
func normalizeStrings(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// normalizeJSON 标准化JSON字符串
// 返回原始JSON或默认值
func normalizeJSON(raw json.RawMessage, fallback string) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fallback
	}
	return string(raw)
}

// redactCredentialsForView 脱敏凭证信息
// 返回用于展示的凭证映射（隐藏敏感信息）
func redactCredentialsForView(account *model.Account, cred *AccountCredentials) map[string]any {
	if cred == nil {
		return nil
	}
	data := map[string]any{
		"email":              cred.Email,
		"expires_at_ms":      cred.ExpiresAtMS,
		"project_id":         cred.ProjectID,
		"oauth_type":         cred.OAuthType,
		"organization_id":    cred.OrganizationID,
		"account_id":         cred.AccountID,
		"plan_type":          cred.PlanType,
		"subscription_until": cred.SubscriptionUntil,
	}
	// 脱敏API密钥
	if cred.APIKey != "" {
		data["api_key_masked"] = maskSecret(cred.APIKey)
	}
	// 脱敏刷新令牌
	if cred.RefreshToken != "" {
		data["refresh_token_masked"] = maskSecret(cred.RefreshToken)
	}
	// 脱敏访问令牌
	if cred.AccessToken != "" {
		data["access_token_masked"] = maskSecret(cred.AccessToken)
	}
	// 脱敏设置令牌
	if cred.SetupToken != "" {
		data["setup_token_masked"] = maskSecret(cred.SetupToken)
	}
	// OAuth类型显示额外信息
	if account.AuthType == "oauth" {
		data["token_url"] = cred.TokenURL
		data["redirect_uri"] = cred.RedirectURI
	}
	return data
}

// detectRoute 检测路由并提取模型信息
// 根据请求路径和请求体判断Provider和模型
func detectRoute(path string, body []byte) (modelName string, stream bool, providerName string, err error) {
	switch {
	// OpenAI兼容接口
	case strings.HasPrefix(path, "/v1/chat/completions"), strings.HasPrefix(path, "/v1/responses"), strings.HasPrefix(path, "/v1/embeddings"):
		providerName = "openai"
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if unmarshalErr := json.Unmarshal(body, &payload); unmarshalErr == nil {
			modelName = payload.Model
			stream = payload.Stream
		}
		return
	// Claude兼容接口
	case strings.HasPrefix(path, "/v1/messages"), strings.HasPrefix(path, "/v1/messages/count_tokens"):
		providerName = "claude"
		var payload struct {
			Model  string `json:"model"`
			Stream bool   `json:"stream"`
		}
		if unmarshalErr := json.Unmarshal(body, &payload); unmarshalErr == nil {
			modelName = payload.Model
			stream = payload.Stream
		}
		return
	// Gemini兼容接口
	case strings.HasPrefix(path, "/v1beta/models/"), strings.HasPrefix(path, "/v1/models/"):
		providerName = "gemini"
		modelName = parseGeminiModelFromPath(path)
		stream = strings.Contains(path, ":streamGenerateContent")
		return
	// Antigravity接口
	case strings.HasPrefix(path, "/v1internal:"):
		providerName = "antigravity"
		var payload struct {
			Model string `json:"model"`
		}
		if unmarshalErr := json.Unmarshal(body, &payload); unmarshalErr == nil {
			modelName = payload.Model
		}
		stream = strings.Contains(path, ":stream")
		return
	default:
		err = fmt.Errorf("unsupported gateway path %s", path)
		return
	}
}

// parseGeminiModelFromPath 从路径解析Gemini模型名
func parseGeminiModelFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		if parts[i] == "models" && i+1 < len(parts) {
			return strings.Split(parts[i+1], ":")[0]
		}
	}
	return ""
}

// defaultString 返回默认值如果为空
func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

// defaultInt 返回默认值如果是0
func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

// defaultInt64 返回默认值如果是0
func defaultInt64(value, fallback int64) int64 {
	if value == 0 {
		return fallback
	}
	return value
}

// isModelAllowed 检查模型是否在允许列表中
// 空列表或包含*表示允许所有
func isModelAllowed(raw, modelName string) bool {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) == 0 || modelName == "" {
		return true
	}
	return slices.Contains(items, "*") || slices.Contains(items, modelName)
}

func parseAllowedModelsSet(raw string) map[string]struct{} {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		set[item] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

func isModelAllowedSet(allowed map[string]struct{}, modelName string) bool {
	if modelName == "" || len(allowed) == 0 {
		return true
	}
	if _, ok := allowed["*"]; ok {
		return true
	}
	_, ok := allowed[modelName]
	return ok
}

func (c *Core) isFreeModel(providerName, modelName string, ratePercent int) bool {
	if strings.TrimSpace(modelName) == "" {
		return false
	}
	cost, err := c.calculateCost(providerName, modelName, ratePercent, 1, 1, 0, 0)
	return err == nil && cost == 0
}

// copyHeaders 复制HTTP请求头
// 跳过Host和Content-Length
func copyHeaders(dst, src http.Header) {
	for k, values := range src {
		if strings.EqualFold(k, "Host") || strings.EqualFold(k, "Content-Length") {
			continue
		}
		for _, value := range values {
			dst.Add(k, value)
		}
	}
}

// cloneHeader 克隆HTTP头
func cloneHeader(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[k] = copied
	}
	return dst
}

// cloneBody 克隆响应体
func cloneBody(body []byte) []byte {
	if len(body) == 0 {
		return nil
	}
	out := make([]byte, len(body))
	copy(out, body)
	return out
}

// cacheKey 生成缓存键
// 判断请求是否可缓存，返回缓存键
// 参数：
//   - providerName: Provider名称
//   - path: 请求路径
//   - rawQuery: 原始查询字符串
//   - body: 请求体
//   - stream: 是否流式请求
//
// 返回：缓存键和是否可缓存
func (c *Core) cacheKey(userID uint64, providerName, path, rawQuery string, body []byte, stream bool) (string, bool) {
	if stream || providerName == "antigravity" {
		return "", false
	}
	raw := fmt.Sprintf("%d|%s|%s?%s|", userID, providerName, path, rawQuery)
	sum := sha256.Sum256(append([]byte(raw), body...))
	return hex.EncodeToString(sum[:]), true
}

// getCachedResponse 获取缓存的响应
// 参数：
//   - key: 缓存键
//
// 返回：缓存的HTTP响应和是否存在
func (c *Core) getCachedResponse(key string) (*http.Response, bool) {
	if key == "" {
		return nil, false
	}
	// 读取缓存
	c.cacheMu.RLock()
	item, ok := c.cacheItems[key]
	c.cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	// 检查是否过期
	if time.Now().After(item.ExpiresAt) {
		c.cacheMu.Lock()
		delete(c.cacheItems, key)
		c.cacheMu.Unlock()
		return nil, false
	}
	// 构造响应
	resp := &http.Response{
		StatusCode: item.StatusCode,
		Header:     cloneHeader(item.Header),
		Body:       io.NopCloser(bytes.NewReader(item.Body)),
	}
	return resp, true
}

// putCachedResponse 保存响应到缓存
// 参数：
//   - key: 缓存键
//   - statusCode: HTTP状态码
//   - header: 响应头
//   - body: 响应体
//   - ttl: 过期时间
const maxCacheItems = 1000

func (c *Core) putCachedResponse(key string, statusCode int, header http.Header, body []byte, ttl time.Duration) {
	if key == "" || ttl <= 0 {
		return
	}
	c.cacheMu.Lock()
	if len(c.cacheItems) >= maxCacheItems {
		oldestKey := ""
		var oldestTime time.Time
		for k, v := range c.cacheItems {
			if oldestKey == "" || v.ExpiresAt.Before(oldestTime) {
				oldestTime = v.ExpiresAt
				oldestKey = k
			}
		}
		if oldestKey != "" {
			delete(c.cacheItems, oldestKey)
		}
	}
	c.cacheItems[key] = cachedProxyResponse{
		StatusCode: statusCode,
		Header:     cloneHeader(header),
		Body:       cloneBody(body),
		ExpiresAt:  time.Now().Add(ttl),
	}
	c.cacheMu.Unlock()
}

// parseExpires 解析过期时间字符串
// 返回秒数
func parseExpires(raw string) int64 {
	if raw == "" {
		return 0
	}
	var n int64
	fmt.Sscanf(raw, "%d", &n)
	return n
}

// randomHex 生成随机十六进制字符串
func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

// randomState 生成随机state参数
func randomState() string {
	return base64.RawURLEncoding.EncodeToString(randomBytes(32))
}

// randomCodeVerifier 生成随机PKCE代码验证器
// 不同Provider有不同的长度要求
func randomCodeVerifier(providerName string) string {
	size := 32
	if providerName == "openai" {
		size = 64
		return hex.EncodeToString(randomBytes(size))
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes(size))
}

// randomBytes 生成随机字节数组
func randomBytes(n int) []byte {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return buf
}

// pkceChallenge 生成PKCE代码挑战
// 使用S256方法
func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// defaultRedirectURI 获取默认回调URI
// 根据Provider和OAuth类型返回默认回调地址
func defaultRedirectURI(providerName, oauthType string) string {
	switch providerName {
	case "openai":
		return openAIDefaultRedirect
	case "claude":
		return claudeRedirectURI
	case "gemini":
		if oauthType == "ai_studio" {
			return geminiAIRedirectURI
		}
		return geminiCLIRedirectURI
	case "antigravity":
		return antigravityRedirectURI
	default:
		return ""
	}
}

// maskSecret 脱敏处理
// 将长字符串脱敏显示，只保留首尾部分
func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 10 {
		return value
	}
	return value[:6] + "..." + value[len(value)-4:]
}

// mustJSON 强制转换为JSON字符串
// 忽略错误
func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

// sortedKeys 获取排序后的键列表
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// hashPassword 密码哈希
// 使用SHA256和随机盐
func hashPassword(password string) (string, string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 6 {
		return "", "", fmt.Errorf("password must be at least 6 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return "", "$bcrypt$" + string(hash), nil
}

func verifyPassword(salt, expectedHash, password string) bool {
	if expectedHash == "" {
		return false
	}
	if strings.HasPrefix(expectedHash, "$bcrypt$") {
		bcryptHash := strings.TrimPrefix(expectedHash, "$bcrypt$")
		return bcrypt.CompareHashAndPassword([]byte(bcryptHash), []byte(password)) == nil
	}
	if strings.HasPrefix(expectedHash, "$hmac$") {
		if salt == "" {
			return false
		}
		mac := hmac.New(sha256.New, []byte(salt))
		mac.Write([]byte(password))
		expected := "$hmac$" + hex.EncodeToString(mac.Sum(nil))
		return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(expected)) == 1
	}
	if salt == "" {
		return false
	}
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(hex.EncodeToString(sum[:]))) == 1
}

// issueUserAuth 发放用户认证信息
// 生成访问令牌和刷新令牌
// 参数：
//   - user: 用户
//
// 返回：用户认证信息和错误
func (c *Core) issueUserAuth(user *model.User) (*UserAuth, error) {
	// 生成访问令牌（2小时有效期）
	accessToken, err := c.signUserToken(user.ID, user.TokenVersion, "access", time.Now().Add(2*time.Hour))
	if err != nil {
		return nil, err
	}
	// 生成刷新令牌（30天有效期）
	refreshToken, err := c.signUserToken(user.ID, user.TokenVersion, "refresh", time.Now().Add(30*24*time.Hour))
	if err != nil {
		return nil, err
	}
	return &UserAuth{
		User:         *user,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64((2 * time.Hour) / time.Second),
	}, nil
}

// signUserToken 签名用户令牌
// 使用RSA签名生成JWT
// 参数：
//   - userID: 用户ID
//   - tokenVersion: 令牌版本
//   - kind: 令牌类型（access/refresh）
//   - expiresAt: 过期时间
//
// 返回：签名的令牌和错误
func (c *Core) signUserToken(userID uint64, tokenVersion int64, kind string, expiresAt time.Time) (string, error) {
	claims := userTokenClaims{
		UserID:       userID,
		TokenVersion: tokenVersion,
		ExpiresAtMS:  expiresAt.UnixMilli(),
		Kind:         kind,
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	rawPayload := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, c.tokenSignKey)
	mac.Write([]byte(rawPayload))
	sig := mac.Sum(nil)
	return "v2." + rawPayload + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// parseUserToken 解析用户令牌
// 验证并解析令牌声明
// 参数：
//   - token: 待解析的令牌
//   - expectedKind: 期望的令牌类型
//
// 返回：令牌声明和错误
func (c *Core) parseUserToken(token, expectedKind string) (*userTokenClaims, error) {
	token = strings.TrimSpace(token)
	var payloadStr, sigStr string
	if strings.HasPrefix(token, "v2.") {
		parts := strings.Split(token[3:], ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid token")
		}
		payloadStr, sigStr = parts[0], parts[1]
		mac := hmac.New(sha256.New, c.tokenSignKey)
		mac.Write([]byte(payloadStr))
		if subtle.ConstantTimeCompare([]byte(sigStr), []byte(base64.RawURLEncoding.EncodeToString(mac.Sum(nil)))) != 1 {
			return nil, fmt.Errorf("invalid token signature")
		}
	} else {
		parts := strings.Split(token, ".")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid token")
		}
		payloadStr, sigStr = parts[0], parts[1]
		return nil, fmt.Errorf("legacy token format is no longer supported")
	}
	payload, err := base64.RawURLEncoding.DecodeString(payloadStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}
	var claims userTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}
	if claims.Kind != expectedKind {
		return nil, fmt.Errorf("invalid token type")
	}
	if claims.ExpiresAtMS <= time.Now().UnixMilli() {
		return nil, fmt.Errorf("token expired")
	}
	return &claims, nil
}

func (c *Core) cacheCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.cacheMu.Lock()
			now := time.Now()
			for k, v := range c.cacheItems {
				if now.After(v.ExpiresAt) {
					delete(c.cacheItems, k)
				}
			}
			c.cacheMu.Unlock()
		}
	}
}

func (c *Core) oauthSessionCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expiry := time.Now().Add(-1 * time.Hour).UnixMilli()
			c.db.Where("expires_at_ms > 0 AND expires_at_ms < ?", expiry).Delete(&model.OAuthSession{})
		}
	}
}

func isValidEmail(email string) bool {
	at := strings.Index(email, "@")
	if at < 1 || at == len(email)-1 {
		return false
	}
	dot := strings.LastIndex(email[at+1:], ".")
	return dot >= 1
}

type UpdateModelPriceInput struct {
	InputPrice       *int64 `json:"input_price"`
	OutputPrice      *int64 `json:"output_price"`
	CacheCreatePrice *int64 `json:"cache_create_price"`
	CacheReadPrice   *int64 `json:"cache_read_price"`
	Status           string `json:"status"`
}

type UpdateAnnouncementInput struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	Status        string `json:"status"`
	PublishedAtMS int64  `json:"published_at_ms"`
}

type UpdateCouponInput struct {
	Status      string `json:"status"`
	MaxUses     *int   `json:"max_uses"`
	ExpiresAtMS *int64 `json:"expires_at_ms"`
}

func (c *Core) ListPublishedAnnouncements() ([]model.Announcement, error) {
	var items []model.Announcement
	now := time.Now().UnixMilli()
	err := c.db.Where("status = ? AND published_at_ms <= ?", "published", now).
		Order("id desc").Limit(20).Find(&items).Error
	return items, err
}

func (c *Core) DeleteModelPrice(id uint64) error {
	err := c.db.Delete(&model.ModelPrice{}, id).Error
	if err == nil {
		c.reloadPriceCache()
	}
	return err
}

func (c *Core) UpdateModelPrice(id uint64, in UpdateModelPriceInput) error {
	updates := map[string]any{}
	if in.InputPrice != nil {
		updates["input_price"] = *in.InputPrice
	}
	if in.OutputPrice != nil {
		updates["output_price"] = *in.OutputPrice
	}
	if in.CacheCreatePrice != nil {
		updates["cache_create_price"] = *in.CacheCreatePrice
	}
	if in.CacheReadPrice != nil {
		updates["cache_read_price"] = *in.CacheReadPrice
	}
	if in.Status != "" {
		updates["status"] = in.Status
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	result := c.db.Model(&model.ModelPrice{}).Where("id = ?", id).Updates(updates)
	if result.RowsAffected == 0 {
		return fmt.Errorf("model price not found")
	}
	if result.Error == nil {
		c.reloadPriceCache()
	}
	return result.Error
}

func (c *Core) DeleteAnnouncement(id uint64) error {
	return c.db.Delete(&model.Announcement{}, id).Error
}

func (c *Core) UpdateAnnouncement(id uint64, in UpdateAnnouncementInput) error {
	updates := map[string]any{}
	if in.Title != "" {
		updates["title"] = in.Title
	}
	if in.Content != "" {
		updates["content"] = in.Content
	}
	if in.Status != "" {
		updates["status"] = in.Status
	}
	if in.PublishedAtMS > 0 {
		updates["published_at_ms"] = in.PublishedAtMS
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	result := c.db.Model(&model.Announcement{}).Where("id = ?", id).Updates(updates)
	if result.RowsAffected == 0 {
		return fmt.Errorf("announcement not found")
	}
	return result.Error
}

func (c *Core) DeleteCoupon(id uint64) error {
	return c.db.Delete(&model.Coupon{}, id).Error
}

func (c *Core) UpdateCoupon(id uint64, in UpdateCouponInput) error {
	updates := map[string]any{}
	if in.Status != "" {
		updates["status"] = in.Status
	}
	if in.MaxUses != nil {
		updates["max_uses"] = *in.MaxUses
	}
	if in.ExpiresAtMS != nil {
		updates["expires_at_ms"] = *in.ExpiresAtMS
	}
	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}
	result := c.db.Model(&model.Coupon{}).Where("id = ?", id).Updates(updates)
	if result.RowsAffected == 0 {
		return fmt.Errorf("coupon not found")
	}
	return result.Error
}

func (c *Core) DeleteErrorLog(id uint64) error {
	return c.db.Delete(&model.ErrorLog{}, id).Error
}

func (c *Core) cacheRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.reloadAccountCache()
			c.reloadPriceCache()
		}
	}
}

func (c *Core) dataCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-30 * 24 * time.Hour).UnixMilli()
			c.db.Where("created_at_ms < ?", cutoff).Delete(&model.UsageLog{})
			c.db.Where("snapshot_at_ms < ?", cutoff).Delete(&model.SystemMetric{})
			minuteCutoff := time.Now().Add(-24 * time.Hour).UnixMilli()
			hourCutoff := time.Now().Add(-90 * 24 * time.Hour).UnixMilli()
			dayCutoff := time.Now().Add(-365 * 24 * time.Hour).UnixMilli()
			c.db.Where("bucket_start_ms < ?", minuteCutoff).Delete(&model.UserUsageMinute{})
			c.db.Where("bucket_start_ms < ?", hourCutoff).Delete(&model.UserUsageHour{})
			c.db.Where("bucket_start_ms < ?", dayCutoff).Delete(&model.UserUsageDay{})
			c.db.Where("bucket_start_ms < ?", dayCutoff).Delete(&model.UserUsageDayDimension{})
		}
	}
}

func (c *Core) flushKeyUpdatesLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flushPendingKeys()
			return
		case <-ticker.C:
			c.flushPendingKeys()
		}
	}
}

func (c *Core) flushUserUpdatesLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flushPendingUsers()
			return
		case <-ticker.C:
			c.flushPendingUsers()
		}
	}
}

func (c *Core) flushPendingKeys() {
	c.pendingKeyMu.Lock()
	pending := c.pendingKeys
	c.pendingKeys = map[uint64]int64{}
	c.pendingKeyMu.Unlock()

	if len(pending) == 0 {
		return
	}

	c.flushPendingLastUsed("api_keys", pending)
}

func (c *Core) flushPendingUsers() {
	c.pendingUserMu.Lock()
	pending := c.pendingUsers
	c.pendingUsers = map[uint64]int64{}
	c.pendingUserMu.Unlock()

	if len(pending) == 0 {
		return
	}

	c.flushPendingLastUsed("users", pending)
}

func (c *Core) flushPendingLastUsed(table string, pending map[uint64]int64) {
	const batchSize = 300
	now := time.Now().UnixMilli()
	switch table {
	case "users", "api_keys":
	default:
		return
	}

	ids := make([]uint64, 0, len(pending))
	for id := range pending {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	for start := 0; start < len(ids); start += batchSize {
		end := min(start+batchSize, len(ids))
		batchIDs := ids[start:end]

		var sql strings.Builder
		sql.Grow(128 + len(batchIDs)*24)
		sql.WriteString("UPDATE ")
		sql.WriteString(table)
		sql.WriteString(" SET last_used_at_ms = CASE id")

		args := make([]any, 0, len(batchIDs)*3+1)
		for _, id := range batchIDs {
			sql.WriteString(" WHEN ? THEN ?")
			args = append(args, id, pending[id])
		}

		sql.WriteString(" ELSE last_used_at_ms END, updated_at_ms = ? WHERE id IN (")
		args = append(args, now)
		for i, id := range batchIDs {
			if i > 0 {
				sql.WriteString(",")
			}
			sql.WriteString("?")
			args = append(args, id)
		}
		sql.WriteString(")")

		_ = c.db.Table(table).Exec(sql.String(), args...).Error
	}
}
