package gopay

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"sub2api/server/internal/payment"
)

type Provider struct {
	baseURL string
	pid     uint64
	key     string
	payType int
	client  *http.Client
}

func New(baseURL string, pid uint64, key string, payType int) *Provider {
	return &Provider{
		baseURL: strings.TrimRight(baseURL, "/"),
		pid:     pid,
		key:     key,
		payType: payType,
		client:  &http.Client{},
	}
}

func (p *Provider) Name() string {
	return "gopay"
}

func (p *Provider) CreateOrder(ctx context.Context, req payment.CreateOrderRequest) (*payment.CreateOrderResponse, error) {
	if p.baseURL == "" || p.pid == 0 || p.key == "" {
		return nil, fmt.Errorf("gopay config incomplete")
	}

	payload := map[string]any{
		"pid":          p.pid,
		"type":         p.payType,
		"out_trade_no": req.OutTradeNo,
		"name":         req.Subject,
		"money":        fenToYuan(req.Amount),
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"clientip":     req.ClientIP,
		"device":       req.Device,
		"sign_type":    "HMAC-SHA256",
	}

	signParams := map[string]string{
		"pid":          strconv.FormatUint(p.pid, 10),
		"type":         strconv.Itoa(p.payType),
		"out_trade_no": req.OutTradeNo,
		"name":         req.Subject,
		"money":        fenToYuan(req.Amount),
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"clientip":     req.ClientIP,
		"device":       req.Device,
	}
	payload["sign"] = sign(signParams, p.key)

	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/pay/create", strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code      int         `json:"code"`
		Msg       string      `json:"msg"`
		TradeNo   string      `json:"trade_no"`
		PayType   string      `json:"pay_type"`
		PayInfo   string      `json:"pay_info"`
		PayData   interface{} `json:"pay_data"`
		Timestamp int64       `json:"timestamp"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	if result.Code != 0 {
		if result.Msg == "" {
			result.Msg = "gopay create order failed"
		}
		return nil, errors.New(result.Msg)
	}
	return &payment.CreateOrderResponse{
		ProviderTradeNo: result.TradeNo,
		PayType:         result.PayType,
		PayInfo:         result.PayInfo,
		Raw:             result.PayData,
	}, nil
}

func (p *Provider) VerifyNotify(r *http.Request) (*payment.NotifyResult, error) {
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	params := map[string]string{
		"trade_no":     strings.TrimSpace(r.FormValue("trade_no")),
		"out_trade_no": strings.TrimSpace(r.FormValue("out_trade_no")),
		"type":         strings.TrimSpace(r.FormValue("type")),
		"status":       strings.TrimSpace(r.FormValue("status")),
		"money":        strings.TrimSpace(r.FormValue("money")),
		"realmoney":    strings.TrimSpace(r.FormValue("realmoney")),
	}
	provided := strings.ToLower(strings.TrimSpace(r.FormValue("sign")))
	expected := sign(params, p.key)
	if provided == "" || provided != expected {
		return nil, fmt.Errorf("invalid gopay notify sign")
	}
	amount, err := yuanToFen(params["money"])
	if err != nil {
		return nil, err
	}
	return &payment.NotifyResult{
		OutTradeNo:      params["out_trade_no"],
		ProviderTradeNo: params["trade_no"],
		Amount:          amount,
		Paid:            params["status"] == "1",
	}, nil
}

func (p *Provider) Refund(ctx context.Context, req payment.RefundRequest) error {
	form := url.Values{}
	form.Set("pid", strconv.FormatUint(p.pid, 10))
	form.Set("trade_no", req.ProviderTradeNo)
	form.Set("money", fenToYuan(req.Amount))
	form.Set("sign_type", "HMAC-SHA256")
	form.Set("sign", sign(map[string]string{
		"pid":      strconv.FormatUint(p.pid, 10),
		"trade_no": req.ProviderTradeNo,
		"money":    fenToYuan(req.Amount),
	}, p.key))

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/pay/refund", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	if result.Code != 0 {
		if result.Msg == "" {
			result.Msg = "gopay refund failed"
		}
		return errors.New(result.Msg)
	}
	return nil
}

func sign(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" && k != "sign" && k != "sign_type" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
		b.WriteByte('&')
	}
	b.WriteString("key=")
	b.WriteString(key)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(b.String()))
	return strings.ToLower(hex.EncodeToString(mac.Sum(nil)))
}

func fenToYuan(amount int64) string {
	return strconv.FormatFloat(float64(amount)/10000, 'f', 2, 64)
}

func yuanToFen(raw string) (int64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, err
	}
	return int64(value*10000 + 0.5), nil
}
