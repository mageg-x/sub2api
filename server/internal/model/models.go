package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// User 用户模型
// 存储用户账户信息、认证凭证和余额等
type User struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Email             string `gorm:"uniqueIndex;size:200;not null" json:"email"`
	Name              string `gorm:"size:100;not null" json:"name"`
	PasswordSalt      string `gorm:"size:64;not null;default:''" json:"-"`
	PasswordHash      string `gorm:"size:128;not null;default:''" json:"-"`
	Role              string `gorm:"size:20;not null;default:user" json:"role"`
	Status            string `gorm:"size:20;not null;default:active" json:"status"`
	TokenVersion      int64  `gorm:"not null;default:1" json:"-"`
	Balance           int64  `gorm:"not null;default:0" json:"balance"`
	RatePercent       int    `gorm:"not null;default:100" json:"rate_percent"`
	AllowedModelsJSON string `gorm:"type:text;not null;default:'[]'" json:"allowed_models_json"`
	MetadataJSON      string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
	LastLoginAtMS     int64  `gorm:"not null;default:0" json:"last_login_at_ms"`
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`
}

// APIKey API密钥模型
// 用户可以通过API密钥访问AI服务
type APIKey struct {
	ID                uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	UserID            uint64 `gorm:"index;not null" json:"user_id"`
	Name              string `gorm:"size:100;not null" json:"name"`
	Secret            string `gorm:"uniqueIndex;size:120;not null" json:"secret"`
	Status            string `gorm:"size:20;not null;default:active" json:"status"`
	AllowedModelsJSON string `gorm:"type:text;not null;default:'[]'" json:"allowed_models_json"`
	ExpiresAtMS       int64  `gorm:"not null;default:0" json:"expires_at_ms"`
	LastUsedAtMS      int64  `gorm:"not null;default:0" json:"last_used_at_ms"`
	CreatedAtMS       int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS       int64  `gorm:"not null" json:"updated_at_ms"`
}

// Account AI账号模型
// 存储AI服务提供商的认证信息，如API Key、OAuth令牌等
type Account struct {
	ID                   uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Provider             string `gorm:"index;size:40;not null" json:"provider"`
	Name                 string `gorm:"size:120;not null" json:"name"`
	AuthType             string `gorm:"size:40;not null" json:"auth_type"`
	BaseURL              string `gorm:"size:300;not null" json:"base_url"`
	ModelScopeJSON       string `gorm:"type:text;not null;default:'[]'" json:"model_scope_json"`
	CredentialsEncrypted string `gorm:"type:text;not null" json:"-"`
	Status               string `gorm:"size:20;not null;default:active" json:"status"`
	Priority             int    `gorm:"not null;default:100" json:"priority"`
	ConcurrencyLimit     int    `gorm:"not null;default:4" json:"concurrency_limit"`
	ExpiresAtMS          int64  `gorm:"not null;default:0" json:"expires_at_ms"`
	LastRefreshedAtMS    int64  `gorm:"not null;default:0" json:"last_refreshed_at_ms"`
	MetadataJSON         string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
	CreatedAtMS          int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS          int64  `gorm:"not null" json:"updated_at_ms"`
}

// ModelPrice 模型价格模型
// 存储不同AI模型的定价信息，用于计算使用成本
type ModelPrice struct {
	ID          uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Provider    string `gorm:"index;size:40;not null" json:"provider"`
	Model       string `gorm:"index;size:120;not null" json:"model"`
	InputPrice  int64  `gorm:"not null" json:"input_price"`
	OutputPrice int64  `gorm:"not null" json:"output_price"`
	Currency    string `gorm:"size:20;not null;default:CNY_1E4" json:"currency"`
	Status      string `gorm:"size:20;not null;default:active" json:"status"`
	CreatedAtMS int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS int64  `gorm:"not null" json:"updated_at_ms"`
}

// PaymentOrder 支付订单模型
// 存储用户的充值订单信息
type PaymentOrder struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	UserID          uint64 `gorm:"index;not null" json:"user_id"`
	Provider        string `gorm:"size:40;not null" json:"provider"`
	OutTradeNo      string `gorm:"uniqueIndex;size:80;not null" json:"out_trade_no"`
	ProviderTradeNo string `gorm:"size:80;not null;default:''" json:"provider_trade_no"`
	Subject         string `gorm:"size:120;not null" json:"subject"`
	Status          string `gorm:"size:20;not null;default:PENDING" json:"status"`
	Amount          int64  `gorm:"not null" json:"amount"`
	CreditedAmount  int64  `gorm:"not null" json:"credited_amount"`
	MetadataJSON    string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
	NotifiedAtMS    int64  `gorm:"not null;default:0" json:"notified_at_ms"`
	CreatedAtMS     int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS     int64  `gorm:"not null" json:"updated_at_ms"`
}

// Coupon 优惠券模型
// 存储可兑换的优惠券信息
type Coupon struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Code         string `gorm:"uniqueIndex;size:80;not null" json:"code"`
	Kind         string `gorm:"size:20;not null" json:"kind"`
	Amount       int64  `gorm:"not null" json:"amount"`
	MaxUses      int    `gorm:"not null;default:1" json:"max_uses"`
	UsedCount    int    `gorm:"not null;default:0" json:"used_count"`
	UsageLogJSON string `gorm:"type:text;not null;default:'[]'" json:"usage_log_json"`
	ExpiresAtMS  int64  `gorm:"not null;default:0" json:"expires_at_ms"`
	Status       string `gorm:"size:20;not null;default:active" json:"status"`
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS  int64  `gorm:"not null" json:"updated_at_ms"`
}

// Announcement 公告模型
// 存储系统公告信息
type Announcement struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Title         string `gorm:"size:200;not null" json:"title"`
	Content       string `gorm:"type:text;not null" json:"content"`
	ReadCount     int64  `gorm:"not null;default:0" json:"read_count"`
	PublishedAtMS int64  `gorm:"not null;default:0" json:"published_at_ms"`
	Status        string `gorm:"size:20;not null;default:draft" json:"status"`
	CreatedAtMS   int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS   int64  `gorm:"not null" json:"updated_at_ms"`
}

// ErrorLog 错误日志模型
// 存储系统错误信息，用于问题排查
type ErrorLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Scope        string `gorm:"index;size:80;not null" json:"scope"`
	Message      string `gorm:"size:300;not null" json:"message"`
	Detail       string `gorm:"type:text;not null;default:''" json:"detail"`
	Count        int64  `gorm:"not null;default:1" json:"count"`
	LastSeenAtMS int64  `gorm:"not null" json:"last_seen_at_ms"`
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`
	UpdatedAtMS  int64  `gorm:"not null" json:"updated_at_ms"`
}

// SystemMetric 系统指标模型
// 存储系统运行时的各种指标数据
type SystemMetric struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MetricKey    string `gorm:"index;size:80;not null" json:"metric_key"`
	MetricValue  string `gorm:"size:200;not null" json:"metric_value"`
	SnapshotAtMS int64  `gorm:"not null" json:"snapshot_at_ms"`
	MetadataJSON string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
}

// UsageLog 使用日志模型
// 记录用户使用AI服务的详细情况
type UsageLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	UserID       uint64 `gorm:"index;not null" json:"user_id"`
	APIKeyID     uint64 `gorm:"index;not null" json:"api_key_id"`
	AccountID    uint64 `gorm:"index;not null" json:"account_id"`
	Provider     string `gorm:"size:40;not null" json:"provider"`
	Model        string `gorm:"size:120;not null" json:"model"`
	Endpoint     string `gorm:"size:120;not null" json:"endpoint"`
	InputTokens  int64  `gorm:"not null;default:0" json:"input_tokens"`
	OutputTokens int64  `gorm:"not null;default:0" json:"output_tokens"`
	Cost         int64  `gorm:"not null;default:0" json:"cost"`
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`
}

// OAuthSession OAuth会话模型
// 存储OAuth授权过程中的会话信息
type OAuthSession struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	Provider     string `gorm:"size:40;not null" json:"provider"`
	State        string `gorm:"uniqueIndex;size:120;not null" json:"state"`
	CodeVerifier string `gorm:"size:160;not null" json:"code_verifier"`
	RedirectURI  string `gorm:"size:300;not null" json:"redirect_uri"`
	ExpiresAtMS  int64  `gorm:"not null" json:"expires_at_ms"`
	MetadataJSON string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
	CreatedAtMS  int64  `gorm:"not null" json:"created_at_ms"`
}

// BeforeCreate 创建记录前的钩子函数
// 自动初始化ID、创建时间和更新时间
func (m *User) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "users", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *APIKey) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "api_keys", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *Account) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "accounts", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *ModelPrice) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "model_prices", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *PaymentOrder) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "payment_orders", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *Coupon) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "coupons", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *Announcement) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "announcements", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *ErrorLog) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "error_logs", &m.ID, &m.CreatedAtMS, &m.UpdatedAtMS)
}
func (m *SystemMetric) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "system_metrics", &m.ID, &m.SnapshotAtMS, nil)
}
func (m *UsageLog) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "usage_logs", &m.ID, &m.CreatedAtMS, nil)
}
func (m *OAuthSession) BeforeCreate(tx *gorm.DB) error {
	return initCreate(tx, "oauth_sessions", &m.ID, &m.CreatedAtMS, nil)
}

// initCreate 初始化创建的记录
// 自动生成ID（如果为0）和时间戳
func initCreate(tx *gorm.DB, table string, id *uint64, createdAt *int64, updatedAt *int64) error {
	if *id == 0 {
		// 生成新的ID
		next, err := nextID(tx, table)
		if err != nil {
			return err
		}
		*id = next
	}
	// 设置创建时间
	now := nowMS()
	if createdAt != nil && *createdAt == 0 {
		*createdAt = now
	}
	// 设置更新时间
	if updatedAt != nil && *updatedAt == 0 {
		*updatedAt = now
	}
	return nil
}

// nextID 生成下一个自增ID
// 从数据库表中获取最大ID并加1
func nextID(tx *gorm.DB, table string) (uint64, error) {
	var max sql.NullInt64
	if err := tx.Table(table).Select("COALESCE(MAX(id), 9999)").Scan(&max).Error; err != nil {
		return 0, err
	}
	return uint64(max.Int64 + 1), nil
}

// nowMS 获取当前时间戳（毫秒）
func nowMS() int64 {
	return time.Now().UnixMilli()
}
