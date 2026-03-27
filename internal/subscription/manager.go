package subscription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"openclaw-mcp/internal/config"
)

// Manager handles subscription management
type Manager struct {
	config *config.Config
	cancel context.CancelFunc
	mu     sync.Mutex
}

// NewManager creates a new subscription manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

// UpdateAll updates all subscriptions
func (m *Manager) UpdateAll() error {
	var lastErr error
	for _, sub := range m.config.Subscriptions {
		if !sub.Enabled {
			continue
		}
		if err := m.Update(sub); err != nil {
			lastErr = err
			// Continue updating other subscriptions even if one fails
		}
	}
	return lastErr
}

var globalHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

// Update updates a single subscription
func (m *Manager) Update(sub *config.Subscription) error {
	resp, err := globalHTTPClient.Get(sub.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Try base64 decoding for standard subscription format
	contentStr := string(content)
	if decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(contentStr)); err == nil {
		contentStr = string(decoded)
	}

	nodes, err := m.parseNodes(contentStr)
	if err != nil {
		return err
	}

	sub.Nodes = nodes
	sub.LastUpdate = time.Now()
	return nil
}

// AddSubscription adds a new subscription
func (m *Manager) AddSubscription(sub *config.Subscription) error {
	if err := m.Update(sub); err != nil {
		return err
	}
	m.config.Subscriptions = append(m.config.Subscriptions, sub)
	return nil
}

// StartAutoUpdate starts automatic periodic subscription update
func (m *Manager) StartAutoUpdate(ctx context.Context) {
	if m.config.SubscriptionUpdateInterval <= 0 {
		return // disabled
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	interval := time.Duration(m.config.SubscriptionUpdateInterval) * time.Hour
	ticker := time.NewTicker(interval)
	var updateCtx context.Context
	updateCtx, m.cancel = context.WithCancel(ctx)
	go func() {
		for {
			select {
			case <-updateCtx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				m.UpdateAll()
			}
		}
	}()
}

// StopAutoUpdate stops automatic subscription update
func (m *Manager) StopAutoUpdate() {
	if m.cancel != nil {
		m.cancel()
	}
}

// RemoveSubscription removes a subscription
func (m *Manager) RemoveSubscription(index int) {
	if index >= 0 && index < len(m.config.Subscriptions) {
		m.config.Subscriptions = append(m.config.Subscriptions[:index], m.config.Subscriptions[index+1:]...)
	}
}

// GetAllNodes returns all nodes from all enabled subscriptions
func (m *Manager) GetAllNodes() []*config.Node {
	var nodes []*config.Node
	for _, sub := range m.config.Subscriptions {
		if sub.Enabled {
			nodes = append(nodes, sub.Nodes...)
		}
	}
	return nodes
}

// GetAllNodesByGroup returns all nodes that belong to specific group from all enabled subscriptions
func (m *Manager) GetAllNodesByGroup(group string) []*config.Node {
	var nodes []*config.Node
	for _, sub := range m.config.Subscriptions {
		if sub.Enabled {
			for _, node := range sub.Nodes {
				if node.Group == group || group == "auto" || group == "" {
					nodes = append(nodes, node)
				}
			}
		}
	}
	return nodes
}

// parseNodes parses nodes from subscription content
func (m *Manager) parseNodes(content string) ([]*config.Node, error) {
	var nodes []*config.Node

	// Split by lines for multiple links
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var node *config.Node
		var err error

		switch {
		case strings.HasPrefix(line, "vmess://"):
			node, err = parseVmess(line)
		case strings.HasPrefix(line, "ss://"):
			node, err = parseShadowsocks(line)
		case strings.HasPrefix(line, "vless://"):
			node, err = parseVLESS(line)
		case strings.HasPrefix(line, "trojan://"):
			node, err = parseTrojan(line)
		case strings.HasPrefix(line, "hysteria://"):
			node, err = parseHysteria(line)
		case strings.HasPrefix(line, "hysteria2://"):
			node, err = parseHysteria2(line)
		case strings.HasPrefix(line, "tuic://"):
			node, err = parseTuic(line)
		default:
			continue
		}

		if err == nil && node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// parseShadowsocks parses ss:// URL
func parseShadowsocks(link string) (*config.Node, error) {
	link = strings.TrimPrefix(link, "ss://")
	nameIndex := strings.Index(link, "#")
	var name string
	if nameIndex != -1 {
		name, _ = url.PathUnescape(link[nameIndex+1:])
		link = link[:nameIndex]
	}

	// Two formats: base64(userinfo@server:port) or userinfo@server:port
	var userInfo string
	var serverPort string
	if i := strings.Index(link, "@"); i != -1 {
		userInfo = link[:i]
		serverPort = link[i+1:]
	} else {
		// Old format: entire thing is base64
		decoded, err := base64.StdEncoding.DecodeString(link)
		if err != nil {
			return nil, err
		}
		decodedStr := string(decoded)
		if i := strings.Index(decodedStr, "@"); i != -1 {
			userInfo = decodedStr[:i]
			serverPort = decodedStr[i+1:]
		} else {
			return nil, fmt.Errorf("invalid ss format")
		}
	}

	methodPassword, err := base64.StdEncoding.DecodeString(userInfo)
	if err != nil {
		methodPassword = []byte(userInfo)
	}
	parts := strings.SplitN(string(methodPassword), ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid ss userinfo")
	}

	hostPortParts := strings.Split(serverPort, ":")
	if len(hostPortParts) != 2 {
		return nil, fmt.Errorf("invalid ss server:port")
	}
	port, _ := strconv.Atoi(hostPortParts[1])

	if name == "" {
		name = fmt.Sprintf("%s:%d", hostPortParts[0], port)
	}

	return &config.Node{
		Name:     name,
		Type:     "ss",
		Server:   hostPortParts[0],
		Port:     port,
		Password: parts[1],
		Security: parts[0],
		Available: true,
	}, nil
}

// parseVmess parses vmess:// URL
func parseVmess(link string) (*config.Node, error) {
	link = strings.TrimPrefix(link, "vmess://")

	// Handle padding
	if len(link)%4 != 0 {
		link += strings.Repeat("=", 4-len(link)%4)
	}

	data, err := base64.StdEncoding.DecodeString(link)
	if err != nil {
		return nil, err
	}

	var vmess struct {
		PS   string `json:"ps"`
		Add  string `json:"add"`
		Port string `json:"port"`
		ID   string `json:"id"`
		Aid  string `json:"aid"`
		Net  string `json:"net"`
		Type string `json:"type"`
		Host string `json:"host"`
		Path string `json:"path"`
		TLS  string `json:"tls"`
		SNI  string `json:"sni"`
	}

	if err := json.Unmarshal(data, &vmess); err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(vmess.Port)
	aid, _ := strconv.Atoi(vmess.Aid)
	tls := vmess.TLS == "tls"

	return &config.Node{
		Name:      vmess.PS,
		Type:      "vmess",
		Server:    vmess.Add,
		Port:      port,
		UUID:      vmess.ID,
		AlterID:   aid,
		Security:  "auto",
		Network:   vmess.Net,
		Path:      vmess.Path,
		Host:      vmess.Host,
		TLS:       tls,
		ServerName: vmess.SNI,
		Available:  true,
	}, nil
}

// parseVLESS parses vless:// URL
func parseVLESS(link string) (*config.Node, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(parsedURL.Port())
	uuid := parsedURL.User.Username()
	name := parsedURL.Fragment

	query := parsedURL.Query()
	security := query.Get("security")
	path := query.Get("path")
	host := query.Get("host")
	sni := query.Get("sni")
	tls := security == "tls"

	return &config.Node{
		Name:      name,
		Type:      "vless",
		Server:    parsedURL.Hostname(),
		Port:      port,
		UUID:      uuid,
		Security:  security,
		Path:      path,
		Host:      host,
		ServerName: sni,
		TLS:       tls,
		Available: true,
	}, nil
}

// parseTrojan parses trojan:// URL
func parseTrojan(link string) (*config.Node, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(parsedURL.Port())
	password := parsedURL.User.Username()
	name := parsedURL.Fragment

	query := parsedURL.Query()
	sni := query.Get("sni")

	return &config.Node{
		Name:       name,
		Type:       "trojan",
		Server:     parsedURL.Hostname(),
		Port:       port,
		Password:   password,
		ServerName: sni,
		TLS:        true,
		Available:  true,
	}, nil
}

// parseHysteria parses hysteria:// URL
func parseHysteria(link string) (*config.Node, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(parsedURL.Port())
	auth := parsedURL.User.Username()
	name := parsedURL.Fragment

	query := parsedURL.Query()
	protocol := query.Get("protocol")
	obfs := query.Get("obfs")
	sni := query.Get("sni")
	upmbps, _ := strconv.Atoi(query.Get("upmbps"))
	downmbps, _ := strconv.Atoi(query.Get("downmbps"))

	extra := make(map[string]string)
	if protocol != "" {
		extra["protocol"] = protocol
	}
	if obfs != "" {
		extra["obfs"] = obfs
	}
	if upmbps > 0 {
		extra["upmbps"] = strconv.Itoa(upmbps)
	}
	if downmbps > 0 {
		extra["downmbps"] = strconv.Itoa(downmbps)
	}

	return &config.Node{
		Name:       name,
		Type:       "hysteria",
		Server:     parsedURL.Hostname(),
		Port:       port,
		Password:   auth,
		ServerName: sni,
		TLS:        true,
		Extra:      extra,
		Available:  true,
	}, nil
}

// parseHysteria2 parses hysteria2:// URL
func parseHysteria2(link string) (*config.Node, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(parsedURL.Port())
	password := parsedURL.User.Username()
	name := parsedURL.Fragment

	query := parsedURL.Query()
	sni := query.Get("sni")
	obfs := query.Get("obfs")

	extra := make(map[string]string)
	if obfs != "" {
		extra["obfs"] = obfs
	}

	return &config.Node{
		Name:       name,
		Type:       "hysteria2",
		Server:     parsedURL.Hostname(),
		Port:       port,
		Password:   password,
		ServerName: sni,
		TLS:        true,
		Extra:      extra,
		Available:  true,
	}, nil
}

// parseTuic parses tuic:// URL
func parseTuic(link string) (*config.Node, error) {
	parsedURL, err := url.Parse(link)
	if err != nil {
		return nil, err
	}

	port, _ := strconv.Atoi(parsedURL.Port())
	uuid := parsedURL.User.Username()
	password, _ := parsedURL.User.Password()
	name := parsedURL.Fragment

	query := parsedURL.Query()
	sni := query.Get("sni")
	alpn := query.Get("alpn")

	extra := make(map[string]string)
	if alpn != "" {
		extra["alpn"] = alpn
	}

	return &config.Node{
		Name:       name,
		Type:       "tuic",
		Server:     parsedURL.Hostname(),
		Port:       port,
		UUID:       uuid,
		Password:   password,
		ServerName: sni,
		TLS:        true,
		Extra:      extra,
		Available:  true,
	}, nil
}
