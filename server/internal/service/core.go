package service

import (
	"bytes"
	"context"
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
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/cryptoext"
	"sub2api/server/internal/model"
	"sub2api/server/internal/payment"
	"sub2api/server/internal/provider"

	"gorm.io/gorm"
)

const (
	openAIAuthorizeURL      = "https://auth.openai.com/oauth/authorize"
	openAITokenURL          = "https://auth.openai.com/oauth/token"
	openAIDefaultRedirect   = "http://localhost:1455/auth/callback"
	openAIScopes            = "openid profile email offline_access"
	openAIRefreshScopes     = "openid profile email"
	claudeAuthorizeURL      = "https://claude.ai/oauth/authorize"
	claudeTokenURL          = "https://platform.claude.com/v1/oauth/token"
	claudeRedirectURI       = "https://platform.claude.com/oauth/code/callback"
	claudeScopeOAuth        = "org:create_api_key user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
	geminiAuthorizeURL      = "https://accounts.google.com/o/oauth2/v2/auth"
	geminiTokenURL          = "https://oauth2.googleapis.com/token"
	geminiCodeAssistScopes  = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile"
	geminiAIStudioScopes    = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/generative-language.retriever"
	geminiAIRedirectURI     = "http://localhost:1455/auth/callback"
	geminiCLIRedirectURI    = "https://codeassist.google.com/authcode"
	geminiBuiltinClientID   = "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com"
	antigravityAuthorizeURL = "https://accounts.google.com/o/oauth2/v2/auth"
	antigravityTokenURL     = "https://oauth2.googleapis.com/token"
	antigravityUserInfoURL  = "https://www.googleapis.com/oauth2/v2/userinfo"
	antigravityRedirectURI  = "http://localhost:8085/callback"
	antigravityScopes       = "https://www.googleapis.com/auth/cloud-platform https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/cclog https://www.googleapis.com/auth/experimentsandconfigs"
	antigravityClientID     = "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com"
)

type Core struct {
	cfg          config.Config
	db           *gorm.DB
	providers    *provider.Registry
	payments     *payment.Registry
	httpClient   *http.Client
	accountLoads map[uint64]int
	loadMu       sync.Mutex
	refreshMu    sync.Map
	cacheMu      sync.RWMutex
	cacheItems   map[string]cachedProxyResponse
}

type cachedProxyResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
	ExpiresAt  time.Time
}

type usageTrackingReadCloser struct {
	src          io.ReadCloser
	buf          bytes.Buffer
	finalizeOnce sync.Once
	finalize     func([]byte)
}

func (r *usageTrackingReadCloser) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)
	if n > 0 {
		_, _ = r.buf.Write(p[:n])
	}
	if err == io.EOF {
		r.finish()
	}
	return n, err
}

func (r *usageTrackingReadCloser) Close() error {
	err := r.src.Close()
	r.finish()
	return err
}

func (r *usageTrackingReadCloser) finish() {
	r.finalizeOnce.Do(func() {
		if r.finalize != nil {
			r.finalize(append([]byte(nil), r.buf.Bytes()...))
		}
	})
}

type userTokenClaims struct {
	UserID       uint64 `json:"user_id"`
	TokenVersion int64  `json:"token_version"`
	ExpiresAtMS  int64  `json:"expires_at_ms"`
	Kind         string `json:"kind"`
}

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

type ProxyAuth struct {
	User   model.User
	APIKey model.APIKey
}

type UserAuth struct {
	User         model.User `json:"user"`
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	TokenType    string     `json:"token_type"`
	ExpiresIn    int64      `json:"expires_in"`
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

type CreateAPIKeyInput struct {
	UserID        uint64   `json:"user_id"`
	Name          string   `json:"name"`
	AllowedModels []string `json:"allowed_models"`
	ExpiresAtMS   int64    `json:"expires_at_ms"`
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
	Status           string `json:"status"`
	Priority         int    `json:"priority"`
	ConcurrencyLimit int    `json:"concurrency_limit"`
}

type CreateModelPriceInput struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	InputPrice  int64  `json:"input_price"`
	OutputPrice int64  `json:"output_price"`
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

type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token"`
}

type UpdateProfileInput struct {
	Name string `json:"name"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type OAuthStartInput struct {
	Provider    string `json:"provider"`
	RedirectURI string `json:"redirect_uri"`
	OAuthType   string `json:"oauth_type"`
	ProjectID   string `json:"project_id"`
	TierID      string `json:"tier_id"`
}

type OAuthStartResult struct {
	Provider  string `json:"provider"`
	SessionID string `json:"session_id"`
	State     string `json:"state"`
	AuthURL   string `json:"auth_url"`
}

type OAuthExchangeInput struct {
	SessionID string `json:"session_id"`
	State     string `json:"state"`
	Code      string `json:"code"`
}

type OAuthExchangeResult struct {
	AccountCredentials
}

type DashboardData struct {
	Users         []model.User         `json:"users"`
	APIKeys       []model.APIKey       `json:"api_keys"`
	Accounts      []AccountView        `json:"accounts"`
	Prices        []model.ModelPrice   `json:"prices"`
	Orders        []model.PaymentOrder `json:"orders"`
	Announcements []model.Announcement `json:"announcements"`
	Coupons       []model.Coupon       `json:"coupons"`
	Stats         map[string]any       `json:"stats"`
}

type AccountView struct {
	model.Account
	Credentials map[string]any `json:"credentials,omitempty"`
}

func New(cfg config.Config, db *gorm.DB, providers *provider.Registry, payments *payment.Registry) *Core {
	return &Core{
		cfg:          cfg,
		db:           db,
		providers:    providers,
		payments:     payments,
		httpClient:   &http.Client{Timeout: 120 * time.Second},
		accountLoads: map[uint64]int{},
		cacheItems:   map[string]cachedProxyResponse{},
	}
}

func (c *Core) Start(ctx context.Context) {
	go c.refreshOAuthLoop(ctx)
	go c.snapshotMetricsLoop(ctx)
}

func (c *Core) CheckAdminToken(token string) bool {
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(c.cfg.AdminToken)) == 1
}

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

func (c *Core) CreateUser(in CreateUserInput) (*model.User, error) {
	allowed, _ := json.Marshal(normalizeStrings(in.AllowedModels))
	salt, hash, err := hashPassword(in.Password)
	if err != nil {
		return nil, err
	}
	user := &model.User{
		Email:             strings.TrimSpace(in.Email),
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

func (c *Core) ListUsers() ([]model.User, error) {
	var items []model.User
	err := c.db.Order("id asc").Find(&items).Error
	return items, err
}

func (c *Core) GetUserByID(id uint64) (*model.User, error) {
	var user model.User
	if err := c.db.Where("id = ? AND status = ?", id, "active").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Core) Register(in RegisterInput) (*UserAuth, error) {
	return c.createUserAuth(CreateUserInput{
		Email:       in.Email,
		Name:        in.Name,
		Password:    in.Password,
		Role:        "user",
		RatePercent: 100,
	})
}

func (c *Core) createUserAuth(in CreateUserInput) (*UserAuth, error) {
	user, err := c.CreateUser(in)
	if err != nil {
		return nil, err
	}
	return c.issueUserAuth(user)
}

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

func (c *Core) ListUserAPIKeys(userID uint64) ([]model.APIKey, error) {
	var items []model.APIKey
	err := c.db.Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

func (c *Core) CreateUserAPIKey(userID uint64, name string, allowedModels []string, expiresAtMS int64) (*model.APIKey, error) {
	return c.CreateAPIKey(CreateAPIKeyInput{
		UserID:        userID,
		Name:          name,
		AllowedModels: allowedModels,
		ExpiresAtMS:   expiresAtMS,
	})
}

func (c *Core) ListUserUsage(userID uint64, limit int) ([]model.UsageLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []model.UsageLog
	err := c.db.Where("user_id = ?", userID).Order("id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (c *Core) ListUsage(limit int) ([]model.UsageLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	var items []model.UsageLog
	err := c.db.Order("id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (c *Core) ListUserPaymentOrders(userID uint64) ([]model.PaymentOrder, error) {
	var items []model.PaymentOrder
	err := c.db.Where("user_id = ?", userID).Order("id desc").Find(&items).Error
	return items, err
}

func (c *Core) GetUserPaymentOrder(userID, orderID uint64) (*model.PaymentOrder, error) {
	var order model.PaymentOrder
	if err := c.db.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error; err != nil {
		return nil, err
	}
	return &order, nil
}

func (c *Core) RedeemCoupon(userID uint64, code string) (*model.Coupon, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, fmt.Errorf("coupon code is required")
	}
	var coupon model.Coupon
	if err := c.db.Where("code = ? AND status = ?", code, "active").First(&coupon).Error; err != nil {
		return nil, fmt.Errorf("coupon not found")
	}
	if coupon.ExpiresAtMS > 0 && coupon.ExpiresAtMS < time.Now().UnixMilli() {
		return nil, fmt.Errorf("coupon expired")
	}
	if coupon.MaxUses > 0 && coupon.UsedCount >= coupon.MaxUses {
		return nil, fmt.Errorf("coupon exhausted")
	}
	now := time.Now().UnixMilli()
	if err := c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&coupon).Updates(map[string]any{
			"used_count":     gorm.Expr("used_count + 1"),
			"updated_at_ms":  now,
			"usage_log_json": coupon.UsageLogJSON,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Updates(map[string]any{
			"balance":       gorm.Expr("balance + ?", coupon.Amount),
			"updated_at_ms": now,
		}).Error
	}); err != nil {
		return nil, err
	}
	coupon.UsedCount++
	return &coupon, nil
}

func (c *Core) CreateAPIKey(in CreateAPIKeyInput) (*model.APIKey, error) {
	allowed, _ := json.Marshal(normalizeStrings(in.AllowedModels))
	key := &model.APIKey{
		UserID:            in.UserID,
		Name:              strings.TrimSpace(in.Name),
		Secret:            "sk-" + randomHex(24),
		Status:            "active",
		AllowedModelsJSON: string(allowed),
		ExpiresAtMS:       in.ExpiresAtMS,
	}
	return key, c.db.Create(key).Error
}

func (c *Core) ListAPIKeys() ([]model.APIKey, error) {
	var items []model.APIKey
	err := c.db.Order("id asc").Find(&items).Error
	return items, err
}

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
		ConcurrencyLimit:     defaultInt(in.ConcurrencyLimit, 4),
		MetadataJSON:         normalizeJSON(in.Metadata, "{}"),
	}
	return account, c.db.Create(account).Error
}

func (c *Core) UpdateAccount(id uint64, in UpdateAccountInput) error {
	updates := map[string]any{}
	if status := strings.TrimSpace(in.Status); status != "" {
		updates["status"] = status
	}
	if in.Priority > 0 {
		updates["priority"] = in.Priority
	}
	if in.ConcurrencyLimit > 0 {
		updates["concurrency_limit"] = in.ConcurrencyLimit
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at_ms"] = time.Now().UnixMilli()
	return c.db.Model(&model.Account{}).Where("id = ?", id).Updates(updates).Error
}

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
	cred, _ = c.accountCredentials(&account)
	return &AccountView{
		Account:     account,
		Credentials: redactCredentialsForView(&account, cred),
	}, nil
}

func (c *Core) ListAccounts() ([]model.Account, error) {
	var items []model.Account
	err := c.db.Order("priority desc, id asc").Find(&items).Error
	return items, err
}

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

func (c *Core) CreateModelPrice(in CreateModelPriceInput) (*model.ModelPrice, error) {
	item := &model.ModelPrice{
		Provider:    strings.TrimSpace(in.Provider),
		Model:       strings.TrimSpace(in.Model),
		InputPrice:  in.InputPrice,
		OutputPrice: in.OutputPrice,
		Currency:    "CNY_1E4",
		Status:      "active",
	}
	return item, c.db.Create(item).Error
}

func (c *Core) ListModelPrices() ([]model.ModelPrice, error) {
	var items []model.ModelPrice
	err := c.db.Order("provider asc, model asc").Find(&items).Error
	return items, err
}

func (c *Core) CreateAnnouncement(in CreateAnnouncementInput) (*model.Announcement, error) {
	item := &model.Announcement{
		Title:         strings.TrimSpace(in.Title),
		Content:       in.Content,
		Status:        defaultString(in.Status, "published"),
		PublishedAtMS: defaultInt64(in.PublishedAtMS, time.Now().UnixMilli()),
	}
	return item, c.db.Create(item).Error
}

func (c *Core) ListAnnouncements() ([]model.Announcement, error) {
	var items []model.Announcement
	err := c.db.Order("published_at_ms desc, id desc").Find(&items).Error
	return items, err
}

func (c *Core) CreateCoupon(in CreateCouponInput) (*model.Coupon, error) {
	item := &model.Coupon{
		Code:         strings.TrimSpace(in.Code),
		Kind:         strings.TrimSpace(in.Kind),
		Amount:       in.Amount,
		MaxUses:      defaultInt(in.MaxUses, 1),
		ExpiresAtMS:  in.ExpiresAtMS,
		Status:       "active",
		UsageLogJSON: "[]",
	}
	return item, c.db.Create(item).Error
}

func (c *Core) ListCoupons() ([]model.Coupon, error) {
	var items []model.Coupon
	err := c.db.Order("id desc").Find(&items).Error
	return items, err
}

func (c *Core) ListErrorLogs(limit int) ([]model.ErrorLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var items []model.ErrorLog
	err := c.db.Order("last_seen_at_ms desc, id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (c *Core) Dashboard() (*DashboardData, error) {
	stats, err := c.Stats()
	if err != nil {
		return nil, err
	}
	users, err := c.ListUsers()
	if err != nil {
		return nil, err
	}
	keys, err := c.ListAPIKeys()
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
		APIKeys:       keys,
		Accounts:      accounts,
		Prices:        prices,
		Orders:        orders,
		Announcements: ann,
		Coupons:       coupons,
		Stats:         stats,
	}, nil
}

func (c *Core) CreatePaymentOrder(ctx context.Context, in CreatePaymentOrderInput, clientIP, device, baseURL string) (*model.PaymentOrder, *payment.CreateOrderResponse, error) {
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return nil, nil, err
	}
	if in.UserID == 0 {
		return nil, nil, fmt.Errorf("user_id is required")
	}
	if in.Amount <= 0 {
		return nil, nil, fmt.Errorf("amount must be greater than 0")
	}
	if baseURL == "" {
		baseURL = c.cfg.PublicBaseURL
	}
	if baseURL == "" {
		return nil, nil, fmt.Errorf("public base url is required for payment notify")
	}
	order := &model.PaymentOrder{
		UserID:         in.UserID,
		Provider:       "gopay",
		OutTradeNo:     "pay_" + randomHex(12),
		Subject:        defaultString(strings.TrimSpace(in.Subject), "Balance Recharge"),
		Status:         "PENDING",
		Amount:         in.Amount,
		CreditedAmount: in.Amount,
		MetadataJSON:   "{}",
	}
	if err := c.db.Create(order).Error; err != nil {
		return nil, nil, err
	}
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
	now := time.Now().UnixMilli()
	if err := c.db.Model(order).Updates(map[string]any{
		"provider_trade_no": result.ProviderTradeNo,
		"updated_at_ms":     now,
	}).Error; err != nil {
		return nil, nil, err
	}
	order.ProviderTradeNo = result.ProviderTradeNo
	order.UpdatedAtMS = now
	return order, result, nil
}

func (c *Core) HandlePaymentNotify(r *http.Request) error {
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return err
	}
	notify, err := providerImpl.VerifyNotify(r)
	if err != nil {
		c.recordError("payment.notify", "gopay notify verify failed", err.Error())
		return err
	}
	if !notify.Paid {
		return fmt.Errorf("payment not completed")
	}
	return c.db.Transaction(func(tx *gorm.DB) error {
		var order model.PaymentOrder
		if err := tx.Where("out_trade_no = ?", notify.OutTradeNo).First(&order).Error; err != nil {
			return err
		}
		if order.Status == "PAID" {
			return nil
		}
		now := time.Now().UnixMilli()
		if err := tx.Model(&order).Updates(map[string]any{
			"status":            "PAID",
			"provider_trade_no": notify.ProviderTradeNo,
			"notified_at_ms":    now,
			"updated_at_ms":     now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", order.UserID).Updates(map[string]any{
			"balance":       gorm.Expr("balance + ?", order.CreditedAmount),
			"updated_at_ms": now,
		}).Error
	})
}

func (c *Core) RefundPayment(ctx context.Context, outTradeNo string, amount int64) error {
	providerImpl, err := c.payments.Get("gopay")
	if err != nil {
		return err
	}
	var order model.PaymentOrder
	if err := c.db.Where("out_trade_no = ?", outTradeNo).First(&order).Error; err != nil {
		return err
	}
	if order.Status != "PAID" {
		return fmt.Errorf("payment order is not paid")
	}
	var user model.User
	if err := c.db.First(&user, order.UserID).Error; err != nil {
		return err
	}
	if user.Balance < amount {
		return fmt.Errorf("user balance is insufficient for refund")
	}
	if err := providerImpl.Refund(ctx, payment.RefundRequest{
		ProviderTradeNo: order.ProviderTradeNo,
		Amount:          amount,
	}); err != nil {
		c.recordError("payment.refund", "gopay refund failed", err.Error())
		return err
	}
	now := time.Now().UnixMilli()
	return c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&order).Updates(map[string]any{
			"status":        "REFUNDED",
			"updated_at_ms": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&user).Updates(map[string]any{
			"balance":       gorm.Expr("balance - ?", amount),
			"updated_at_ms": now,
		}).Error
	})
}

func (c *Core) AuthenticateAPIKey(secret string) (*ProxyAuth, error) {
	var key model.APIKey
	if err := c.db.Where("secret = ? AND status = ?", strings.TrimSpace(secret), "active").First(&key).Error; err != nil {
		return nil, fmt.Errorf("invalid api key")
	}
	if key.ExpiresAtMS > 0 && key.ExpiresAtMS < time.Now().UnixMilli() {
		return nil, fmt.Errorf("api key expired")
	}
	var user model.User
	if err := c.db.Where("id = ? AND status = ?", key.UserID, "active").First(&user).Error; err != nil {
		return nil, fmt.Errorf("user not available")
	}
	now := time.Now().UnixMilli()
	_ = c.db.Model(&key).Updates(map[string]any{"last_used_at_ms": now, "updated_at_ms": now}).Error
	return &ProxyAuth{User: user, APIKey: key}, nil
}

func (c *Core) Proxy(ctx context.Context, auth *ProxyAuth, path, rawQuery string, hdr http.Header, body []byte) (*http.Response, []byte, error) {
	modelName, stream, providerName, err := detectRoute(path, body)
	if err != nil {
		return nil, nil, err
	}
	if auth.User.Balance <= 0 {
		return nil, nil, fmt.Errorf("insufficient balance")
	}
	if modelName != "" && (!isModelAllowed(auth.User.AllowedModelsJSON, modelName) || !isModelAllowed(auth.APIKey.AllowedModelsJSON, modelName)) {
		return nil, nil, fmt.Errorf("model is not allowed")
	}
	cacheKey, cacheable := c.cacheKey(providerName, path, rawQuery, body, stream)
	if cacheable {
		if cached, ok := c.getCachedResponse(cacheKey); ok {
			respBody, err := io.ReadAll(cached.Body)
			if err != nil {
				return nil, nil, err
			}
			cached.Body = io.NopCloser(bytes.NewReader(respBody))
			return cached, respBody, nil
		}
	}
	account, err := c.pickAccount(providerName, modelName, path)
	if err != nil {
		return nil, nil, err
	}
	defer c.releaseAccount(account.ID)
	token, err := c.accountToken(ctx, &account)
	if err != nil {
		c.recordError("proxy.token", "resolve upstream token failed", err.Error())
		return nil, nil, err
	}
	providerImpl, err := c.providers.Get(account.Provider)
	if err != nil {
		return nil, nil, err
	}
	upstreamURL := providerImpl.BuildUpstreamURL(account, path, rawQuery)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}
	copyHeaders(req.Header, hdr)
	req.Header.Del("Authorization")
	if err := providerImpl.ApplyRequest(req, account, token); err != nil {
		return nil, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordError("proxy.request", "upstream request failed", err.Error())
		return nil, nil, err
	}
	if stream || strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "event-stream") {
		if resp.StatusCode < 400 && modelName != "" {
			resp.Body = &usageTrackingReadCloser{
				src: resp.Body,
				finalize: func(body []byte) {
					inTokens, outTokens := providerImpl.ParseUsage(body)
					if parser, ok := providerImpl.(provider.StreamUsageParser); ok {
						if in, out, found := parser.ParseStreamUsage(body); found {
							inTokens = in
							outTokens = out
						}
					}
					if inTokens > 0 || outTokens > 0 {
						_ = c.recordUsage(auth, &account, modelName, path, inTokens, outTokens)
					}
				},
			}
		}
		return resp, nil, nil
	}
	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return nil, nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(respBody))
	if cacheable && resp.StatusCode < 400 {
		c.putCachedResponse(cacheKey, resp.StatusCode, resp.Header, respBody, 30*time.Second)
	}
	if resp.StatusCode < 400 && modelName != "" {
		inTokens, outTokens := providerImpl.ParseUsage(respBody)
		_ = c.recordUsage(auth, &account, modelName, path, inTokens, outTokens)
	}
	return resp, respBody, nil
}

func (c *Core) OAuthStart(in OAuthStartInput) (*OAuthStartResult, error) {
	providerName := normalizeProvider(in.Provider)
	sessionID := randomHex(16)
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
	_ = sessionID
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

func (c *Core) OAuthExchange(ctx context.Context, in OAuthExchangeInput) (*OAuthExchangeResult, error) {
	var session model.OAuthSession
	if err := c.db.Where("id = ?", in.SessionID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("oauth session not found")
	}
	if session.ExpiresAtMS < time.Now().UnixMilli() {
		return nil, fmt.Errorf("oauth session expired")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(in.State)), []byte(session.State)) != 1 {
		return nil, fmt.Errorf("invalid oauth state")
	}
	meta := map[string]string{}
	_ = json.Unmarshal([]byte(session.MetadataJSON), &meta)
	result, err := c.exchangeOAuthCode(ctx, &session, meta, strings.TrimSpace(in.Code))
	if err != nil {
		return nil, err
	}
	_ = c.db.Delete(&session).Error
	return &OAuthExchangeResult{AccountCredentials: *result}, nil
}

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

func (c *Core) Stats() (map[string]any, error) {
	type target struct {
		name  string
		model any
	}
	targets := []target{
		{"users", &model.User{}},
		{"api_keys", &model.APIKey{}},
		{"accounts", &model.Account{}},
		{"model_prices", &model.ModelPrice{}},
		{"payment_orders", &model.PaymentOrder{}},
		{"announcements", &model.Announcement{}},
		{"coupons", &model.Coupon{}},
		{"usage_logs", &model.UsageLog{}},
	}
	data := map[string]any{}
	for _, item := range targets {
		var count int64
		if err := c.db.Model(item.model).Count(&count).Error; err != nil {
			return nil, err
		}
		data[item.name] = count
	}
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	data["memory_alloc_mb"] = ms.Alloc / 1024 / 1024
	data["timestamp_ms"] = time.Now().UnixMilli()
	return data, nil
}

func (c *Core) ListPaymentOrders() ([]model.PaymentOrder, error) {
	var items []model.PaymentOrder
	err := c.db.Order("id desc").Find(&items).Error
	return items, err
}

func (c *Core) snapshotMetricsLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats, err := c.Stats()
			if err != nil {
				continue
			}
			for k, v := range stats {
				_ = c.db.Create(&model.SystemMetric{
					MetricKey:    k,
					MetricValue:  fmt.Sprintf("%v", v),
					SnapshotAtMS: time.Now().UnixMilli(),
					MetadataJSON: "{}",
				}).Error
			}
		}
	}
}

func (c *Core) refreshOAuthLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var accounts []model.Account
			if err := c.db.Where("auth_type = ? AND status = ?", "oauth", "active").Find(&accounts).Error; err != nil {
				continue
			}
			for i := range accounts {
				if accounts[i].ExpiresAtMS > 0 && accounts[i].ExpiresAtMS-time.Now().UnixMilli() > 10*60*1000 {
					continue
				}
				_, _ = c.accountToken(ctx, &accounts[i])
			}
		}
	}
}

func (c *Core) accountCredentials(account *model.Account) (*AccountCredentials, error) {
	plain, err := cryptoext.Decrypt(c.cfg.AESKey, account.CredentialsEncrypted)
	if err != nil {
		return nil, err
	}
	var cred AccountCredentials
	if err := json.Unmarshal([]byte(plain), &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

func (c *Core) accountToken(ctx context.Context, account *model.Account) (string, error) {
	cred, err := c.accountCredentials(account)
	if err != nil {
		return "", err
	}
	switch account.AuthType {
	case "api_key", "static":
		if cred.APIKey != "" {
			return cred.APIKey, nil
		}
		return cred.AccessToken, nil
	case "oauth":
		if cred.AccessToken != "" && cred.ExpiresAtMS-time.Now().UnixMilli() > 3*60*1000 {
			return cred.AccessToken, nil
		}
		return c.refreshOAuthToken(ctx, account, cred)
	default:
		return "", fmt.Errorf("unsupported auth_type %s", account.AuthType)
	}
}

func (c *Core) refreshOAuthToken(ctx context.Context, account *model.Account, cred *AccountCredentials) (string, error) {
	if cred.RefreshToken == "" {
		return "", fmt.Errorf("oauth refresh token is missing")
	}
	lockAny, _ := c.refreshMu.LoadOrStore(account.ID, &sync.Mutex{})
	lock := lockAny.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()

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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("oauth refresh failed: %s", strings.TrimSpace(string(body)))
	}
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
	cred.AccessToken = result.AccessToken
	if result.RefreshToken != "" {
		cred.RefreshToken = result.RefreshToken
	}
	if result.ExpiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second).UnixMilli()
	}
	if result.IDToken != "" {
		c.populateOpenAIIDToken(cred, result.IDToken)
	}
	return cred.AccessToken, c.persistCredentials(account, cred)
}

func (c *Core) persistCredentials(account *model.Account, cred *AccountCredentials) error {
	encrypted, err := cryptoext.Encrypt(c.cfg.AESKey, mustJSON(cred))
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	if err := c.db.Model(&model.Account{}).Where("id = ?", account.ID).Updates(map[string]any{
		"credentials_encrypted": encrypted,
		"expires_at_ms":         cred.ExpiresAtMS,
		"last_refreshed_at_ms":  now,
		"updated_at_ms":         now,
	}).Error; err != nil {
		return err
	}
	account.CredentialsEncrypted = encrypted
	account.ExpiresAtMS = cred.ExpiresAtMS
	account.LastRefreshedAtMS = now
	return nil
}

func (c *Core) buildAuthorizationURL(providerName, state, codeVerifier, redirectURI string, meta map[string]string) (string, error) {
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

func (c *Core) exchangeOpenAI(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", c.cfg.OpenAI.ClientID)
	form.Set("code", code)
	form.Set("redirect_uri", session.RedirectURI)
	form.Set("code_verifier", session.CodeVerifier)
	resp, err := c.oauthFormRequest(ctx, openAITokenURL, form)
	if err != nil {
		return nil, err
	}
	cred := &AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     c.cfg.OpenAI.ClientID,
		TokenURL:     openAITokenURL,
		RedirectURI:  session.RedirectURI,
	}
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	c.populateOpenAIIDToken(cred, resp["id_token"])
	return cred, nil
}

func (c *Core) exchangeClaude(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
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
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
	return cred, nil
}

func (c *Core) exchangeAntigravity(ctx context.Context, session *model.OAuthSession, code string) (*AccountCredentials, error) {
	form := url.Values{}
	form.Set("client_id", antigravityClientID)
	form.Set("client_secret", c.cfg.Antigravity.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", antigravityRedirectURI)
	form.Set("grant_type", "authorization_code")
	form.Set("code_verifier", session.CodeVerifier)
	resp, err := c.oauthFormRequest(ctx, antigravityTokenURL, form)
	if err != nil {
		return nil, err
	}
	cred := &AccountCredentials{
		AccessToken:  resp["access_token"],
		RefreshToken: resp["refresh_token"],
		ClientID:     antigravityClientID,
		ClientSecret: c.cfg.Antigravity.ClientSecret,
		TokenURL:     antigravityTokenURL,
		RedirectURI:  antigravityRedirectURI,
		UserAgent:    "antigravity/1.21.9 windows/amd64",
	}
	if expiresIn := parseExpires(resp["expires_in"]); expiresIn > 0 {
		cred.ExpiresAtMS = time.Now().Add(time.Duration(expiresIn) * time.Second).UnixMilli()
	}
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

func (c *Core) oauthFormRequest(ctx context.Context, endpoint string, form url.Values) (map[string]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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
		return nil, fmt.Errorf("oauth request failed: %s", strings.TrimSpace(string(body)))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	result := map[string]string{}
	for k, v := range data {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result, nil
}

func (c *Core) populateOpenAIIDToken(cred *AccountCredentials, idToken string) {
	if idToken == "" {
		return
	}
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return
	}
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
	var claims map[string]any
	if json.Unmarshal(raw, &claims) != nil {
		return
	}
	if email, _ := claims["email"].(string); email != "" {
		cred.Email = email
	}
	if authClaims, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if id, _ := authClaims["chatgpt_account_id"].(string); id != "" {
			cred.AccountID = id
		}
		if plan, _ := authClaims["chatgpt_plan_type"].(string); plan != "" {
			cred.PlanType = plan
		}
		if oid, _ := authClaims["poid"].(string); oid != "" {
			cred.OrganizationID = oid
		}
	}
}

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

func (c *Core) pickAccount(providerName, modelName, path string) (model.Account, error) {
	var accounts []model.Account
	if err := c.db.Where("provider = ? AND status = ?", providerName, "active").Order("priority desc, id asc").Find(&accounts).Error; err != nil {
		return model.Account{}, err
	}
	providerImpl, err := c.providers.Get(providerName)
	if err != nil {
		return model.Account{}, err
	}
	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	bestIndex := -1
	bestLoad := int(^uint(0) >> 1)
	for i := range accounts {
		if !providerImpl.SupportsPath(path) {
			continue
		}
		if modelName != "" && !isModelAllowed(accounts[i].ModelScopeJSON, modelName) {
			continue
		}
		load := c.accountLoads[accounts[i].ID]
		limit := accounts[i].ConcurrencyLimit
		if limit <= 0 {
			limit = 1
		}
		if load >= limit {
			continue
		}
		if bestIndex == -1 || load < bestLoad {
			bestIndex = i
			bestLoad = load
		}
	}
	if bestIndex == -1 {
		return model.Account{}, fmt.Errorf("no active %s account available", providerName)
	}
	account := accounts[bestIndex]
	c.accountLoads[account.ID]++
	return account, nil
}

func (c *Core) releaseAccount(accountID uint64) {
	c.loadMu.Lock()
	defer c.loadMu.Unlock()
	if c.accountLoads[accountID] > 0 {
		c.accountLoads[accountID]--
	}
}

func (c *Core) recordUsage(auth *ProxyAuth, account *model.Account, modelName, endpoint string, inTokens, outTokens int64) error {
	cost, err := c.calculateCost(account.Provider, modelName, auth.User.RatePercent, inTokens, outTokens)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	return c.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model.UsageLog{
			UserID:       auth.User.ID,
			APIKeyID:     auth.APIKey.ID,
			AccountID:    account.ID,
			Provider:     account.Provider,
			Model:        modelName,
			Endpoint:     endpoint,
			InputTokens:  inTokens,
			OutputTokens: outTokens,
			Cost:         cost,
		}).Error; err != nil {
			return err
		}
		if cost > 0 {
			return tx.Model(&model.User{}).Where("id = ?", auth.User.ID).Updates(map[string]any{
				"balance":       gorm.Expr("balance - ?", cost),
				"updated_at_ms": now,
			}).Error
		}
		return nil
	})
}

func (c *Core) calculateCost(providerName, modelName string, ratePercent int, inTokens, outTokens int64) (int64, error) {
	var price model.ModelPrice
	if err := c.db.Where("provider = ? AND model = ? AND status = ?", providerName, modelName, "active").First(&price).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, err
	}
	base := (inTokens*price.InputPrice + outTokens*price.OutputPrice + 999) / 1000
	return int64(ratePercent) * base / 100, nil
}

func (c *Core) recordError(scope, message, detail string) {
	now := time.Now().UnixMilli()
	var item model.ErrorLog
	err := c.db.Where("scope = ? AND message = ?", scope, message).First(&item).Error
	if err == nil {
		_ = c.db.Model(&item).Updates(map[string]any{
			"count":           gorm.Expr("count + 1"),
			"detail":          detail,
			"last_seen_at_ms": now,
			"updated_at_ms":   now,
		}).Error
		return
	}
	_ = c.db.Create(&model.ErrorLog{
		Scope:        scope,
		Message:      message,
		Detail:       detail,
		Count:        1,
		LastSeenAtMS: now,
	}).Error
}

func normalizeProvider(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "openai", "claude", "gemini", "antigravity":
		return name
	default:
		return name
	}
}

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

func normalizeJSON(raw json.RawMessage, fallback string) string {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fallback
	}
	return string(raw)
}

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
	if cred.APIKey != "" {
		data["api_key_masked"] = maskSecret(cred.APIKey)
	}
	if cred.RefreshToken != "" {
		data["refresh_token_masked"] = maskSecret(cred.RefreshToken)
	}
	if cred.AccessToken != "" {
		data["access_token_masked"] = maskSecret(cred.AccessToken)
	}
	if cred.SetupToken != "" {
		data["setup_token_masked"] = maskSecret(cred.SetupToken)
	}
	if account.AuthType == "oauth" {
		data["token_url"] = cred.TokenURL
		data["redirect_uri"] = cred.RedirectURI
	}
	return data
}

func detectRoute(path string, body []byte) (modelName string, stream bool, providerName string, err error) {
	switch {
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
	case strings.HasPrefix(path, "/v1/messages"), strings.HasPrefix(path, "/v1/messages/count_tokens"):
		providerName = "claude"
		var payload struct {
			Model string `json:"model"`
		}
		if unmarshalErr := json.Unmarshal(body, &payload); unmarshalErr == nil {
			modelName = payload.Model
		}
		stream = bytes.Contains(body, []byte(`"stream":true`))
		return
	case strings.HasPrefix(path, "/v1beta/models/"), strings.HasPrefix(path, "/v1/models/"):
		providerName = "gemini"
		modelName = parseGeminiModelFromPath(path)
		stream = strings.Contains(path, ":streamGenerateContent")
		return
	case strings.HasPrefix(path, "/v1internal:"):
		providerName = "antigravity"
		return
	default:
		err = fmt.Errorf("unsupported gateway path %s", path)
		return
	}
}

func parseGeminiModelFromPath(path string) string {
	parts := strings.Split(path, "/")
	for i := range parts {
		if parts[i] == "models" && i+1 < len(parts) {
			return strings.Split(parts[i+1], ":")[0]
		}
	}
	return ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func defaultInt(value, fallback int) int {
	if value == 0 {
		return fallback
	}
	return value
}

func defaultInt64(value, fallback int64) int64 {
	if value == 0 {
		return fallback
	}
	return value
}

func isModelAllowed(raw, modelName string) bool {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil || len(items) == 0 || modelName == "" {
		return true
	}
	return slices.Contains(items, "*") || slices.Contains(items, modelName)
}

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

func cloneHeader(src http.Header) http.Header {
	dst := make(http.Header, len(src))
	for k, values := range src {
		copied := make([]string, len(values))
		copy(copied, values)
		dst[k] = copied
	}
	return dst
}

func cloneBody(body []byte) []byte {
	if len(body) == 0 {
		return nil
	}
	out := make([]byte, len(body))
	copy(out, body)
	return out
}

func (c *Core) cacheKey(providerName, path, rawQuery string, body []byte, stream bool) (string, bool) {
	if stream || providerName == "antigravity" {
		return "", false
	}
	if bytes.Contains(body, []byte(`"stream":true`)) || bytes.Contains(body, []byte(`"stream": true`)) {
		return "", false
	}
	sum := sha256.Sum256(append([]byte(providerName+"|"+path+"?"+rawQuery+"|"), body...))
	return hex.EncodeToString(sum[:]), true
}

func (c *Core) getCachedResponse(key string) (*http.Response, bool) {
	if key == "" {
		return nil, false
	}
	c.cacheMu.RLock()
	item, ok := c.cacheItems[key]
	c.cacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(item.ExpiresAt) {
		c.cacheMu.Lock()
		delete(c.cacheItems, key)
		c.cacheMu.Unlock()
		return nil, false
	}
	resp := &http.Response{
		StatusCode: item.StatusCode,
		Header:     cloneHeader(item.Header),
		Body:       io.NopCloser(bytes.NewReader(item.Body)),
	}
	return resp, true
}

func (c *Core) putCachedResponse(key string, statusCode int, header http.Header, body []byte, ttl time.Duration) {
	if key == "" || ttl <= 0 {
		return
	}
	c.cacheMu.Lock()
	c.cacheItems[key] = cachedProxyResponse{
		StatusCode: statusCode,
		Header:     cloneHeader(header),
		Body:       cloneBody(body),
		ExpiresAt:  time.Now().Add(ttl),
	}
	c.cacheMu.Unlock()
}

func parseExpires(raw string) int64 {
	if raw == "" {
		return 0
	}
	var n int64
	fmt.Sscanf(raw, "%d", &n)
	return n
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func randomState() string {
	return base64.RawURLEncoding.EncodeToString(randomBytes(32))
}

func randomCodeVerifier(providerName string) string {
	size := 32
	if providerName == "openai" {
		size = 64
		return hex.EncodeToString(randomBytes(size))
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes(size))
}

func randomBytes(n int) []byte {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return buf
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

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

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 10 {
		return value
	}
	return value[:6] + "..." + value[len(value)-4:]
}

func mustJSON(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func hashPassword(password string) (string, string, error) {
	password = strings.TrimSpace(password)
	if len(password) < 6 {
		return "", "", fmt.Errorf("password must be at least 6 characters")
	}
	salt := randomHex(16)
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return salt, hex.EncodeToString(sum[:]), nil
}

func verifyPassword(salt, expectedHash, password string) bool {
	if salt == "" || expectedHash == "" {
		return false
	}
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return subtle.ConstantTimeCompare([]byte(expectedHash), []byte(hex.EncodeToString(sum[:]))) == 1
}

func (c *Core) issueUserAuth(user *model.User) (*UserAuth, error) {
	accessToken, err := c.signUserToken(user.ID, user.TokenVersion, "access", time.Now().Add(2*time.Hour))
	if err != nil {
		return nil, err
	}
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
	mac := sha256.Sum256([]byte(rawPayload + "." + hex.EncodeToString(c.cfg.AESKey)))
	return rawPayload + "." + base64.RawURLEncoding.EncodeToString(mac[:]), nil
}

func (c *Core) parseUserToken(token, expectedKind string) (*userTokenClaims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid token")
	}
	mac := sha256.Sum256([]byte(parts[0] + "." + hex.EncodeToString(c.cfg.AESKey)))
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(base64.RawURLEncoding.EncodeToString(mac[:]))) != 1 {
		return nil, fmt.Errorf("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
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
