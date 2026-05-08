// Package handler 提供HTTP请求处理
// 处理所有HTTP路由和请求
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/model"
	"sub2api/server/internal/service"
	webdist "sub2api/server/internal/web"
)

// HTTP HTTP处理器结构
// 包含配置、核心服务和静态资源
type HTTP struct {
	cfg  config.Config
	core *service.Core
	web  fs.FS
}

const maxProxyBodyBytes = 8 << 20

// New 创建HTTP处理器
// 参数：
//   - cfg: 应用配置
//   - core: 核心服务
//
// 返回：HTTP处理器
func New(cfg config.Config, core *service.Core) *HTTP {
	dist, _ := webdist.Dist()
	return &HTTP{cfg: cfg, core: core, web: dist}
}

// Routes 注册所有HTTP路由
// 返回：配置好的HTTP多路复用器
func (h *HTTP) Routes() http.Handler {
	mux := http.NewServeMux()
	// 静态资源路由
	mux.HandleFunc("GET /", h.home)
	mux.HandleFunc("GET /login", h.home)
	mux.HandleFunc("GET /admin/", h.home)
	mux.HandleFunc("GET /user/", h.home)
	mux.HandleFunc("GET /favicon.ico", h.webAsset)
	mux.HandleFunc("GET /assets/", h.webAsset)
	// 健康检查
	mux.HandleFunc("GET /healthz", h.healthz)
	// 认证相关
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/refresh", h.refreshToken)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	// 管理后台
	mux.HandleFunc("POST /api/admin/bootstrap", h.bootstrapAdmin)
	mux.HandleFunc("GET /api/admin/dashboard", h.adminDashboard)
	mux.HandleFunc("GET /api/admin/users", h.adminUsers)
	mux.HandleFunc("PATCH /api/admin/users/", h.adminUpdateUser)
	mux.HandleFunc("GET /api/admin/accounts", h.adminAccounts)
	mux.HandleFunc("POST /api/admin/accounts", h.adminCreateAccount)
	mux.HandleFunc("DELETE /api/admin/accounts/", h.adminDeleteAccount)
	mux.HandleFunc("PATCH /api/admin/accounts/", h.adminUpdateAccount)
	mux.HandleFunc("POST /api/admin/accounts/", h.adminAccountAction)
	mux.HandleFunc("POST /api/admin/accounts/oauth/start", h.adminOAuthStart)
	mux.HandleFunc("POST /api/admin/accounts/oauth/exchange", h.adminOAuthExchange)
	mux.HandleFunc("POST /api/admin/accounts/oauth/create", h.adminOAuthCreateAccount)
	mux.HandleFunc("GET /api/admin/model-prices", h.adminModelPrices)
	mux.HandleFunc("POST /api/admin/model-prices", h.adminCreateModelPrice)
	mux.HandleFunc("PATCH /api/admin/model-prices/", h.adminUpdateModelPrice)
	mux.HandleFunc("DELETE /api/admin/model-prices/", h.adminDeleteModelPrice)
	mux.HandleFunc("GET /api/model-catalog", h.userModelCatalog)
	mux.HandleFunc("GET /api/providers", h.userProviders)
	mux.HandleFunc("GET /api/model-prices", h.userModelPrices)
	mux.HandleFunc("GET /api/announcements", h.userAnnouncements)
	mux.HandleFunc("GET /api/admin/announcements", h.adminAnnouncements)
	mux.HandleFunc("POST /api/admin/announcements", h.adminCreateAnnouncement)
	mux.HandleFunc("PATCH /api/admin/announcements/", h.adminUpdateAnnouncement)
	mux.HandleFunc("DELETE /api/admin/announcements/", h.adminDeleteAnnouncement)
	mux.HandleFunc("GET /api/admin/coupons", h.adminCoupons)
	mux.HandleFunc("POST /api/admin/coupons", h.adminCreateCoupon)
	mux.HandleFunc("PATCH /api/admin/coupons/", h.adminUpdateCoupon)
	mux.HandleFunc("DELETE /api/admin/coupons/", h.adminDeleteCoupon)
	mux.HandleFunc("GET /api/admin/errors", h.adminErrors)
	mux.HandleFunc("DELETE /api/admin/errors/", h.adminDeleteError)
	mux.HandleFunc("GET /api/admin/stats", h.adminStats)
	mux.HandleFunc("GET /api/admin/payment-orders", h.adminPaymentOrders)
	mux.HandleFunc("POST /api/admin/payment-orders/refund", h.adminRefundPayment)
	// 用户相关
	mux.HandleFunc("GET /api/user/profile", h.userProfile)
	mux.HandleFunc("PUT /api/user/profile", h.userUpdateProfile)
	mux.HandleFunc("POST /api/user/change-password", h.userChangePassword)
	mux.HandleFunc("POST /api/user/redeem", h.userRedeemCoupon)
	mux.HandleFunc("GET /api/user/dashboard", h.userDashboard)
	mux.HandleFunc("GET /api/keys", h.userAPIKeys)
	mux.HandleFunc("POST /api/keys", h.userCreateAPIKey)
	mux.HandleFunc("DELETE /api/keys/", h.userDeleteAPIKey)
	mux.HandleFunc("GET /api/usage", h.userUsage)
	mux.HandleFunc("GET /api/payment/orders/my", h.userPaymentOrders)
	mux.HandleFunc("GET /api/payment/orders/", h.userPaymentOrderByID)
	// 支付相关
	mux.HandleFunc("POST /api/payments/orders", h.createPaymentOrder)
	mux.HandleFunc("POST /api/payments/notify/gopay", h.gopayNotify)
	// AI代理
	mux.HandleFunc("POST /v1/chat/completions", h.proxy)
	mux.HandleFunc("POST /v1/responses", h.proxy)
	mux.HandleFunc("POST /v1/embeddings", h.proxy)
	mux.HandleFunc("POST /v1/messages", h.proxy)
	mux.HandleFunc("POST /v1/messages/count_tokens", h.proxy)
	mux.HandleFunc("POST /v1beta/models/", h.proxy)
	mux.HandleFunc("POST /v1/models/", h.proxy)
	mux.HandleFunc("POST /", h.postRouter)
	return withCORS(mux)
}

// postRouter POST路由分发
// 处理无法通过路径前缀匹配的请求
func (h *HTTP) postRouter(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/v1internal:"):
		h.proxy(w, r)
	default:
		http.NotFound(w, r)
	}
}

// home 处理首页请求
// 尝试服务Web应用，否则返回HTML
func (h *HTTP) home(w http.ResponseWriter, r *http.Request) {
	if h.serveWebApp(w, r) {
		return
	}
	http.NotFound(w, r)
}

// healthz 健康检查
// 返回服务健康状态
func (h *HTTP) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UnixMilli()})
}

// webAsset 服务静态资源
func (h *HTTP) webAsset(w http.ResponseWriter, r *http.Request) {
	if h.web == nil {
		http.NotFound(w, r)
		return
	}
	http.FileServerFS(h.web).ServeHTTP(w, r)
}

// register 用户注册
func (h *HTTP) register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.ClientIP = clientIP(r)
	item, err := h.core.Register(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// login 用户登录
func (h *HTTP) login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.Login(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// refreshToken 刷新令牌
func (h *HTTP) refreshToken(w http.ResponseWriter, r *http.Request) {
	var req service.RefreshTokenInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.RefreshUserToken(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// logout 用户登出
func (h *HTTP) logout(w http.ResponseWriter, r *http.Request) {
	var req service.RefreshTokenInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.LogoutUser(req.RefreshToken); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// me 获取当前用户信息
func (h *HTTP) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// bootstrapAdmin 引导创建管理员
// 如果系统没有管理员，则创建第一个管理员
func (h *HTTP) bootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.AllowBootstrap {
		writeError(w, http.StatusForbidden, fmt.Errorf("bootstrap is disabled"))
		return
	}
	// 解析请求体
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	// 解析JSON请求
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 调用核心服务创建管理员
	item, err := h.core.BootstrapAdmin(req.Name, req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// adminDashboard 管理后台仪表盘
// 返回系统概览数据
func (h *HTTP) adminDashboard(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取仪表盘数据
	data, err := h.core.Dashboard()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回数据
	writeJSON(w, http.StatusOK, data)
}

// adminUsers 管理后台用户列表
func (h *HTTP) adminUsers(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取用户列表
	items, err := h.core.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回用户列表
	writeJSON(w, http.StatusOK, items)
}

// adminUpdateUser 管理后台更新用户
func (h *HTTP) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/users/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid user id"))
		return
	}
	var req service.UpdateUserInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.UpdateUser(id, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// adminAccounts 管理后台AI账号列表
func (h *HTTP) adminAccounts(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取账号列表（脱敏后）
	items, err := h.core.ListAccountViews()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminCreateAccount 管理后台创建AI账号
func (h *HTTP) adminCreateAccount(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.CreateAccountInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建账号
	item, err := h.core.CreateAccount(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminDeleteAccount(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/accounts/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid account id"))
		return
	}
	if err := h.core.DeleteAccount(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminUpdateAccount(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 从URL路径提取账号ID
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/accounts/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid account id"))
		return
	}
	// 解析请求
	var req service.UpdateAccountInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 更新账号
	if err := h.core.UpdateAccount(id, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回成功
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// adminAccountAction 管理后台账号操作
// 支持刷新OAuth令牌等操作
func (h *HTTP) adminAccountAction(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析URL路径，提取账号ID和操作类型
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/accounts/"), "/")
	parts := strings.Split(trimmed, "/")
	// 验证操作类型
	if len(parts) != 2 || parts[1] != "refresh" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported account action"))
		return
	}
	// 解析账号ID
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid account id"))
		return
	}
	// 执行刷新OAuth操作
	item, err := h.core.RefreshAccountOAuth(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// adminOAuthStart 管理后台OAuth开始
// 发起OAuth授权流程
func (h *HTTP) adminOAuthStart(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.OAuthStartInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 开始OAuth流程
	item, err := h.core.OAuthStart(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回授权URL等信息
	writeJSON(w, http.StatusOK, item)
}

// adminOAuthExchange 管理后台OAuth令牌交换
// 使用授权码换取访问令牌
func (h *HTTP) adminOAuthExchange(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.OAuthExchangeInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 交换令牌
	item, err := h.core.OAuthExchange(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回令牌信息
	writeJSON(w, http.StatusOK, item)
}

// adminOAuthCreateAccount 管理后台通过OAuth创建账号
// 使用OAuth授权创建AI账号
func (h *HTTP) adminOAuthCreateAccount(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req struct {
		Provider         string                     `json:"provider"`
		Name             string                     `json:"name"`
		ModelScope       []string                   `json:"model_scope"`
		BaseURL          string                     `json:"base_url"`
		Priority         int                        `json:"priority"`
		ConcurrencyLimit int                        `json:"concurrency_limit"`
		Credentials      service.AccountCredentials `json:"credentials"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建OAuth账号
	item, err := h.core.CreateAccountFromOAuth(req.Provider, req.Name, req.ModelScope, &req.Credentials, req.BaseURL, req.Priority, req.ConcurrencyLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// adminModelPrices 管理后台模型价格列表
func (h *HTTP) adminModelPrices(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取模型价格列表
	items, err := h.core.ListModelPrices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminCreateModelPrice 管理后台创建模型价格
func (h *HTTP) adminCreateModelPrice(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.CreateModelPriceInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建模型价格
	item, err := h.core.CreateModelPrice(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// userModelPrices 获取公开模型价格列表
func (h *HTTP) userModelPrices(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	_ = user
	items, err := h.core.ListModelPrices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// userModelCatalog 获取公开模型目录
func (h *HTTP) userModelCatalog(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	_ = user
	items, err := h.core.ModelCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) userProviders(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	_ = user
	writeJSON(w, http.StatusOK, h.core.SupportedProviders())
}

// adminAnnouncements 管理后台公告列表
func (h *HTTP) adminAnnouncements(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取公告列表
	items, err := h.core.ListAnnouncements()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminCreateAnnouncement 管理后台创建公告
func (h *HTTP) adminCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.CreateAnnouncementInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建公告
	item, err := h.core.CreateAnnouncement(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// adminCoupons 管理后台优惠券列表
func (h *HTTP) adminCoupons(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取优惠券列表
	items, err := h.core.ListCoupons()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminCreateCoupon 管理后台创建优惠券
func (h *HTTP) adminCreateCoupon(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req service.CreateCouponInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建优惠券
	item, err := h.core.CreateCoupon(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// adminErrors 管理后台错误日志
func (h *HTTP) adminErrors(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取错误日志（支持limit参数）
	items, err := h.core.ListErrorLogs(parseIntDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminStats 管理后台统计数据
func (h *HTTP) adminStats(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取统计数据
	stats, err := h.core.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, stats)
}

// adminPaymentOrders 管理后台支付订单列表
func (h *HTTP) adminPaymentOrders(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 获取支付订单列表
	items, err := h.core.ListPaymentOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// adminRefundPayment 管理后台退款
func (h *HTTP) adminRefundPayment(w http.ResponseWriter, r *http.Request) {
	// 检查管理员权限
	if !h.requireAdmin(w, r) {
		return
	}
	// 解析请求
	var req struct {
		OutTradeNo string `json:"out_trade_no"`
		Amount     int64  `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 执行退款
	if err := h.core.RefundPayment(r.Context(), req.OutTradeNo, req.Amount); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回成功
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// userProfile 用户获取个人信息
func (h *HTTP) userProfile(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 返回用户信息
	writeJSON(w, http.StatusOK, user)
}

// userUpdateProfile 用户更新个人信息
func (h *HTTP) userUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 解析请求
	var req service.UpdateProfileInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 更新用户资料
	item, err := h.core.UpdateProfile(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// userChangePassword 用户修改密码
func (h *HTTP) userChangePassword(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 解析请求
	var req service.ChangePasswordInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 修改密码
	if err := h.core.ChangePassword(user.ID, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回成功
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// userRedeemCoupon 用户兑换优惠券
func (h *HTTP) userRedeemCoupon(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 解析请求
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 兑换优惠券
	item, err := h.core.RedeemCoupon(user.ID, req.Code)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// userDashboard 用户首页聚合数据
func (h *HTTP) userDashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	data, err := h.core.UserDashboard(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// userAPIKeys 用户API密钥列表
func (h *HTTP) userAPIKeys(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 获取用户的API密钥列表
	items, err := h.core.ListUserAPIKeys(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// userCreateAPIKey 用户创建API密钥
func (h *HTTP) userCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 解析请求
	var req struct {
		Provider    string `json:"provider"`
		Name        string `json:"name"`
		ExpiresAtMS int64  `json:"expires_at_ms"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 创建API密钥
	item, err := h.core.CreateUserAPIKey(user.ID, req.Provider, req.Name, req.ExpiresAtMS)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果（包含密钥）
	writeJSON(w, http.StatusOK, item)
}

// userDeleteAPIKey 用户删除API密钥
func (h *HTTP) userDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/keys/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid key id"))
		return
	}
	if err := h.core.DeleteUserAPIKey(user.ID, id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// userUsage 用户使用量查询
func (h *HTTP) userUsage(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 获取用户使用量记录
	items, err := h.core.ListUserUsage(user.ID, parseIntDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// userPaymentOrders 用户支付订单列表
func (h *HTTP) userPaymentOrders(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 获取用户的支付订单列表
	items, err := h.core.ListUserPaymentOrders(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, items)
}

// userPaymentOrderByID 用户获取单个订单详情
func (h *HTTP) userPaymentOrderByID(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 从URL路径提取订单ID
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/payment/orders/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid order id"))
		return
	}
	// 获取订单详情
	item, err := h.core.GetUserPaymentOrder(user.ID, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回结果
	writeJSON(w, http.StatusOK, item)
}

// createPaymentOrder 创建支付订单
func (h *HTTP) createPaymentOrder(w http.ResponseWriter, r *http.Request) {
	// 验证用户身份
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	// 解析请求
	var req service.CreatePaymentOrderInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 设置用户ID
	req.UserID = user.ID
	// 创建支付订单（包含用户IP、设备信息、回调URL等）
	order, result, err := h.core.CreatePaymentOrder(r.Context(), req, clientIP(r), detectDevice(r.UserAgent()), requestBaseURL(r, h.cfg.PublicBaseURL))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 返回订单和支付信息
	writeJSON(w, http.StatusOK, map[string]any{"order": order, "paying": result})
}

// gopayNotify Gopay支付回调
// 处理支付成功后的异步通知
func (h *HTTP) gopayNotify(w http.ResponseWriter, r *http.Request) {
	// 处理支付回调
	if err := h.core.HandlePaymentNotify(r); err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	// 返回成功状态
	_, _ = io.WriteString(w, "success")
}

// proxy AI请求代理
// 核心代理逻辑：将用户请求转发到上游AI服务
func (h *HTTP) proxy(w http.ResponseWriter, r *http.Request) {
	// 获取Authorization头
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	// 验证Bearer令牌
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing bearer token"))
		return
	}
	// 使用API密钥认证
	auth, err := h.core.AuthenticateAPIKey(strings.TrimSpace(authHeader[len("Bearer "):]))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	// 读取请求体，限制最大体积避免恶意大包耗尽内存
	body, err := io.ReadAll(io.LimitReader(r.Body, maxProxyBodyBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(body) > maxProxyBodyBytes {
		writeError(w, http.StatusRequestEntityTooLarge, fmt.Errorf("request body too large"))
		return
	}
	// 执行代理请求
	resp, respBody, err := h.core.Proxy(r.Context(), auth, r.URL.Path, r.URL.RawQuery, r.Header, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// 确保响应体被关闭
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	// 复制响应头
	for k, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(k, value)
		}
	}
	// 返回状态码
	w.WriteHeader(resp.StatusCode)
	// 返回响应体
	if respBody != nil {
		_, _ = w.Write(respBody)
		return
	}
	// 流式响应
	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, 4096)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				_ = resp.Body.Close()
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			break
		}
	}
}

// requireAdmin 检查管理员权限
// 使用X-Admin-Token头进行验证
func (h *HTTP) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	// 检查管理员令牌
	if h.core.CheckAdminToken(strings.TrimSpace(r.Header.Get("X-Admin-Token"))) {
		return true
	}
	// 返回未授权错误
	writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid admin token"))
	return false
}

// requireUser 检查用户身份
// 使用Bearer令牌进行认证
func (h *HTTP) requireUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	// 获取Authorization头
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	// 验证Bearer令牌格式
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing bearer token"))
		return nil, false
	}
	// 使用用户令牌认证
	user, err := h.core.AuthenticateUserToken(strings.TrimSpace(authHeader[len("Bearer "):]))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return nil, false
	}
	// 返回用户信息
	return user, true
}

// decodeJSON 解析JSON请求体
func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

// writeJSON 写入JSON响应
func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

// writeError 写入错误响应
func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

// withCORS CORS中间件
// 添加跨域资源共享头
func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 添加CORS头
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		// 处理预检请求
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		// 继续处理请求
		next.ServeHTTP(w, r)
	})
}

// requestBaseURL 获取请求的基础URL
// 从请求头或TLS信息中推断
func requestBaseURL(r *http.Request, fallback string) string {
	// 使用配置的fallback
	if fallback != "" {
		return fallback
	}
	// 确定协议
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// 优先使用X-Forwarded-Proto头
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	// 确定主机
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	// 组合基础URL
	return scheme + "://" + host
}

// serveWebApp 服务Web应用
// 尝试从嵌入的文件系统提供index.html
func (h *HTTP) serveWebApp(w http.ResponseWriter, _ *http.Request) bool {
	if h.web != nil {
		// 读取index.html
		if raw, err := fs.ReadFile(h.web, "index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(raw)
			return true
		}
	}
	return false
}

// clientIP 获取客户端IP地址
// 仅在请求来自可信内网/本机代理时信任 X-Forwarded-For
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	// 优先使用代理头
	if isTrustedProxyIP(host) {
		if raw := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); raw != "" {
			return strings.TrimSpace(strings.Split(raw, ",")[0])
		}
	}
	// 从RemoteAddr解析
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func isTrustedProxyIP(raw string) bool {
	ip := net.ParseIP(strings.TrimSpace(raw))
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if private := ip.To4(); private != nil {
		switch {
		case private[0] == 10:
			return true
		case private[0] == 172 && private[1] >= 16 && private[1] <= 31:
			return true
		case private[0] == 192 && private[1] == 168:
			return true
		}
	}
	if ip.IsPrivate() {
		return true
	}
	return false
}

// parseIntDefault 解析整数并提供默认值
func parseIntDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// detectDevice 检测设备类型
// 根据User-Agent判断移动设备
func detectDevice(ua string) string {
	ua = strings.ToLower(ua)
	for _, marker := range []string{"iphone", "android", "mobile", "micromessenger"} {
		if strings.Contains(ua, marker) {
			return "mobile"
		}
	}
	return "pc"
}

func (h *HTTP) userAnnouncements(w http.ResponseWriter, r *http.Request) {
	items, err := h.core.ListPublishedAnnouncements()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminDeleteModelPrice(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/model-prices/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid model price id"))
		return
	}
	if err := h.core.DeleteModelPrice(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminUpdateModelPrice(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/model-prices/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid model price id"))
		return
	}
	var req service.UpdateModelPriceInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.UpdateModelPrice(id, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminDeleteAnnouncement(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/announcements/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid announcement id"))
		return
	}
	if err := h.core.DeleteAnnouncement(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminUpdateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/announcements/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid announcement id"))
		return
	}
	var req service.UpdateAnnouncementInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.UpdateAnnouncement(id, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminDeleteCoupon(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/coupons/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid coupon id"))
		return
	}
	if err := h.core.DeleteCoupon(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminUpdateCoupon(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/coupons/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid coupon id"))
		return
	}
	var req service.UpdateCouponInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.UpdateCoupon(id, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminDeleteError(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/errors/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid error log id"))
		return
	}
	if err := h.core.DeleteErrorLog(id); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
