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
		"money":        fenToYuan(req.Amount), // 转换为元（分->元）
		"notify_url":   req.NotifyURL,
		"return_url":   req.ReturnURL,
		"clientip":     req.ClientIP,
		"device":       req.Device,
		"sign_type":    "HMAC-SHA256",
	}

	// 签名参数（不含sign本身）
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
	}
	// 验证签名
	provided := strings.ToLower(strings.TrimSpace(r.FormValue("sign")))
	expected := sign(params, p.key)
	if provided == "" || provided != expected {
		return nil, fmt.Errorf("invalid gopay notify sign")
	}
	// 解析金额（元转分）
	amount, err := yuanToFen(params["money"])
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
	form.Set("money", fenToYuan(req.Amount))
	form.Set("sign_type", "HMAC-SHA256")
	form.Set("sign", sign(map[string]string{
		"pid":      strconv.FormatUint(p.pid, 10),
		"trade_no": req.ProviderTradeNo,
		"money":    fenToYuan(req.Amount),
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
// 使用HMAC-SHA256算法
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

// fenToYuan 分转元
// 参数：
//   - amount: 金额（分）
//
// 返回：金额（元）的字符串表示
func fenToYuan(amount int64) string {
	return strconv.FormatFloat(float64(amount)/10000, 'f', 2, 64)
}

// yuanToFen 元转分
// 参数：
//   - raw: 金额（元）字符串
//
// 返回：金额（分）和错误
func yuanToFen(raw string) (int64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, err
	}
	// 四舍五入
	return int64(value*10000 + 0.5), nil
}
