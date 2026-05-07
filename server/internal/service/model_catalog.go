package service

import (
	"fmt"
	"strings"

	"sub2api/server/internal/model"
)

type ModelCatalogChannel struct {
	Key        string           `json:"key"`
	Name       string           `json:"name"`
	Multiplier string           `json:"multiplier"`
	Note       string           `json:"note"`
	Models     []model.ModelPrice `json:"models"`
}

type modelCatalogSeed struct {
	Key        string
	Name       string
	Multiplier string
	Note       string
}

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

func (c *Core) SeedDefaultModelPrices() error {
	existing, err := c.ListModelPrices()
	if err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		seen[modelPriceKey(item.Provider, item.Model)] = struct{}{}
	}
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

func (c *Core) ModelCatalog() ([]ModelCatalogChannel, error) {
	items, err := c.ListModelPrices()
	if err != nil {
		return nil, err
	}
	byProvider := make(map[string][]model.ModelPrice)
	for _, item := range items {
		byProvider[strings.TrimSpace(item.Provider)] = append(byProvider[strings.TrimSpace(item.Provider)], item)
	}
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

func modelPriceKey(provider, model string) string {
	return strings.TrimSpace(strings.ToLower(provider)) + "/" + strings.TrimSpace(strings.ToLower(model))
}

func hasCatalogKey(key string) bool {
	key = strings.TrimSpace(strings.ToLower(key))
	for _, item := range defaultModelCatalog {
		if item.Key == key {
			return true
		}
	}
	return false
}
