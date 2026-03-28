package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/a523959108/openclaw-proxy/api"
	"github.com/a523959108/openclaw-proxy/internal/config"
	"github.com/a523959108/openclaw-proxy/internal/dns"
	"github.com/a523959108/openclaw-proxy/internal/geoip"
	"github.com/a523959108/openclaw-proxy/internal/lighthouse"
	"github.com/a523959108/openclaw-proxy/internal/selection"
	"github.com/a523959108/openclaw-proxy/internal/stats"
	"github.com/a523959108/openclaw-proxy/internal/subscription"
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

	// 测速服务
	lightHouse := lighthouse.New(cfg, dnsResolver)
	// Set node source for automatic testing
	lightHouse.SetNodeSource(func() []*config.Node {
		return subManager.GetAllNodes()
	})

	// 智能选择器
	selector := selection.New(cfg, lightHouse, subManager, dnsResolver)
	if cfg.EnableAutoSelect {
		selector.StartAutoSelection()
	}

	// 统计收集器
	statsCollector := stats.NewStatsCollector()

	// 启动API服务
	server := api.NewServer(cfg, subManager, selector, lightHouse, dnsResolver, statsCollector)

	// 配置热重载 - 仅在非Windows系统监听SIGHUP信号
	if runtime.GOOS != "windows" {
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
				cfg.Mu.Lock()
				*cfg = *newCfg
				cfg.Mu.Unlock()

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
	}

	// 处理退出信号
	c := make(chan os.Signal, 1)
	exitSignals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		exitSignals = append(exitSignals, syscall.SIGTERM)
	}
	signal.Notify(c, exitSignals...)
	go func() {
		<-c
		log.Println("Shutting down...")
		geoip.GetInstance().Stop()
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
		if cfg.Username == "admin" {
			log.Printf("WARNING: Default username 'admin' is in use, please change it in config.yaml")
		}
		log.Println("Please set/change your admin password in config.yaml")
	}
	if cfg.EnableHTTPS {
		log.Println("HTTPS enabled")
	}
	log.Println("Config hot reload enabled via SIGHUP signal")
	if err := server.Start(); err != nil {
		log.Printf("Server failed: %v", err)
		// Cleanup resources
		if dnsResolver != nil {
			dnsResolver.Stop()
		}
		lightHouse.Stop()
		selector.StopAutoSelection()
		subManager.StopAutoUpdate()
		cancel()
		os.Exit(1)
	}
}
