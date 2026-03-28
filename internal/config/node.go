package config

import (
	"sync"
	"time"
)

// Node 代表一个代理节点
type Node struct {
	Name           string            `json:"name" yaml:"name"`
	Type           string            `json:"type" yaml:"type"` // ss, vmess, vless, trojan, hysteria, hysteria2, tuic
	Server         string            `json:"server" yaml:"server"`
	Port           int               `json:"port" yaml:"port"`
	Password       string            `json:"password,omitempty" yaml:"password,omitempty"`
	UUID           string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	AlterID        int               `json:"alterId,omitempty" yaml:"alterId,omitempty"`
	Security       string            `json:"security,omitempty" yaml:"security,omitempty"`
	Network        string            `json:"network,omitempty" yaml:"network,omitempty"`
	Path           string            `json:"path,omitempty" yaml:"path,omitempty"`
	Host           string            `json:"host,omitempty" yaml:"host,omitempty"`
	TLS            bool              `json:"tls,omitempty" yaml:"tls,omitempty"`
	ServerName     string            `json:"servername,omitempty" yaml:"servername,omitempty"`
	ALPN           []string          `json:"alpn,omitempty" yaml:"alpn,omitempty"`
	FP             string            `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Tag            string            `json:"tag,omitempty" yaml:"tag,omitempty"`
	Group          string            `json:"group,omitempty" yaml:"group,omitempty"`
	Latency        int64             `json:"latency" yaml:"-"` // 延迟，单位ms
	LatencyHistory []int64           `json:"-" yaml:"-"`       // 历史延迟记录
	AverageLatency float64           `json:"average_latency" yaml:"-"`
	LastCheck      time.Time         `json:"last_check" yaml:"-"`
	Available      bool              `json:"available" yaml:"-"`
	Extra          map[string]string `json:"extra,omitempty" yaml:"extra,omitempty"`
}

// Subscription 订阅配置
type Subscription struct {
	URL        string    `json:"url" yaml:"url"`
	Name       string    `json:"name" yaml:"name"`
	Enabled    bool      `json:"enabled" yaml:"enabled"`
	LastUpdate time.Time `json:"last_update" yaml:"last_update"`
	Nodes      []*Node   `json:"nodes" yaml:"-"`
}

// Config MCP 主配置
type Config struct {
	Mu                         sync.Mutex      `json:"-" yaml:"-"`
	ListenAddr                 string          `json:"listen_addr" yaml:"listen_addr"`
	Subscriptions              []*Subscription `json:"subscriptions" yaml:"subscriptions"`
	Groups                     []*Group        `json:"groups" yaml:"groups"`
	Rules                      []*Rule         `json:"rules" yaml:"rules"`
	RuleStrings                []string        `json:"rule_strings,omitempty" yaml:"rule_strings,omitempty"` // Clash-format rules (type,pattern,target)
	EnableAuth                 bool            `json:"enable_auth" yaml:"enable_auth"`
	Username                   string          `json:"username" yaml:"username"`
	Password                   string          `json:"password" yaml:"password"`
	EnableAutoSelect           bool            `json:"enable_auto_select" yaml:"enable_auto_select"`
	AutoSelectInterval         int             `json:"auto_select_interval" yaml:"auto_select_interval"`                 // 单位分钟
	SubscriptionUpdateInterval int             `json:"subscription_update_interval" yaml:"subscription_update_interval"` // 单位小时
	SelectedGroup              string          `json:"selected_group" yaml:"selected_group"`
	SelectionStrategy          string          `json:"selection_strategy" yaml:"selection_strategy"` // Selection strategy: latency, round-robin, least-connections, failover, random
	EnableHTTPS                bool            `json:"enable_https" yaml:"enable_https"`
	CertFile                   string          `json:"cert_file" yaml:"cert_file"`
	KeyFile                    string          `json:"key_file" yaml:"key_file"`
	DNS                        *DNSConfig      `json:"dns,omitempty" yaml:"dns,omitempty"` // DNS configuration for anti-pollution
}

// DNSConfig DNS resolver configuration
type DNSConfig struct {
	Enable         bool     `json:"enable" yaml:"enable"`                   // Enable DNS anti-pollution
	TrustedDNS     []string `json:"trusted_dns" yaml:"trusted_dns"`         // List of trusted DNS servers
	CacheTTL       int      `json:"cache_ttl" yaml:"cache_ttl"`             // Cache TTL in minutes
	Timeout        int      `json:"timeout" yaml:"timeout"`                 // Query timeout in seconds
	CheckPollution bool     `json:"check_pollution" yaml:"check_pollution"` // Check if resolved IP is polluted
}

// Rule 分流规则
type Rule struct {
	Pattern string `json:"pattern" yaml:"pattern"`
	Target  string `json:"target" yaml:"target"` // 节点组或 DIRECT/REJECT
	Type    string `json:"type" yaml:"type"`     // domain, domain-suffix, ip-cidr, etc.
}

// Group 节点分组
type Group struct {
	Name     string  `json:"name" yaml:"name"`
	Nodes    []*Node `json:"nodes" yaml:"nodes"`
	Selected *Node   `json:"selected" yaml:"selected"`
	IsAuto   bool    `json:"is_auto" yaml:"is_auto"`
}
