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

type HTTP struct {
	cfg  config.Config
	core *service.Core
	web  fs.FS
}

func New(cfg config.Config, core *service.Core) *HTTP {
	dist, _ := webdist.Dist()
	return &HTTP{cfg: cfg, core: core, web: dist}
}

func (h *HTTP) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.home)
	mux.HandleFunc("GET /login", h.home)
	mux.HandleFunc("GET /admin/", h.home)
	mux.HandleFunc("GET /user/", h.home)
	mux.HandleFunc("GET /favicon.ico", h.webAsset)
	mux.HandleFunc("GET /assets/", h.webAsset)
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/refresh", h.refreshToken)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.HandleFunc("POST /api/admin/bootstrap", h.bootstrapAdmin)
	mux.HandleFunc("GET /api/admin/dashboard", h.adminDashboard)
	mux.HandleFunc("GET /api/admin/users", h.adminUsers)
	mux.HandleFunc("POST /api/admin/users", h.adminCreateUser)
	mux.HandleFunc("GET /api/admin/api-keys", h.adminAPIKeys)
	mux.HandleFunc("POST /api/admin/api-keys", h.adminCreateAPIKey)
	mux.HandleFunc("GET /api/admin/accounts", h.adminAccounts)
	mux.HandleFunc("POST /api/admin/accounts", h.adminCreateAccount)
	mux.HandleFunc("PATCH /api/admin/accounts/", h.adminUpdateAccount)
	mux.HandleFunc("POST /api/admin/accounts/", h.adminAccountAction)
	mux.HandleFunc("POST /api/admin/accounts/oauth/start", h.adminOAuthStart)
	mux.HandleFunc("POST /api/admin/accounts/oauth/exchange", h.adminOAuthExchange)
	mux.HandleFunc("POST /api/admin/accounts/oauth/create", h.adminOAuthCreateAccount)
	mux.HandleFunc("GET /api/admin/model-prices", h.adminModelPrices)
	mux.HandleFunc("POST /api/admin/model-prices", h.adminCreateModelPrice)
	mux.HandleFunc("GET /api/admin/announcements", h.adminAnnouncements)
	mux.HandleFunc("POST /api/admin/announcements", h.adminCreateAnnouncement)
	mux.HandleFunc("GET /api/admin/coupons", h.adminCoupons)
	mux.HandleFunc("POST /api/admin/coupons", h.adminCreateCoupon)
	mux.HandleFunc("GET /api/admin/errors", h.adminErrors)
	mux.HandleFunc("GET /api/admin/stats", h.adminStats)
	mux.HandleFunc("GET /api/admin/usage", h.adminUsage)
	mux.HandleFunc("GET /api/admin/payment-orders", h.adminPaymentOrders)
	mux.HandleFunc("POST /api/admin/payment-orders/refund", h.adminRefundPayment)
	mux.HandleFunc("GET /api/user/profile", h.userProfile)
	mux.HandleFunc("PUT /api/user/profile", h.userUpdateProfile)
	mux.HandleFunc("POST /api/user/change-password", h.userChangePassword)
	mux.HandleFunc("POST /api/user/redeem", h.userRedeemCoupon)
	mux.HandleFunc("GET /api/keys", h.userAPIKeys)
	mux.HandleFunc("POST /api/keys", h.userCreateAPIKey)
	mux.HandleFunc("GET /api/usage", h.userUsage)
	mux.HandleFunc("GET /api/payment/orders/my", h.userPaymentOrders)
	mux.HandleFunc("GET /api/payment/orders/", h.userPaymentOrderByID)
	mux.HandleFunc("POST /api/payments/orders", h.createPaymentOrder)
	mux.HandleFunc("POST /api/payments/notify/gopay", h.gopayNotify)
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

func (h *HTTP) postRouter(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/v1internal:"):
		h.proxy(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *HTTP) home(w http.ResponseWriter, r *http.Request) {
	if h.serveWebApp(w, r) {
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, h.homeHTML())
}

func (h *HTTP) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UnixMilli()})
}

func (h *HTTP) webAsset(w http.ResponseWriter, r *http.Request) {
	if h.web == nil {
		http.NotFound(w, r)
		return
	}
	http.FileServerFS(h.web).ServeHTTP(w, r)
}

func (h *HTTP) register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.Register(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

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

func (h *HTTP) logout(w http.ResponseWriter, r *http.Request) {
	var req service.RefreshTokenInput
	_ = decodeJSON(r, &req)
	if err := h.core.LogoutUser(req.RefreshToken); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) me(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *HTTP) bootstrapAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.BootstrapAdmin(req.Name, req.Email, req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminDashboard(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	data, err := h.core.Dashboard()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *HTTP) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateUserInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateUser(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminAPIKeys(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListAPIKeys()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateAPIKeyInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateAPIKey(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminAccounts(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListAccountViews()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateAccountInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateAccount(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminUpdateAccount(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/admin/accounts/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid account id"))
		return
	}
	var req service.UpdateAccountInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.UpdateAccount(id, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) adminAccountAction(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/admin/accounts/"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[1] != "refresh" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("unsupported account action"))
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid account id"))
		return
	}
	item, err := h.core.RefreshAccountOAuth(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminOAuthStart(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.OAuthStartInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.OAuthStart(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminOAuthExchange(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.OAuthExchangeInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.OAuthExchange(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminOAuthCreateAccount(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
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
	item, err := h.core.CreateAccountFromOAuth(req.Provider, req.Name, req.ModelScope, &req.Credentials, req.BaseURL, req.Priority, req.ConcurrencyLimit)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminModelPrices(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListModelPrices()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateModelPrice(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateModelPriceInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateModelPrice(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminAnnouncements(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListAnnouncements()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateAnnouncement(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateAnnouncementInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateAnnouncement(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminCoupons(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListCoupons()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminCreateCoupon(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req service.CreateCouponInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateCoupon(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) adminErrors(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListErrorLogs(parseIntDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	stats, err := h.core.Stats()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *HTTP) adminUsage(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListUsage(parseIntDefault(r.URL.Query().Get("limit"), 200))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminPaymentOrders(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	items, err := h.core.ListPaymentOrders()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) adminRefundPayment(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}
	var req struct {
		OutTradeNo string `json:"out_trade_no"`
		Amount     int64  `json:"amount"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.RefundPayment(r.Context(), req.OutTradeNo, req.Amount); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) userProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *HTTP) userUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req service.UpdateProfileInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.UpdateProfile(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) userChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req service.ChangePasswordInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.core.ChangePassword(user.ID, req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *HTTP) userRedeemCoupon(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.RedeemCoupon(user.ID, req.Code)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) userAPIKeys(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.core.ListUserAPIKeys(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) userCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req struct {
		Name          string   `json:"name"`
		AllowedModels []string `json:"allowed_models"`
		ExpiresAtMS   int64    `json:"expires_at_ms"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := h.core.CreateUserAPIKey(user.ID, req.Name, req.AllowedModels, req.ExpiresAtMS)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) userUsage(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.core.ListUserUsage(user.ID, parseIntDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) userPaymentOrders(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	items, err := h.core.ListUserPaymentOrders(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *HTTP) userPaymentOrderByID(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(r.URL.Path, "/api/payment/orders/"), 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid order id"))
		return
	}
	item, err := h.core.GetUserPaymentOrder(user.ID, id)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *HTTP) createPaymentOrder(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}
	var req service.CreatePaymentOrderInput
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	req.UserID = user.ID
	order, result, err := h.core.CreatePaymentOrder(r.Context(), req, clientIP(r), detectDevice(r.UserAgent()), requestBaseURL(r, h.cfg.PublicBaseURL))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": order, "paying": result})
}

func (h *HTTP) gopayNotify(w http.ResponseWriter, r *http.Request) {
	if err := h.core.HandlePaymentNotify(r); err != nil {
		http.Error(w, "fail", http.StatusOK)
		return
	}
	_, _ = io.WriteString(w, "success")
}

func (h *HTTP) proxy(w http.ResponseWriter, r *http.Request) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing bearer token"))
		return
	}
	auth, err := h.core.AuthenticateAPIKey(strings.TrimSpace(authHeader[len("Bearer "):]))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, respBody, err := h.core.Proxy(r.Context(), auth, r.URL.Path, r.URL.RawQuery, r.Header, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	for k, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(k, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if respBody != nil {
		_, _ = w.Write(respBody)
		return
	}
	_, _ = io.Copy(w, resp.Body)
}

func (h *HTTP) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if h.core.CheckAdminToken(strings.TrimSpace(r.Header.Get("X-Admin-Token"))) {
		return true
	}
	writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid admin token"))
	return false
}

func (h *HTTP) requireUser(w http.ResponseWriter, r *http.Request) (*model.User, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing bearer token"))
		return nil, false
	}
	user, err := h.core.AuthenticateUserToken(strings.TrimSpace(authHeader[len("Bearer "):]))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return nil, false
	}
	return user, true
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	raw, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestBaseURL(r *http.Request, fallback string) string {
	if fallback != "" {
		return fallback
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwarded != "" {
		scheme = forwarded
	}
	host := strings.TrimSpace(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

func (h *HTTP) serveWebApp(w http.ResponseWriter, r *http.Request) bool {
	if h.web != nil {
		if raw, err := fs.ReadFile(h.web, "index.html"); err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(raw)
			return true
		}
	}
	return false
}

func clientIP(r *http.Request) string {
	if raw := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); raw != "" {
		return strings.TrimSpace(strings.Split(raw, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func parseIntDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func detectDevice(ua string) string {
	ua = strings.ToLower(ua)
	for _, marker := range []string{"iphone", "android", "mobile", "micromessenger"} {
		if strings.Contains(ua, marker) {
			return "mobile"
		}
	}
	return "pc"
}

func (h *HTTP) homeHTML() string {
	return `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>sub2api lite console</title>
  <style>
    :root{
      --bg:#f6efe4;
      --paper:#fffaf2;
      --ink:#14110f;
      --muted:#6c655d;
      --accent:#d66b28;
      --accent-2:#1d5c63;
      --line:#dccfb9;
      --ok:#2f7d4b;
      --bad:#9a3412;
      --shadow:0 20px 50px rgba(20,17,15,.12);
    }
    *{box-sizing:border-box}
    body{
      margin:0;
      color:var(--ink);
      background:
        radial-gradient(circle at top right, rgba(214,107,40,.14), transparent 30%),
        radial-gradient(circle at left 20%, rgba(29,92,99,.15), transparent 28%),
        linear-gradient(180deg,#f8f3ea, #f3e8d6 70%, #efe2cd);
      font-family:"Iowan Old Style","Palatino Linotype","Book Antiqua",Palatino,serif;
    }
    .shell{max-width:1400px;margin:0 auto;padding:28px}
    .hero{
      display:grid;grid-template-columns:1.2fr .8fr;gap:24px;margin-bottom:24px;
    }
    .card{
      background:rgba(255,250,242,.86);
      backdrop-filter:blur(6px);
      border:1px solid rgba(220,207,185,.9);
      box-shadow:var(--shadow);
      border-radius:24px;
    }
    .hero-main{padding:28px 28px 24px;position:relative;overflow:hidden}
    .hero-main::after{
      content:"";
      position:absolute;inset:auto -100px -90px auto;width:240px;height:240px;
      background:radial-gradient(circle, rgba(214,107,40,.20), transparent 70%);
      transform:rotate(18deg);
    }
    h1{margin:0 0 8px;font-size:44px;line-height:1;letter-spacing:-.04em}
    .sub{margin:0;color:var(--muted);font-size:15px;max-width:760px}
    .hero-side{padding:24px;display:flex;flex-direction:column;justify-content:space-between}
    .token-box{display:flex;gap:10px;align-items:center;margin-top:14px}
    input,select,textarea,button{
      font:inherit;border-radius:14px;border:1px solid var(--line);background:#fff;padding:12px 14px;color:var(--ink)
    }
    input,select,textarea{width:100%}
    textarea{min-height:110px;resize:vertical}
    button{
      background:var(--ink);color:#fff;border:none;cursor:pointer;transition:.18s transform,.18s opacity
    }
    button.secondary{background:#fff;color:var(--ink);border:1px solid var(--line)}
    button.accent{background:var(--accent)}
    button:hover{transform:translateY(-1px)}
    .grid{
      display:grid;grid-template-columns:repeat(12,minmax(0,1fr));gap:18px
    }
    .panel{padding:20px}
    .span-4{grid-column:span 4}
    .span-6{grid-column:span 6}
    .span-8{grid-column:span 8}
    .span-12{grid-column:span 12}
    .title{font-size:13px;text-transform:uppercase;letter-spacing:.14em;color:var(--muted);margin:0 0 14px}
    .stats{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}
    .stat{padding:14px;border:1px dashed var(--line);border-radius:18px;background:rgba(255,255,255,.65)}
    .stat b{display:block;font-size:28px}
    .row{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:12px}
    .row-3{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:12px}
    .actions{display:flex;gap:10px;flex-wrap:wrap}
    .stack{display:flex;flex-direction:column;gap:12px}
    .muted{color:var(--muted);font-size:13px}
    .log{white-space:pre-wrap;background:#171412;color:#f4efe7;padding:16px;border-radius:18px;min-height:140px;font-family:"SFMono-Regular",Consolas,monospace;font-size:12px}
    table{width:100%;border-collapse:collapse;font-size:13px}
    th,td{padding:10px 8px;border-bottom:1px solid rgba(220,207,185,.8);text-align:left;vertical-align:top}
    th{font-size:11px;text-transform:uppercase;letter-spacing:.12em;color:var(--muted)}
    .badge{display:inline-block;padding:4px 10px;border-radius:999px;background:#efe6d8;color:var(--accent-2);font-size:12px}
    .ok{color:var(--ok)}
    .bad{color:var(--bad)}
    .split{display:grid;grid-template-columns:1fr 1fr;gap:18px}
    @media (max-width:1100px){
      .hero,.split{grid-template-columns:1fr}
      .span-4,.span-6,.span-8{grid-column:span 12}
    }
    @media (max-width:720px){
      .shell{padding:16px}
      h1{font-size:34px}
      .row,.row-3,.stats{grid-template-columns:1fr}
    }
  </style>
</head>
<body>
  <div class="shell">
    <section class="hero">
      <div class="card hero-main">
        <div class="badge">sub2api · lite console</div>
        <h1>可用闭环控制台</h1>
        <p class="sub">这不是展示页，而是当前仓库的最小管理台：管理员可以直接在这里创建用户、API Key、模型价格、发起 OpenAI / Claude / Gemini / Antigravity OAuth，并把拿到的 token 落成账户，随后用 API Key 调代理入口。</p>
      </div>
      <div class="card hero-side">
        <div>
          <p class="title">管理员令牌</p>
          <div class="token-box">
            <input id="adminToken" placeholder="X-Admin-Token">
            <button class="accent" onclick="refreshAll()">载入</button>
          </div>
        </div>
        <p class="muted">代理入口已开放 OpenAI、Claude、Gemini、Antigravity 四类路径。支付一期只接入 gopay。</p>
      </div>
    </section>

    <section class="grid">
      <div class="card panel span-4">
        <p class="title">概览</p>
        <div class="stats" id="stats"></div>
      </div>

      <div class="card panel span-8">
        <p class="title">OAuth 接入</p>
        <div class="split">
          <div class="stack">
            <div class="row">
              <select id="oauthProvider">
                <option value="openai">OpenAI</option>
                <option value="claude">Claude</option>
                <option value="gemini">Gemini</option>
                <option value="antigravity">Antigravity</option>
              </select>
              <select id="oauthType">
                <option value="">默认</option>
                <option value="code_assist">gemini code_assist</option>
                <option value="google_one">gemini google_one</option>
                <option value="ai_studio">gemini ai_studio</option>
              </select>
            </div>
            <div class="row">
              <input id="oauthRedirect" placeholder="redirect_uri 可留空">
              <input id="oauthProject" placeholder="project_id 可选">
            </div>
            <div class="row">
              <input id="oauthTier" placeholder="tier_id 可选">
              <button onclick="startOAuth()">生成授权链接</button>
            </div>
            <div class="stack">
              <input id="oauthSession" placeholder="session_id">
              <input id="oauthState" placeholder="state">
              <textarea id="oauthCode" placeholder="授权后把 code 粘贴到这里"></textarea>
              <button class="secondary" onclick="exchangeOAuth()">交换 token</button>
            </div>
          </div>
          <div class="stack">
            <div class="row">
              <input id="oauthAccountName" placeholder="账户名">
              <input id="oauthAccountBaseURL" placeholder="base_url 可选">
            </div>
            <div class="row">
              <input id="oauthAccountModels" placeholder="模型白名单，逗号分隔">
              <input id="oauthAccountPriority" placeholder="priority，默认100" value="100">
            </div>
            <div class="row">
              <input id="oauthAccountConcurrency" placeholder="并发，默认4" value="4">
              <button class="accent" onclick="createOAuthAccount()">创建 OAuth 账户</button>
            </div>
            <textarea id="oauthCredentialBox" placeholder="交换成功后会在这里生成账户凭据 JSON"></textarea>
          </div>
        </div>
      </div>

      <div class="card panel span-6">
        <p class="title">创建用户 / Key / 价格</p>
        <div class="stack">
          <div class="row">
            <input id="userEmail" placeholder="用户邮箱">
            <input id="userName" placeholder="用户名">
          </div>
          <div class="row">
            <input id="userBalance" placeholder="初始余额，单位 1e-4 元" value="0">
            <input id="userModels" placeholder="允许模型，逗号分隔">
          </div>
          <button onclick="createUser()">创建用户</button>
          <div class="row">
            <input id="keyUserID" placeholder="user_id">
            <input id="keyName" placeholder="API Key 名称">
          </div>
          <div class="row">
            <input id="keyModels" placeholder="允许模型，逗号分隔">
            <button onclick="createKey()">创建 API Key</button>
          </div>
          <div class="row">
            <input id="priceProvider" placeholder="provider，如 openai" value="openai">
            <input id="priceModel" placeholder="model">
          </div>
          <div class="row">
            <input id="priceInput" placeholder="input price / 1k tokens">
            <input id="priceOutput" placeholder="output price / 1k tokens">
          </div>
          <button class="secondary" onclick="createPrice()">创建价格</button>
        </div>
      </div>

      <div class="card panel span-6">
        <p class="title">充值</p>
        <div class="stack">
          <div class="row">
            <input id="payUserID" placeholder="user_id">
            <input id="payAmount" placeholder="充值金额，单位 1e-4 元">
          </div>
          <div class="row">
            <input id="paySubject" placeholder="订单标题" value="Balance Recharge">
            <input id="payReturnURL" placeholder="return_url">
          </div>
          <button class="accent" onclick="createPayment()">创建充值订单</button>
          <div id="paymentResult" class="log"></div>
        </div>
      </div>

      <div class="card panel span-12">
        <p class="title">账户</p>
        <div id="accountsTable"></div>
      </div>
      <div class="card panel span-6">
        <p class="title">用户</p>
        <div id="usersTable"></div>
      </div>
      <div class="card panel span-6">
        <p class="title">API Keys</p>
        <div id="keysTable"></div>
      </div>
      <div class="card panel span-6">
        <p class="title">模型价格</p>
        <div id="pricesTable"></div>
      </div>
      <div class="card panel span-6">
        <p class="title">支付订单</p>
        <div id="ordersTable"></div>
      </div>

      <div class="card panel span-12">
        <p class="title">控制台</p>
        <div id="log" class="log"></div>
      </div>
    </section>
  </div>

  <script>
    let latestOAuthCredentials = null;

    function token() {
      return document.getElementById('adminToken').value.trim();
    }

    function adminHeaders() {
      return {
        'Content-Type': 'application/json',
        'X-Admin-Token': token(),
      };
    }

    function log(msg) {
      const box = document.getElementById('log');
      const line = typeof msg === 'string' ? msg : JSON.stringify(msg, null, 2);
      box.textContent = "[" + new Date().toLocaleTimeString() + "] " + line + "\n\n" + box.textContent;
    }

    async function api(url, options = {}) {
      const res = await fetch(url, options);
      const text = await res.text();
      let data = text;
      try { data = JSON.parse(text); } catch {}
      if (!res.ok) {
        throw new Error(typeof data === 'string' ? data : (data.error || JSON.stringify(data)));
      }
      return data;
    }

    function parseCSV(v) {
      return v.split(',').map(s => s.trim()).filter(Boolean);
    }

    function renderTable(target, rows, cols) {
      const el = document.getElementById(target);
      if (!rows || !rows.length) {
        el.innerHTML = '<div class="muted">暂无数据</div>';
        return;
      }
      const thead = cols.map(c => '<th>' + htmlEscape(c.label) + '</th>').join('');
      const tbody = rows.map(row => {
        const tds = cols.map(c => '<td>' + htmlEscape(c.render ? c.render(row) : row[c.key]) + '</td>').join('');
        return '<tr>' + tds + '</tr>';
      }).join('');
      el.innerHTML = '<table><thead><tr>' + thead + '</tr></thead><tbody>' + tbody + '</tbody></table>';
    }

    function htmlEscape(v) {
      return String(v ?? '')
        .replaceAll('&','&amp;')
        .replaceAll('<','&lt;')
        .replaceAll('>','&gt;')
        .replaceAll('"','&quot;');
    }

    async function refreshAll() {
      try {
        const data = await api('/api/admin/dashboard', { headers: adminHeaders() });
        renderStats(data.stats || {});
        renderTable('usersTable', data.users || [], [
          { label:'ID', key:'id' }, { label:'Email', key:'email' }, { label:'Balance', key:'balance' }, { label:'Models', render:r => r.allowed_models_json }
        ]);
        renderTable('keysTable', data.api_keys || [], [
          { label:'ID', key:'id' }, { label:'User', key:'user_id' }, { label:'Name', key:'name' }, { label:'Secret', key:'secret' }
        ]);
        renderTable('pricesTable', data.prices || [], [
          { label:'Provider', key:'provider' }, { label:'Model', key:'model' }, { label:'Input', key:'input_price' }, { label:'Output', key:'output_price' }
        ]);
        renderTable('ordersTable', data.orders || [], [
          { label:'Trade', key:'out_trade_no' }, { label:'User', key:'user_id' }, { label:'Amount', key:'amount' }, { label:'Status', key:'status' }
        ]);
        renderTable('accountsTable', data.accounts || [], [
          { label:'ID', key:'id' },
          { label:'Provider', key:'provider' },
          { label:'Name', key:'name' },
          { label:'Auth', key:'auth_type' },
          { label:'Status', key:'status' },
          { label:'Models', render:r => r.model_scope_json },
          { label:'Creds', render:r => JSON.stringify(r.credentials || {}) }
        ]);
        log('dashboard refreshed');
      } catch (err) {
        log('refresh failed: ' + err.message);
      }
    }

    function renderStats(stats) {
      const el = document.getElementById('stats');
      const items = Object.entries(stats).map(([k, v]) => '<div class="stat"><span class="muted">' + htmlEscape(k) + '</span><b>' + htmlEscape(v) + '</b></div>');
      el.innerHTML = items.join('');
    }

    async function createUser() {
      try {
        const body = {
          email: document.getElementById('userEmail').value.trim(),
          name: document.getElementById('userName').value.trim(),
          balance: Number(document.getElementById('userBalance').value || 0),
          allowed_models: parseCSV(document.getElementById('userModels').value),
        };
        const data = await api('/api/admin/users', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        log(data);
        refreshAll();
      } catch (err) { log(err.message); }
    }

    async function createKey() {
      try {
        const body = {
          user_id: Number(document.getElementById('keyUserID').value || 0),
          name: document.getElementById('keyName').value.trim(),
          allowed_models: parseCSV(document.getElementById('keyModels').value),
        };
        const data = await api('/api/admin/api-keys', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        log(data);
        refreshAll();
      } catch (err) { log(err.message); }
    }

    async function createPrice() {
      try {
        const body = {
          provider: document.getElementById('priceProvider').value.trim(),
          model: document.getElementById('priceModel').value.trim(),
          input_price: Number(document.getElementById('priceInput').value || 0),
          output_price: Number(document.getElementById('priceOutput').value || 0),
        };
        const data = await api('/api/admin/model-prices', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        log(data);
        refreshAll();
      } catch (err) { log(err.message); }
    }

    async function startOAuth() {
      try {
        const body = {
          provider: document.getElementById('oauthProvider').value,
          oauth_type: document.getElementById('oauthType').value.trim(),
          redirect_uri: document.getElementById('oauthRedirect').value.trim(),
          project_id: document.getElementById('oauthProject').value.trim(),
          tier_id: document.getElementById('oauthTier').value.trim(),
        };
        const data = await api('/api/admin/accounts/oauth/start', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        document.getElementById('oauthSession').value = data.session_id || '';
        document.getElementById('oauthState').value = data.state || '';
        log(data);
        window.open(data.auth_url, '_blank', 'noopener');
      } catch (err) { log(err.message); }
    }

    async function exchangeOAuth() {
      try {
        const body = {
          session_id: document.getElementById('oauthSession').value.trim(),
          state: document.getElementById('oauthState').value.trim(),
          code: document.getElementById('oauthCode').value.trim(),
        };
        const data = await api('/api/admin/accounts/oauth/exchange', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        latestOAuthCredentials = data;
        document.getElementById('oauthCredentialBox').value = JSON.stringify(data, null, 2);
        log(data);
      } catch (err) { log(err.message); }
    }

    async function createOAuthAccount() {
      try {
        const credentials = latestOAuthCredentials?.access_token ? latestOAuthCredentials : JSON.parse(document.getElementById('oauthCredentialBox').value);
        const body = {
          provider: document.getElementById('oauthProvider').value,
          name: document.getElementById('oauthAccountName').value.trim(),
          base_url: document.getElementById('oauthAccountBaseURL').value.trim(),
          model_scope: parseCSV(document.getElementById('oauthAccountModels').value),
          priority: Number(document.getElementById('oauthAccountPriority').value || 100),
          concurrency_limit: Number(document.getElementById('oauthAccountConcurrency').value || 4),
          credentials,
        };
        const data = await api('/api/admin/accounts/oauth/create', { method:'POST', headers: adminHeaders(), body: JSON.stringify(body) });
        log(data);
        refreshAll();
      } catch (err) { log(err.message); }
    }

    async function createPayment() {
      try {
        const body = {
          user_id: Number(document.getElementById('payUserID').value || 0),
          amount: Number(document.getElementById('payAmount').value || 0),
          subject: document.getElementById('paySubject').value.trim(),
          return_url: document.getElementById('payReturnURL').value.trim(),
        };
        const data = await api('/api/payments/orders', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body) });
        document.getElementById('paymentResult').textContent = JSON.stringify(data, null, 2);
        log(data);
        refreshAll();
      } catch (err) { log(err.message); }
    }

    refreshAll();
  </script>
</body>
</html>`
}
