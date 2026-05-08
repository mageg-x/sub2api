package app

import (
	"context"
	"net/http"
	"time"

	"sub2api/server/internal/config"
	"sub2api/server/internal/handler"
	"sub2api/server/internal/payment"
	gopayimpl "sub2api/server/internal/payment/gopay"
	"sub2api/server/internal/provider"
	antigravity "sub2api/server/internal/provider/antigravity"
	claude "sub2api/server/internal/provider/claude"
	gemini "sub2api/server/internal/provider/gemini"
	openai "sub2api/server/internal/provider/openai"
	"sub2api/server/internal/service"
	"sub2api/server/internal/storage"
)

// New 创建并初始化应用程序
// 参数 cfg: 应用程序配置
// 返回值:
//   - *http.Server: HTTP服务器实例
//   - func(): 清理函数，用于关闭数据库连接等资源
//   - error: 初始化过程中的错误
func New(cfg config.Config) (*http.Server, func(), error) {
	// 打开SQLite数据库连接
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return nil, nil, err
	}

	// 创建AI Provider注册表并注册支持的Provider
	// 支持: OpenAI, Claude, Gemini, Antigravity
	providers := provider.NewRegistry()
	providers.Register(openai.New(cfg))
	providers.Register(claude.New(cfg))
	providers.Register(gemini.New(cfg))
	providers.Register(antigravity.New(cfg))

	// 创建支付Provider注册表
	payments := payment.NewRegistry()
	// 如果配置了Gopay，则注册Gopay支付Provider
	if cfg.GopayURL != "" && cfg.GopayPID > 0 && cfg.GopayKey != "" {
		payments.Register(gopayimpl.New(cfg.GopayURL, cfg.GopayPID, cfg.GopayKey, cfg.GopayType))
	}

	// 创建核心服务实例，包含所有业务逻辑
	core := service.New(cfg, db, providers, payments)
	if err := core.SeedDefaultModelPrices(); err != nil {
		return nil, nil, err
	}

	// 创建可取消的上下文，用于管理后台goroutine的生命周期
	ctx, cancel := context.WithCancel(context.Background())

	// 启动核心服务（后台任务：OAuth刷新循环、指标快照循环）
	core.Start(ctx)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:              cfg.Addr,                        // 监听地址
		Handler:           handler.New(cfg, core).Routes(), // HTTP路由处理器
		ReadHeaderTimeout: 15 * time.Second,                // 读取请求头超时时间
	}

	// 返回服务器实例和清理函数
	// 清理函数会取消上下文，停止后台goroutine
	return server, cancel, nil
}
