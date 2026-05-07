package payment

import (
	"context"
	"fmt"
	"net/http"
)

type CreateOrderRequest struct {
	OutTradeNo string
	Subject    string
	Amount     int64
	NotifyURL  string
	ReturnURL  string
	ClientIP   string
	Device     string
}

type CreateOrderResponse struct {
	ProviderTradeNo string      `json:"provider_trade_no"`
	PayType         string      `json:"pay_type"`
	PayInfo         string      `json:"pay_info"`
	Raw             interface{} `json:"raw"`
}

type NotifyResult struct {
	OutTradeNo      string
	ProviderTradeNo string
	Amount          int64
	Paid            bool
}

type RefundRequest struct {
	ProviderTradeNo string
	Amount          int64
}

type Provider interface {
	Name() string
	CreateOrder(ctx context.Context, req CreateOrderRequest) (*CreateOrderResponse, error)
	VerifyNotify(r *http.Request) (*NotifyResult, error)
	Refund(ctx context.Context, req RefundRequest) error
}

type Registry struct {
	items map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, error) {
	item, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("payment provider %s not registered", name)
	}
	return item, nil
}
