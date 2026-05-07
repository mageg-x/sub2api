package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sub2api/server/internal/app"
	"sub2api/server/internal/config"
)

// main 是整个应用程序的入口点
// 负责初始化配置、启动HTTP服务器、处理系统信号并优雅关闭
func main() {
	// 从环境变量和命令行参数加载配置
	cfg := config.Load()

	// 创建HTTP服务器和清理函数
	// app.New 会初始化数据库、Provider注册、支付注册、核心服务等
	server, cleanup, err := app.New(cfg)
	if err != nil {
		// 如果初始化失败，记录错误并退出
		log.Fatalf("bootstrap failed: %v", err)
	}
	// 确保程序退出前执行清理操作
	defer cleanup()

	// 启动HTTP服务器在后台goroutine中
	go func() {
		log.Printf("sub2api listening on %s", cfg.Addr)
		// 启动HTTP服务监听
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 如果服务器启动失败，记录错误并退出
			log.Fatalf("server failed: %v", err)
		}
	}()

	// 创建一个通道来接收系统信号（SIGINT和SIGTERM）
	stop := make(chan os.Signal, 1)
	// 监听中断(ctrl+c)和终止信号
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	// 阻塞等待信号
	<-stop

	// 收到信号后，创建带超时的上下文用于优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 关闭HTTP服务器，等待最多5秒让正在处理的请求完成
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown failed: %v", err)
	}
}
