// Package gopay 提供Gopay支付渠道实现
// 包含创建订单、验证回调、退款等功能
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

// Provider Gopay支付提供商结构
// 实现payment.Provider接口
type Provider struct {
	baseURL string       // Gopay API基础URL
	pid     uint64       // 商户ID
	key     string       // 商户密钥
	payType int          // 支付类型
	client  *http.Client // HTTP客户端
}

// New 创建Gopay提供商实例
// 参数：
//   - baseURL: Gopay API地址
//   - pid: 商户ID
//   - key: 商户密钥
//   - payType: 支付类型
//
// 返回：Gopay提供商实例
func New(baseURL string, pid uint64, key string, payType int) *Provider {
	return &Provider{
		baseURL: strings.TrimRight(baseURL, "/"),
		pid:     pid,
		key:     key,
		payType: payType,
		client:  &http.Client{},
	}
}

// Name 返回提供商名称
func (p *Provider) Name() string {
	return "gopay"
}

// CreateOrder 创建支付订单
// 参数：
//   - ctx: 上下文
//   - req: 创建订单请求
//
// 返回：创建订单响应和错误
func (p *Provider) CreateOrder(ctx context.Context, req payment.CreateOrderRequest) (*payment.CreateOrderResponse, error) {
	// 检查配置完整性
	if p.baseURL == "" || p.pid == 0 || p.key == "" {
		return nil, fmt.Errorf("gopay config incomplete")
	}

	// 构建请求载荷
	payload := map[string]any{
		"pid":          p.pid,
		"type":         p.payType,
		"out_trade_no": req.OutTradeNo,
		"name":         req.Subject,
		"money":        amountToYuan(req.Amount),
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
		"money":        amountToYuan(req.Amount),
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"clientip":     req.ClientIP,
		"device":       req.Device,
	}
	// 生成签名
	payload["sign"] = sign(signParams, p.key)

	// 序列化JSON
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	// 发送创建订单请求
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

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// 解析响应
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
	// 检查响应状态
	if result.Code != 0 {
		if result.Msg == "" {
			result.Msg = "gopay create order failed"
		}
		return nil, errors.New(result.Msg)
	}
	// 返回订单信息
	return &payment.CreateOrderResponse{
		ProviderTradeNo: result.TradeNo,
		PayType:         result.PayType,
		PayInfo:         result.PayInfo,
		Raw:             result.PayData,
	}, nil
}

// VerifyNotify 验证支付回调
// 参数：
//   - r: HTTP请求
//
// 返回：回调结果和错误
func (p *Provider) VerifyNotify(r *http.Request) (*payment.NotifyResult, error) {
	// 解析表单数据
	if err := r.ParseForm(); err != nil {
		return nil, err
	}
	// 提取参数
	params := map[string]string{
		"trade_no":     strings.TrimSpace(r.FormValue("trade_no")),
		"out_trade_no": strings.TrimSpace(r.FormValue("out_trade_no")),
		"type":         strings.TrimSpace(r.FormValue("type")),
		"status":       strings.TrimSpace(r.FormValue("status")),
		"money":        strings.TrimSpace(r.FormValue("money")),
		"realmoney":    strings.TrimSpace(r.FormValue("realmoney")),
		"sign_type":    strings.TrimSpace(r.FormValue("sign_type")),
	}
	// 验证签名
	provided := strings.ToLower(strings.TrimSpace(r.FormValue("sign")))
	expected := sign(params, p.key)
	if provided == "" || provided != expected {
		return nil, fmt.Errorf("invalid gopay notify sign")
	}
	// 解析金额（元转分）
	amount, err := yuanToAmount(params["money"])
	if err != nil {
		return nil, err
	}
	// 返回回调结果
	return &payment.NotifyResult{
		OutTradeNo:      params["out_trade_no"],
		ProviderTradeNo: params["trade_no"],
		Amount:          amount,
		Paid:            params["status"] == "1",
	}, nil
}

// Refund 退款
// 参数：
//   - ctx: 上下文
//   - req: 退款请求
//
// 返回：错误
func (p *Provider) Refund(ctx context.Context, req payment.RefundRequest) error {
	// 构建表单数据
	form := url.Values{}
	form.Set("pid", strconv.FormatUint(p.pid, 10))
	form.Set("trade_no", req.ProviderTradeNo)
	form.Set("money", amountToYuan(req.Amount))
	form.Set("sign_type", "HMAC-SHA256")
	form.Set("sign", sign(map[string]string{
		"pid":      strconv.FormatUint(p.pid, 10),
		"trade_no": req.ProviderTradeNo,
		"money":    amountToYuan(req.Amount),
	}, p.key))

	// 发送退款请求
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

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// 解析响应
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}
	// 检查退款结果
	if result.Code != 0 {
		if result.Msg == "" {
			result.Msg = "gopay refund failed"
		}
		return errors.New(result.Msg)
	}
	return nil
}

// sign 生成签名
// Gopay 当前接口要求先在待签名串末尾追加 `key=<merchantKey>`，
// 再以同一个 merchantKey 作为 HMAC-SHA256 密钥计算摘要。
// 参数：
//   - params: 待签名参数
//   - key: 密钥
//
// 返回：签名字符串
func sign(params map[string]string, key string) string {
	// 收集需要签名的参数
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" && k != "sign" && k != "sign_type" {
			keys = append(keys, k)
		}
	}
	// 按键名排序
	sort.Strings(keys)
	// 拼接参数字符串
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(params[k])
		b.WriteByte('&')
	}
	b.WriteString("key=")
	b.WriteString(key)
	// 计算HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(b.String()))
	return strings.ToLower(hex.EncodeToString(mac.Sum(nil)))
}

func amountToYuan(amount int64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
		amount = -amount
	}
	whole := amount / 10000
	fractional := amount % 10000
	return fmt.Sprintf("%s%d.%04d", sign, whole, fractional)
}

func yuanToAmount(raw string) (int64, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return 0, fmt.Errorf("empty amount")
	}
	sign := int64(1)
	if strings.HasPrefix(text, "-") {
		sign = -1
		text = strings.TrimPrefix(text, "-")
	}
	parts := strings.Split(text, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount")
	}
	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}
	whole, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, err
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 4 {
		roundDigit := frac[4]
		frac = frac[:4]
		for len(frac) < 4 {
			frac += "0"
		}
		fractional, err := strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
		if roundDigit >= '5' {
			fractional++
			if fractional >= 10000 {
				whole++
				fractional = 0
			}
		}
		return sign * (whole*10000 + fractional), nil
	}
	for len(frac) < 4 {
		frac += "0"
	}
	fractional := int64(0)
	if frac != "" {
		fractional, err = strconv.ParseInt(frac, 10, 64)
		if err != nil {
			return 0, err
		}
	}
	return sign * (whole*10000 + fractional), nil
}
