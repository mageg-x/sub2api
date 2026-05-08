package provider

import (
	"fmt"
	"net/http"
	"sort"

	"sub2api/server/internal/model"
)

// Provider AI服务Provider接口
// 所有AI服务提供商需要实现此接口
type Provider interface {
	// Name 返回Provider名称
	Name() string
	// BuildUpstreamURL 构建上游API的完整URL
	BuildUpstreamURL(account model.Account, path, rawQuery string) string
	// ApplyRequest 对请求进行必要的处理（如添加认证头）
	ApplyRequest(req *http.Request, account model.Account, token string) error
	// ParseUsage 从响应体解析token使用量
	ParseUsage(body []byte) (int64, int64)
	// SupportsPath 判断该Provider是否支持指定的API路径
	SupportsPath(path string) bool
}

// CacheUsageParser 缓存使用量解析器接口
// 用于解析缓存创建和读取 token
type CacheUsageParser interface {
	ParseCacheUsage(body []byte) (int64, int64, bool)
}

// StreamUsageParser 流式响应使用量解析器接口
// 对于支持流式输出的Provider，需要实现此接口来解析流式响应中的使用量
type StreamUsageParser interface {
	// ParseStreamUsage 解析流式响应中的使用量
	// 返回: 输入token数, 输出token数, 是否成功解析
	ParseStreamUsage(body []byte) (int64, int64, bool)
}

// Registry Provider注册表
// 用于管理所有可用的AI服务Provider
type Registry struct {
	items map[string]Provider // 以Provider名称为键的映射
}

// NewRegistry 创建新的Provider注册表
func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

// Register 注册一个Provider
func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

// Get 根据名称获取Provider
func (r *Registry) Get(name string) (Provider, error) {
	p, ok := r.items[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not registered", name)
	}
	return p, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.items))
	for name := range r.items {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
