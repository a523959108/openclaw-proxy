package route

import (
	"net"
	"strings"

	"openclaw-mcp/internal/config"
)

// Router handles route matching
type Router struct {
	rules []*config.Rule
}

// NewRouter creates a new router
func NewRouter(rules []*config.Rule) *Router {
	return &Router{
		rules: rules,
	}
}

// AddRule adds a new rule
func (r *Router) AddRule(rule *config.Rule) {
	r.rules = append(r.rules, rule)
}

// Match matches a domain/IP to a target
func (r *Router) Match(domain string, ip string) string {
	// Check rules in order
	for _, rule := range r.rules {
		if r.matchRule(rule, domain, ip) {
			return rule.Target
		}
	}

	// Default to proxy
	return "proxy"
}

func (r *Router) matchRule(rule *config.Rule, domain string, ip string) bool {
	switch rule.Type {
	case "domain":
		return domain == rule.Pattern
	case "domain-suffix":
		return strings.HasSuffix(domain, rule.Pattern)
	case "domain-keyword":
		return strings.Contains(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "ip-cidr":
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil || ip == "" {
			return false
		}
		ipAddr := net.ParseIP(ip)
		return cidrNet.Contains(ipAddr)
	default:
		return false
	}
}

// LoadRulesFromYAML loads rules from YAML format compatible with Clash
func (r *Router) LoadRulesFromYAML(rawRules []string) error {
	// Parse Clash-format rules
	for _, rawRule := range rawRules {
		// Format: type,pattern,target
		parts := strings.SplitN(rawRule, ",", 3)
		if len(parts) != 3 {
			continue
		}
		rule := &config.Rule{
			Type:    strings.TrimSpace(parts[0]),
			Pattern: strings.TrimSpace(parts[1]),
			Target:  strings.TrimSpace(parts[2]),
		}
		r.AddRule(rule)
	}
	return nil
}
