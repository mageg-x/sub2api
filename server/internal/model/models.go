package model

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

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

type SystemMetric struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement:false" json:"id"`
	MetricKey    string `gorm:"index;size:80;not null" json:"metric_key"`
	MetricValue  string `gorm:"size:200;not null" json:"metric_value"`
	SnapshotAtMS int64  `gorm:"not null" json:"snapshot_at_ms"`
	MetadataJSON string `gorm:"type:text;not null;default:'{}'" json:"metadata_json"`
}

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

func initCreate(tx *gorm.DB, table string, id *uint64, createdAt *int64, updatedAt *int64) error {
	if *id == 0 {
		next, err := nextID(tx, table)
		if err != nil {
			return err
		}
		*id = next
	}
	now := nowMS()
	if createdAt != nil && *createdAt == 0 {
		*createdAt = now
	}
	if updatedAt != nil && *updatedAt == 0 {
		*updatedAt = now
	}
	return nil
}

func nextID(tx *gorm.DB, table string) (uint64, error) {
	var max sql.NullInt64
	if err := tx.Table(table).Select("COALESCE(MAX(id), 9999)").Scan(&max).Error; err != nil {
		return 0, err
	}
	return uint64(max.Int64 + 1), nil
}

func nowMS() int64 {
	return time.Now().UnixMilli()
}
