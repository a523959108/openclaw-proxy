package config

import (
	"time"
)

// Node 代表一个代理节点
type Node struct {
	Name        string            `json:"name" yaml:"name"`
	Type        string            `json:"type" yaml:"type"` // ss, vmess, vless, trojan, hysteria, hysteria2, tuic
	Server      string            `json:"server" yaml:"server"`
	Port        int               `json:"port" yaml:"port"`
	Password    string            `json:"password,omitempty" yaml:"password,omitempty"`
	UUID        string            `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	AlterID     int               `json:"alterId,omitempty" yaml:"alterId,omitempty"`
	Security    string            `json:"security,omitempty" yaml:"security,omitempty"`
	Network     string            `json:"network,omitempty" yaml:"network,omitempty"`
	Path        string            `json:"path,omitempty" yaml:"path,omitempty"`
	Host        string            `json:"host,omitempty" yaml:"host,omitempty"`
	TLS         bool              `json:"tls,omitempty" yaml:"tls,omitempty"`
	ServerName  string            `json:"servername,omitempty" yaml:"servername,omitempty"`
	ALPN        []string          `json:"alpn,omitempty" yaml:"alpn,omitempty"`
	FP          string            `json:"fingerprint,omitempty" yaml:"fingerprint,omitempty"`
	Tag         string            `json:"tag,omitempty" yaml:"tag,omitempty"`
	Latency     int64             `json:"latency" yaml:"-"` // 延迟，单位ms
	LastCheck   time.Time         `json:"last_check" yaml:"-"`
	Available   bool              `json:"available" yaml:"-"`
	Extra       map[string]string `json:"extra,omitempty" yaml:"extra,omitempty"`
}

// Subscription 订阅配置
type Subscription struct {
	URL      string    `json:"url" yaml:"url"`
	Name     string    `json:"name" yaml:"name"`
	Enabled  bool      `json:"enabled" yaml:"enabled"`
	LastUpdate time.Time `json:"last_update" yaml:"last_update"`
	Nodes    []*Node   `json:"nodes" yaml:"-"`
}

// Config MCP 主配置
type Config struct {
	ListenAddr     string          `json:"listen_addr" yaml:"listen_addr"`
	Subscriptions  []*Subscription `json:"subscriptions" yaml:"subscriptions"`
	Rules          []*Rule         `json:"rules" yaml:"rules"`
	EnableAutoSelect bool          `json:"enable_auto_select" yaml:"enable_auto_select"`
	AutoSelectInterval int         `json:"auto_select_interval" yaml:"auto_select_interval"` // 单位分钟
	SelectedGroup  string          `json:"selected_group" yaml:"selected_group"`
}

// Rule 分流规则
type Rule struct {
	Pattern string `json:"pattern" yaml:"pattern"`
	Target  string `json:"target" yaml:"target"` // 节点组或 DIRECT/REJECT
	Type    string `json:"type" yaml:"type"`     // domain, domain-suffix, ip-cidr, etc.
}

// Group 节点分组
type Group struct {
	Name    string  `json:"name" yaml:"name"`
	Nodes   []*Node `json:"nodes" yaml:"nodes"`
	Selected *Node  `json:"selected" yaml:"selected"`
	IsAuto  bool    `json:"is_auto" yaml:"is_auto"`
}
