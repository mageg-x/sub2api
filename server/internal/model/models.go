package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表
type User struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                         // 用户ID，自增主键
	Email             string `gorm:"uniqueIndex;size:200;not null" json:"email"`                 // 邮箱，唯一索引
	Name              string `gorm:"size:100;not null" json:"name"`                              // 用户名
	PasswordSalt      string `gorm:"size:64;not null;default:''" json:"-"`                       // 密码盐值，不序列化到JSON
	PasswordHash      string `gorm:"size:128;not null;default:''" json:"-"`                      // 密码哈希，不序列化到JSON
	Role              string `gorm:"size:20;not null;default:user" json:"role"`                  // 角色（admin/user）
	Status            string `gorm:"size:20;not null;default:active" json:"status"`              // 状态（active/disabled）
	TokenVersion      int64  `gorm:"not null;default:1" json:"-"`                                // 令牌版本，用于强制登出
	Balance           int64  `gorm:"not null;default:0" json:"balance"`                          // 余额（单位：万分之CNY）
	RatePercent       int    `gorm:"not null;default:100" json:"rate_percent"`                   // 费率百分比（100=原价）
	AllowedModelsJSON string `gorm:"type:text;not null;default:'[]'" json:"allowed_models_json"` // 允许使用的模型列表JSON
	MetadataJSON      string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`       // 扩展元数据JSON
	LastLoginAtMS     int64  `gorm:"not null;default:0" json:"last_login_at_ms"`                 // 最后登录时间（毫秒）
	LastUsedAtMS      int64  `gorm:"not null;default:0" json:"last_used_at_ms"`                  // 最后使用时间（毫秒）
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                              // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                              // 更新时间（毫秒）
}

// APIKey API密钥表
type APIKey struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                         // 密钥ID，自增主键
	UserID            uint64 `gorm:"index;not null" json:"user_id"`                              // 所属用户ID
	Name              string `gorm:"size:100;not null" json:"name"`                              // 密钥名称
	Secret            string `gorm:"uniqueIndex;size:120;not null" json:"secret"`                // 密钥字符串，唯一索引
	Status            string `gorm:"size:20;not null;default:active" json:"status"`              // 状态（active/disabled）
	AllowedModelsJSON string `gorm:"type:text;not null;default:'[]'" json:"allowed_models_json"` // 允许使用的模型列表JSON
	ExpiresAtMS       int64  `gorm:"not null;default:0" json:"expires_at_ms"`                    // 过期时间（毫秒，0=永不过期）
	LastUsedAtMS      int64  `gorm:"not null;default:0" json:"last_used_at_ms"`                  // 最后使用时间（毫秒）
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                              // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                              // 更新时间（毫秒）
}

// Account AI账号表
type Account struct {
	ID                   uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                                                                                    // 账号ID，自增主键
	Provider             string `gorm:"index:idx_accounts_provider_status,priority:1;size:40;not null" json:"provider"`                                                        // AI提供商，复合索引(provider+status)
	Name                 string `gorm:"size:120;not null" json:"name"`                                                                                                         // 账号名称
	AuthType             string `gorm:"index:idx_accounts_auth_status,priority:1;size:40;not null" json:"auth_type"`                                                           // 认证类型(apikey/oauth)，复合索引(auth_type+status)
	BaseURL              string `gorm:"size:300;not null" json:"base_url"`                                                                                                     // API基础URL
	ModelScopeJSON       string `gorm:"type:text;not null;default:'[]'" json:"model_scope_json"`                                                                               // 支持的模型范围JSON
	CredentialsEncrypted string `gorm:"type:text;not null" json:"-"`                                                                                                           // 加密后的凭证，不序列化到JSON
	Status               string `gorm:"index:idx_accounts_auth_status,priority:2;index:idx_accounts_provider_status,priority:2;size:20;not null;default:active" json:"status"` // 状态（active/disabled）
	Priority             int    `gorm:"not null;default:100" json:"priority"`                                                                                                  // 优先级（越大越优先）
	ConcurrencyLimit     int    `gorm:"not null;default:4" json:"concurrency_limit"`                                                                                           // 并发限制
	ExpiresAtMS          int64  `gorm:"not null;default:0" json:"expires_at_ms"`                                                                                               // 凭证过期时间（毫秒）
	LastRefreshedAtMS    int64  `gorm:"not null;default:0" json:"last_refreshed_at_ms"`                                                                                        // 最后刷新时间（毫秒）
	MetadataJSON         string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`                                                                                  // 扩展元数据JSON
	CreatedAtMS          int64  `gorm:"not null" json:"created_at_ms"`                                                                                                         // 创建时间（毫秒）
	UpdatedAtMS          int64  `gorm:"not null" json:"updated_at_ms"`                                                                                                         // 更新时间（毫秒）
}

// ModelPrice 模型价格表
type ModelPrice struct {
	ID               uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                                  // 价格ID，自增主键
	Provider         string `gorm:"index:idx_model_prices_pms,priority:1;size:40;not null" json:"provider"`              // AI提供商，复合索引(provider+model+status)
	Model            string `gorm:"index:idx_model_prices_pms,priority:2;size:120;not null" json:"model"`                // 模型名称
	InputPrice       int64  `gorm:"not null" json:"input_price"`                                                         // 输入价格
	OutputPrice      int64  `gorm:"not null" json:"output_price"`                                                        // 输出价格
	CacheCreatePrice int64  `gorm:"not null;default:0" json:"cache_create_price"`                                        // 缓存创建价格
	CacheReadPrice   int64  `gorm:"not null;default:0" json:"cache_read_price"`                                          // 缓存读取价格
	Currency         string `gorm:"size:20;not null;default:CNY_1E4" json:"currency"`                                    // 货币单位（CNY_1E4=万分之人民币）
	Status           string `gorm:"index:idx_model_prices_pms,priority:3;size:20;not null;default:active" json:"status"` // 状态（active/disabled）
	CreatedAtMS      int64  `gorm:"not null" json:"created_at_ms"`                                                       // 创建时间（毫秒）
	UpdatedAtMS      int64  `gorm:"not null" json:"updated_at_ms"`                                                       // 更新时间（毫秒）
}

// PaymentOrder 支付订单表
type PaymentOrder struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                   // 订单ID，自增主键
	UserID          uint64 `gorm:"index;not null" json:"user_id"`                        // 用户ID
	Provider        string `gorm:"size:40;not null" json:"provider"`                     // 支付提供商
	OutTradeNo      string `gorm:"uniqueIndex;size:80;not null" json:"out_trade_no"`     // 商户订单号，唯一索引
	ProviderTradeNo string `gorm:"size:80;not null;default:''" json:"provider_trade_no"` // 支付提供商交易号
	Subject         string `gorm:"size:120;not null" json:"subject"`                     // 订单标题
	Status          string `gorm:"index;size:20;not null;default:PENDING" json:"status"` // 订单状态（PENDING/PAID/FAILED）
	Amount          int64  `gorm:"not null" json:"amount"`                               // 订单金额
	CreditedAmount  int64  `gorm:"not null" json:"credited_amount"`                      // 已充值金额
	RefundedAmount  int64  `gorm:"not null;default:0" json:"refunded_amount"`            // 已退款金额
	MetadataJSON    string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"` // 扩展元数据JSON
	NotifiedAtMS    int64  `gorm:"not null;default:0" json:"notified_at_ms"`             // 回调通知时间（毫秒）
	CreatedAtMS     int64  `gorm:"not null" json:"created_at_ms"`                        // 创建时间（毫秒）
	UpdatedAtMS     int64  `gorm:"not null" json:"updated_at_ms"`                        // 更新时间（毫秒）
}

// Coupon 优惠券表
type Coupon struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                    // 优惠券ID，自增主键
	Code         string `gorm:"uniqueIndex;size:80;not null" json:"code"`              // 优惠码，唯一索引
	Kind         string `gorm:"size:20;not null" json:"kind"`                          // 类型（balance/percent）
	Amount       int64  `gorm:"not null" json:"amount"`                                // 金额/折扣
	MaxUses      int    `gorm:"not null;default:1" json:"max_uses"`                    // 最大使用次数
	UsedCount    int    `gorm:"not null;default:0" json:"used_count"`                  // 已使用次数
	UsageLogJSON string `gorm:"type:text;not null;default:'[]'" json:"usage_log_json"` // 使用记录JSON
	ExpiresAtMS  int64  `gorm:"not null;default:0" json:"expires_at_ms"`               // 过期时间（毫秒）
	Status       string `gorm:"size:20;not null;default:active" json:"status"`         // 状态（active/disabled）
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`                         // 创建时间（毫秒）
	UpdatedAtMS  int64  `gorm:"not null" json:"updated_at_ms"`                         // 更新时间（毫秒）
}

// Announcement 公告表
type Announcement struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`           // 公告ID，自增主键
	Title         string `gorm:"size:200;not null" json:"title"`               // 标题
	Content       string `gorm:"type:text;not null" json:"content"`            // 内容
	ReadCount     int64  `gorm:"not null;default:0" json:"read_count"`         // 阅读次数
	PublishedAtMS int64  `gorm:"not null;default:0" json:"published_at_ms"`    // 发布时间（毫秒）
	Status        string `gorm:"size:20;not null;default:draft" json:"status"` // 状态（draft/published）
	CreatedAtMS   int64  `gorm:"not null" json:"created_at_ms"`                // 创建时间（毫秒）
	UpdatedAtMS   int64  `gorm:"not null" json:"updated_at_ms"`                // 更新时间（毫秒）
}

// ErrorLog 错误日志表
type ErrorLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`          // 错误ID，自增主键
	Scope        string `gorm:"index;size:80;not null" json:"scope"`         // 错误范围/模块
	Message      string `gorm:"size:300;not null" json:"message"`            // 错误消息
	Detail       string `gorm:"type:text;not null;default:''" json:"detail"` // 错误详情
	Count        int64  `gorm:"not null;default:1" json:"count"`             // 出现次数
	LastSeenAtMS int64  `gorm:"not null" json:"last_seen_at_ms"`             // 最后出现时间（毫秒）
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`               // 创建时间（毫秒）
	UpdatedAtMS  int64  `gorm:"not null" json:"updated_at_ms"`               // 更新时间（毫秒）
}

// SystemMetric 系统指标快照表
type SystemMetric struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                   // 指标ID，自增主键
	MetricKey    string `gorm:"index;size:80;not null" json:"metric_key"`             // 指标键名
	MetricValue  string `gorm:"size:200;not null" json:"metric_value"`                // 指标值
	SnapshotAtMS int64  `gorm:"not null" json:"snapshot_at_ms"`                       // 快照时间（毫秒）
	MetadataJSON string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"` // 扩展元数据JSON
}

// UsageLog 使用日志表（每次API调用的详细记录）
type UsageLog struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                            // 日志ID，自增主键
	UserID            uint64 `gorm:"index:idx_usage_logs_user_time,priority:1;not null" json:"user_id"`             // 用户ID，复合索引(user_id+created_at_ms)
	APIKeyID          uint64 `gorm:"index;not null" json:"api_key_id"`                                              // API密钥ID
	AccountID         uint64 `gorm:"index;not null" json:"account_id"`                                              // AI账号ID
	Provider          string `gorm:"size:40;not null" json:"provider"`                                              // AI提供商
	Model             string `gorm:"size:120;not null" json:"model"`                                                // 模型名称
	Endpoint          string `gorm:"size:120;not null" json:"endpoint"`                                             // API端点
	InputTokens       int64  `gorm:"not null;default:0" json:"input_tokens"`                                        // 输入token数
	OutputTokens      int64  `gorm:"not null;default:0" json:"output_tokens"`                                       // 输出token数
	CacheCreateTokens int64  `gorm:"not null;default:0" json:"cache_create_tokens"`                                 // 缓存创建token数
	CacheReadTokens   int64  `gorm:"not null;default:0" json:"cache_read_tokens"`                                   // 缓存读取token数
	Cost              int64  `gorm:"not null;default:0" json:"cost"`                                                // 费用
	CreatedAtMS       int64  `gorm:"index;index:idx_usage_logs_user_time,priority:2;not null" json:"created_at_ms"` // 创建时间（毫秒），复合索引
}

// UserUsageMinute 用户分钟级用量汇总表
type UserUsageMinute struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                                  // ID，自增主键
	UserID            uint64 `gorm:"uniqueIndex:idx_user_usage_minute_bucket,priority:1;not null" json:"user_id"`         // 用户ID，唯一复合索引(user_id+bucket_start_ms)
	BucketStartMS     int64  `gorm:"uniqueIndex:idx_user_usage_minute_bucket,priority:2;not null" json:"bucket_start_ms"` // 桶起始时间（毫秒）
	RequestCount      int64  `gorm:"not null;default:0" json:"request_count"`                                             // 请求次数
	InputTokens       int64  `gorm:"not null;default:0" json:"input_tokens"`                                              // 输入token数
	OutputTokens      int64  `gorm:"not null;default:0" json:"output_tokens"`                                             // 输出token数
	CacheCreateTokens int64  `gorm:"not null;default:0" json:"cache_create_tokens"`                                       // 缓存创建token数
	CacheReadTokens   int64  `gorm:"not null;default:0" json:"cache_read_tokens"`                                         // 缓存读取token数
	Cost              int64  `gorm:"not null;default:0" json:"cost"`                                                      // 费用
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                                                       // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                                                       // 更新时间（毫秒）
}

// UserUsageHour 用户小时级用量汇总表
type UserUsageHour struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                                // ID，自增主键
	UserID            uint64 `gorm:"uniqueIndex:idx_user_usage_hour_bucket,priority:1;not null" json:"user_id"`         // 用户ID，唯一复合索引(user_id+bucket_start_ms)
	BucketStartMS     int64  `gorm:"uniqueIndex:idx_user_usage_hour_bucket,priority:2;not null" json:"bucket_start_ms"` // 桶起始时间（毫秒）
	RequestCount      int64  `gorm:"not null;default:0" json:"request_count"`                                           // 请求次数
	InputTokens       int64  `gorm:"not null;default:0" json:"input_tokens"`                                            // 输入token数
	OutputTokens      int64  `gorm:"not null;default:0" json:"output_tokens"`                                           // 输出token数
	CacheCreateTokens int64  `gorm:"not null;default:0" json:"cache_create_tokens"`                                     // 缓存创建token数
	CacheReadTokens   int64  `gorm:"not null;default:0" json:"cache_read_tokens"`                                       // 缓存读取token数
	Cost              int64  `gorm:"not null;default:0" json:"cost"`                                                    // 费用
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                                                     // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                                                     // 更新时间（毫秒）
}

// UserUsageDay 用户天级用量汇总表
type UserUsageDay struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                               // ID，自增主键
	UserID            uint64 `gorm:"uniqueIndex:idx_user_usage_day_bucket,priority:1;not null" json:"user_id"`         // 用户ID，唯一复合索引(user_id+bucket_start_ms)
	BucketStartMS     int64  `gorm:"uniqueIndex:idx_user_usage_day_bucket,priority:2;not null" json:"bucket_start_ms"` // 桶起始时间（毫秒）
	RequestCount      int64  `gorm:"not null;default:0" json:"request_count"`                                          // 请求次数
	InputTokens       int64  `gorm:"not null;default:0" json:"input_tokens"`                                           // 输入token数
	OutputTokens      int64  `gorm:"not null;default:0" json:"output_tokens"`                                          // 输出token数
	CacheCreateTokens int64  `gorm:"not null;default:0" json:"cache_create_tokens"`                                    // 缓存创建token数
	CacheReadTokens   int64  `gorm:"not null;default:0" json:"cache_read_tokens"`                                      // 缓存读取token数
	Cost              int64  `gorm:"not null;default:0" json:"cost"`                                                   // 费用
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                                                    // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                                                    // 更新时间（毫秒）
}

// UserUsageDayDimension 用户天级维度用量汇总表（按provider+model细分）
type UserUsageDayDimension struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                                                   // ID，自增主键
	UserID            uint64 `gorm:"uniqueIndex:idx_user_usage_day_dimension,priority:1;not null" json:"user_id"`          // 用户ID，唯一复合索引(user_id+bucket_start_ms+provider+model)
	BucketStartMS     int64  `gorm:"uniqueIndex:idx_user_usage_day_dimension,priority:2;not null" json:"bucket_start_ms"`  // 桶起始时间（毫秒）
	Provider          string `gorm:"size:40;uniqueIndex:idx_user_usage_day_dimension,priority:3;not null" json:"provider"` // AI提供商
	Model             string `gorm:"size:120;uniqueIndex:idx_user_usage_day_dimension,priority:4;not null" json:"model"`   // 模型名称
	RequestCount      int64  `gorm:"not null;default:0" json:"request_count"`                                              // 请求次数
	InputTokens       int64  `gorm:"not null;default:0" json:"input_tokens"`                                               // 输入token数
	OutputTokens      int64  `gorm:"not null;default:0" json:"output_tokens"`                                              // 输出token数
	CacheCreateTokens int64  `gorm:"not null;default:0" json:"cache_create_tokens"`                                        // 缓存创建token数
	CacheReadTokens   int64  `gorm:"not null;default:0" json:"cache_read_tokens"`                                          // 缓存读取token数
	Cost              int64  `gorm:"not null;default:0" json:"cost"`                                                       // 费用
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`                                                        // 创建时间（毫秒）
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`                                                        // 更新时间（毫秒）
}

// OAuthSession OAuth会话表（用于OAuth认证流程中的状态保持）
type OAuthSession struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement" json:"id"`                   // 会话ID，自增主键
	Provider     string `gorm:"size:40;not null" json:"provider"`                     // AI提供商
	State        string `gorm:"uniqueIndex;size:120;not null" json:"state"`           // OAuth state参数，唯一索引
	CodeVerifier string `gorm:"size:160;not null" json:"code_verifier"`               // PKCE代码验证器
	RedirectURI  string `gorm:"size:300;not null" json:"redirect_uri"`                // 回调URI
	ExpiresAtMS  int64  `gorm:"not null" json:"expires_at_ms"`                        // 过期时间（毫秒）
	MetadataJSON string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"` // 扩展元数据JSON
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`                        // 创建时间（毫秒）
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *User) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *APIKey) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *Account) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *ModelPrice) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *PaymentOrder) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *Coupon) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *Announcement) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *ErrorLog) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充快照时间
func (m *SystemMetric) BeforeCreate(tx *gorm.DB) error {
	if m.SnapshotAtMS == 0 {
		m.SnapshotAtMS = nowMS()
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充创建时间
func (m *UsageLog) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = nowMS()
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *UserUsageMinute) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *UserUsageHour) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充时间戳
func (m *UserUsageDay) BeforeCreate(tx *gorm.DB) error {
	now := nowMS()
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = now
	}
	if m.UpdatedAtMS == 0 {
		m.UpdatedAtMS = now
	}
	return nil
}

// BeforeCreate GORM钩子：创建前自动填充创建时间
func (m *OAuthSession) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAtMS == 0 {
		m.CreatedAtMS = nowMS()
	}
	return nil
}

// nowMS 返回当前时间的毫秒时间戳
func nowMS() int64 {
	return time.Now().UnixMilli()
}
