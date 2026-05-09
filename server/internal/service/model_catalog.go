package service

import (
	"fmt"
	"strings"

	"sub2api/server/internal/model"
	"sub2api/server/internal/provider"
)

// ModelCatalogChannel 模型目录渠道
// 按渠道分组展示模型价格信息
type ModelCatalogChannel struct {
	Key        string             `json:"key"`        // 渠道标识
	Name       string             `json:"name"`       // 渠道名称
	Multiplier string             `json:"multiplier"` // 倍率
	Note       string             `json:"note"`       // 备注
	Models     []model.ModelPrice `json:"models"`     // 该渠道下的模型列表
}

// modelCatalogSeed 模型目录种子数据
type modelCatalogSeed struct {
	Key        string // 渠道标识
	Name       string // 渠道名称
	Multiplier string // 倍率
	Note       string // 备注
}

// defaultModelCatalog 默认模型目录渠道配置
var defaultModelCatalog = []modelCatalogSeed{
	{Key: "claude", Name: "Claude 官方渠道", Multiplier: "2x", Note: "官方模型优先展示，适合主力对话与长上下文任务。"},
	{Key: "awsq", Name: "Claude awsq", Multiplier: "0.3x", Note: "低倍率通道，适合成本优先场景。"},
	{Key: "codexplus", Name: "Codex 日抛plus", Multiplier: "0.2x", Note: "短周期通道，适合临时高频调用。"},
	{Key: "codex", Name: "Codex 自营", Multiplier: "0.4x", Note: "自营池，适合稳定性优先场景。"},
	{Key: "deepseek_anthropic", Name: "DeepSeek V4 - Anthropic格式", Multiplier: "1x", Note: "Anthropic 兼容格式入口。"},
	{Key: "deepseek_openai", Name: "DeepSeek V4 - OpenAI格式", Multiplier: "1x", Note: "OpenAI 兼容格式入口。"},
	{Key: "gemini", Name: "Gemini", Multiplier: "0.6x", Note: "Google 系模型通道。"},
	{Key: "gptdraw", Name: "gpt 画图", Multiplier: "1x", Note: "图像生成与图像理解入口。"},
}

// defaultModelPriceSeeds 默认模型价格种子数据
var defaultModelPriceSeeds = []CreateModelPriceInput{
	{Provider: "claude", Model: "claude-haiku-4-5", InputPrice: 3000, OutputPrice: 15000, CacheCreatePrice: 3800, CacheReadPrice: 300, Status: "active"},
	{Provider: "claude", Model: "claude-haiku-4-5-20251001", InputPrice: 3000, OutputPrice: 15000, CacheCreatePrice: 3800, CacheReadPrice: 300, Status: "active"},
	{Provider: "claude", Model: "claude-opus-4-5", InputPrice: 15000, OutputPrice: 75000, CacheCreatePrice: 18800, CacheReadPrice: 1500, Status: "inactive"},
	{Provider: "claude", Model: "claude-opus-4-5-20251101", InputPrice: 15000, OutputPrice: 75000, CacheCreatePrice: 18800, CacheReadPrice: 1500, Status: "inactive"},
	{Provider: "claude", Model: "claude-opus-4-6", InputPrice: 15000, OutputPrice: 75000, CacheCreatePrice: 18800, CacheReadPrice: 1500, Status: "inactive"},
	{Provider: "claude", Model: "claude-sonnet-4-5", InputPrice: 9000, OutputPrice: 45000, CacheCreatePrice: 11300, CacheReadPrice: 900, Status: "active"},
	{Provider: "claude", Model: "claude-sonnet-4-5-20250929", InputPrice: 9000, OutputPrice: 45000, CacheCreatePrice: 11300, CacheReadPrice: 900, Status: "inactive"},
	{Provider: "claude", Model: "claude-sonnet-4-6", InputPrice: 9000, OutputPrice: 45000, CacheCreatePrice: 11300, CacheReadPrice: 900, Status: "active"},
}

// SeedDefaultModelPrices 初始化默认模型价格
// 仅在数据库中不存在对应价格时创建，跳过已存在的
func (c *Core) SeedDefaultModelPrices() error {
	existing, err := c.ListModelPrices()
	if err != nil {
		return err
	}
	// 构建已存在价格的索引
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		seen[modelPriceKey(item.Provider, item.Model)] = struct{}{}
	}
	// 跳过已存在的价格，仅创建新的
	for _, seed := range defaultModelPriceSeeds {
		if _, ok := seen[modelPriceKey(seed.Provider, seed.Model)]; ok {
			continue
		}
		if _, err := c.CreateModelPrice(seed); err != nil {
			return fmt.Errorf("seed model price %s/%s: %w", seed.Provider, seed.Model, err)
		}
	}
	return nil
}

// ModelCatalog 获取模型目录
// 按渠道分组返回模型价格信息，包含默认渠道和自定义渠道
func (c *Core) ModelCatalog() ([]ModelCatalogChannel, error) {
	items, err := c.ListModelPrices()
	if err != nil {
		return nil, err
	}
	// 按 Provider 分组
	byProvider := make(map[string][]model.ModelPrice)
	for _, item := range items {
		byProvider[strings.TrimSpace(item.Provider)] = append(byProvider[strings.TrimSpace(item.Provider)], item)
	}
	// 构建默认渠道目录
	catalog := make([]ModelCatalogChannel, 0, len(defaultModelCatalog))
	for _, seed := range defaultModelCatalog {
		catalog = append(catalog, ModelCatalogChannel{
			Key:        seed.Key,
			Name:       seed.Name,
			Multiplier: seed.Multiplier,
			Note:       seed.Note,
			Models:     byProvider[seed.Key],
		})
	}
	// 追加不在默认目录中的自定义渠道
	for providerKey, models := range byProvider {
		if hasCatalogKey(providerKey) {
			continue
		}
		catalog = append(catalog, ModelCatalogChannel{
			Key:        providerKey,
			Name:       providerKey,
			Multiplier: "1x",
			Note:       "后台自定义价格项",
			Models:     models,
		})
	}
	return catalog, nil
}

// SupportedProviders 获取支持的 Provider 列表
func (c *Core) SupportedProviders() []string {
	return c.providers.Names()
}

// ProviderCapabilities 获取所有 Provider 的能力信息
func (c *Core) ProviderCapabilities() []provider.AccountCapability {
	return c.providers.Capabilities()
}

// modelPriceKey 生成模型价格的唯一键
func modelPriceKey(provider, model string) string {
	return strings.TrimSpace(strings.ToLower(provider)) + "/" + strings.TrimSpace(strings.ToLower(model))
}

// hasCatalogKey 检查渠道是否在默认目录中
func hasCatalogKey(key string) bool {
	key = strings.TrimSpace(strings.ToLower(key))
	for _, item := range defaultModelCatalog {
		if item.Key == key {
			return true
		}
	}
	return false
}
