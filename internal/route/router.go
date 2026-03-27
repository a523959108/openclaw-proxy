package route

import (
	"net"
	"strings"
	"sync"

	"openclaw-mcp/internal/config"
)

// Router handles route matching
type Router struct {
	rules []*config.Rule
	mu    sync.RWMutex
}

// NewRouter creates a new router
func NewRouter(rules []*config.Rule) *Router {
	return &Router{
		rules: rules,
	}
}

// AddRule adds a new rule
func (r *Router) AddRule(rule *config.Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
}

// SetRules replaces all rules
func (r *Router) SetRules(rules []*config.Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = rules
}

// GetRules gets all rules
func (r *Router) GetRules() []*config.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]*config.Rule, len(r.rules))
	copy(copied, r.rules)
	return copied
}

// Match matches a domain/IP to a target
func (r *Router) Match(domain string, ip net.IP) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check rules in order
	for _, rule := range r.rules {
		if r.matchRule(rule, domain, ip) {
			return rule.Target
		}
	}

	// Default to proxy
	return "proxy"
}

func (r *Router) matchRule(rule *config.Rule, domain string, ip net.IP) bool {
	switch strings.TrimPrefix(rule.Type, "DOMAIN-") {
	case "DOMAIN":
		return strings.EqualFold(domain, rule.Pattern)
	case "domain":
		return strings.EqualFold(domain, rule.Pattern)
	case "SUFFIX":
		return strings.HasSuffix(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "domain-suffix":
		return strings.HasSuffix(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "KEYWORD":
		return strings.Contains(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "domain-keyword":
		return strings.Contains(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "IP-CIDR":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "ip-cidr":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "IP-CIDR6":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "ip-cidr6":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "GEOIP":
		// TODO: implement GeoIP matching, currently not supported
		// Return false to match next rule
		return false
	case "SRC-IP-CIDR":
		// Source IP matching, not applicable here
		return false
	case "DST-PORT":
		// Destination port matching, not applicable here
		return false
	case "SRC-PORT":
		// Source port matching, not applicable here
		return false
	case "PROCESS-NAME":
		// Process name matching, not applicable here
		return false
	case "PROCESS-PATH":
		// Process path matching, not applicable here
		return false
	default:
		// Handle other formats by case insensitive matching
		lowerType := strings.ToLower(rule.Type)
		lowerPattern := strings.ToLower(rule.Pattern)
		switch lowerType {
		case "domain":
			return strings.EqualFold(domain, rule.Pattern)
		case "domain_suffix", "suffix":
			return strings.HasSuffix(strings.ToLower(domain), lowerPattern)
		case "domain_keyword", "keyword":
			return strings.Contains(strings.ToLower(domain), lowerPattern)
		case "ip_cidr", "cidr":
			if ip == nil {
				return false
			}
			_, cidrNet, err := net.ParseCIDR(rule.Pattern)
			if err != nil {
				return false
			}
			return cidrNet.Contains(ip)
		}
		return false
	}
}

// LoadRulesFromClashYAML loads rules from Clash YAML format string array
func (r *Router) LoadRulesFromClashYAML(rawRules []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Parse Clash-format rules
	for _, rawRule := range rawRules {
		rawRule = strings.TrimSpace(rawRule)
		if rawRule == "" || strings.HasPrefix(rawRule, "#") {
			continue
		}
		// Format: type,pattern,target[,no-resolve]
		parts := strings.SplitN(rawRule, ",", 4)
		if len(parts) < 3 {
			continue
		}
		ruleType := strings.TrimSpace(parts[0])
		pattern := strings.TrimSpace(parts[1])
		target := strings.TrimSpace(parts[2])
		
		rule := &config.Rule{
			Type:    ruleType,
			Pattern: pattern,
			Target:  target,
		}
		r.rules = append(r.rules, rule)
	}
	return nil
}
