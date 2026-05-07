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

func New(cfg config.Config) (*http.Server, func(), error) {
	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		return nil, nil, err
	}

	providers := provider.NewRegistry()
	providers.Register(openai.New())
	providers.Register(claude.New())
	providers.Register(gemini.New())
	providers.Register(antigravity.New())

	payments := payment.NewRegistry()
	if cfg.GopayURL != "" && cfg.GopayPID > 0 && cfg.GopayKey != "" {
		payments.Register(gopayimpl.New(cfg.GopayURL, cfg.GopayPID, cfg.GopayKey, cfg.GopayType))
	}

	core := service.New(cfg, db, providers, payments)
	ctx, cancel := context.WithCancel(context.Background())
	core.Start(ctx)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler.New(cfg, core).Routes(),
		ReadHeaderTimeout: 15 * time.Second,
	}
	return server, cancel, nil
}
