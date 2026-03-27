package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/subscription"
	"openclaw-mcp/internal/lighthouse"
	"openclaw-mcp/internal/selection"
	"openclaw-mcp/api"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Printf("Warning: No config file found, using default: %v", err)
		cfg = config.DefaultConfig()
	}

	// 订阅管理器
	subManager := subscription.NewManager(cfg)
	if err := subManager.UpdateAll(); err != nil {
		log.Printf("Failed to update subscriptions: %v", err)
	}

	// 测速服务
	lightHouse := lighthouse.New(cfg)
	go lightHouse.Start()

	// 智能选择器
	selector := selection.New(lightHouse, subManager)
	if cfg.EnableAutoSelect {
		selector.StartAutoSelection()
	}

	// 启动API服务
	server := api.NewServer(cfg, subManager, selector, lightHouse)

	// 处理退出信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down...")
		lightHouse.Stop()
		selector.StopAutoSelection()
		os.Exit(0)
	}()

	log.Printf("Openclaw MCP starting on %s", cfg.ListenAddr)
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
