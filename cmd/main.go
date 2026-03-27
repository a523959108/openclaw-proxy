package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/dns"
	"openclaw-mcp/internal/subscription"
	"openclaw-mcp/internal/lighthouse"
	"openclaw-mcp/internal/selection"
	"openclaw-mcp/internal/stats"
	"openclaw-mcp/api"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Printf("Warning: No config file found, using default: %v", err)
		cfg = config.DefaultConfig()
		// Save default config if not exists
		if err := config.SaveConfig("config.yaml", cfg); err != nil {
			log.Printf("Warning: Failed to save default config: %v", err)
		}
	}

	// 订阅管理器
	subManager := subscription.NewManager(cfg)
	if err := subManager.UpdateAll(); err != nil {
		log.Printf("Failed to update subscriptions: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start auto update subscriptions
	subManager.StartAutoUpdate(ctx)

	// 测速服务
	lightHouse := lighthouse.New(cfg)
	lightHouse.Start()

	// 智能选择器
	// DNS resolver for anti-pollution
	var dnsResolver *dns.Resolver
	if cfg.DNS != nil && cfg.DNS.Enable {
		dnsResolver = dns.NewResolver(&dns.Config{
			TrustedDNS: cfg.DNS.TrustedDNS,
			CacheTTL:   cfg.DNS.CacheTTL,
			Timeout:    cfg.DNS.Timeout,
		})
		log.Printf("DNS anti-pollution enabled with %d trusted servers", len(cfg.DNS.TrustedDNS))
	}

	// 智能选择器
	selector := selection.New(cfg, lightHouse, subManager, dnsResolver)
	if cfg.EnableAutoSelect {
		selector.StartAutoSelection()
	}

	// 统计收集器
	statsCollector := stats.NewStatsCollector()

	// 启动API服务
	server := api.NewServer(cfg, subManager, selector, lightHouse, dnsResolver, statsCollector)

	// 配置热重载 - 监听 SIGHUP 信号重新加载配置
	sigHup := make(chan os.Signal, 1)
	signal.Notify(sigHup, syscall.SIGHUP)
	go func() {
		for {
			<-sigHup
			log.Println("Reloading configuration...")
			newCfg, err := config.LoadConfig("config.yaml")
			if err != nil {
				log.Printf("Failed to reload config: %v", err)
				continue
			}

			// Update configuration
			*cfg = *newCfg

			// Update selection strategy
			selector.UpdateStrategy()

			// Restart auto selection with new interval
			selector.StopAutoSelection()
			if cfg.EnableAutoSelect {
				selector.StartAutoSelection()
			}

			// Restart subscription update with new interval
			subManager.StopAutoUpdate()
			subManager.StartAutoUpdate(ctx)

			log.Println("Configuration reloaded successfully")
		}
	}()

	// 处理退出信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutting down...")
		if dnsResolver != nil {
			dnsResolver.Stop()
		}
		lightHouse.Stop()
		selector.StopAutoSelection()
		subManager.StopAutoUpdate()
		cancel()
		os.Exit(0)
	}()

	log.Printf("Openclaw MCP starting on %s", cfg.ListenAddr)
	if cfg.EnableAuth {
		log.Println("Authentication enabled")
	}
	if cfg.EnableHTTPS {
		log.Println("HTTPS enabled")
	}
	log.Println("Config hot reload enabled via SIGHUP signal")
	if err := server.Start(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
