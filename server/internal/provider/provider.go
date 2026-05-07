package provider

import (
	"fmt"
	"net/http"

	"sub2api/server/internal/model"
)

type Provider interface {
	Name() string
	BuildUpstreamURL(account model.Account, path, rawQuery string) string
	ApplyRequest(req *http.Request, account model.Account, token string) error
	ParseUsage(body []byte) (int64, int64)
	SupportsPath(path string) bool
}

type StreamUsageParser interface {
	ParseStreamUsage(body []byte) (int64, int64, bool)
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
	p, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", name)
	}
	return p, nil
}
