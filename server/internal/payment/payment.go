package payment

import (
	"context"
	"fmt"
	"net/http"
)

// CreateOrderRequest 创建支付订单的请求参数
type CreateOrderRequest struct {
	OutTradeNo string // 商户订单号（唯一）
	Subject    string // 订单主题/描述
	Amount     int64  // 订单金额（分）
	NotifyURL  string // 异步通知URL
	ReturnURL  string // 支付完成后返回的URL
	ClientIP   string // 客户端IP地址
	Device     string // 设备类型（pc/mobile）
}

// CreateOrderResponse 创建支付订单的响应
type CreateOrderResponse struct {
	ProviderTradeNo string      `json:"provider_trade_no"` // 支付平台订单号
	PayType         string      `json:"pay_type"`          // 支付方式
	PayInfo         string      `json:"pay_info"`          // 支付信息（如支付链接）
	Raw             interface{} `json:"raw"`               // 原始响应数据
}

// NotifyResult 支付回调通知的解析结果
type NotifyResult struct {
	OutTradeNo      string // 商户订单号
	ProviderTradeNo string // 支付平台订单号
	Amount          int64  // 支付金额（分）
	Paid            bool   // 是否已支付
}

// RefundRequest 退款请求参数
type RefundRequest struct {
	ProviderTradeNo string // 支付平台订单号
	Amount          int64  // 退款金额（分）
}

// Provider 支付渠道Provider接口
// 所有支付渠道需要实现此接口
type Provider interface {
	// Name 返回支付渠道名称
	Name() string
	// CreateOrder 创建支付订单
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	// VerifyNotify 验证支付回调通知
	VerifyNotify(r *http.Request) (*NotifyResult, error)
	// Refund 发起退款
	Refund(ctx context.Context, req RefundRequest) error
}

// Registry 支付Provider注册表
// 用于管理所有可用的支付渠道
type Registry struct {
	items map[string]Provider // 以Provider名称为键的映射
}

// NewRegistry 创建新的支付Provider注册表
func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

// Register 注册一个支付Provider
func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

// Get 根据名称获取支付Provider
func (r *Registry) Get(name string) (Provider, error) {
	item, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("payment provider %s not registered", name)
	}
	return item, nil
}
