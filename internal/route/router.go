package route

import (
	"net"
	"strings"
	"sync"

	"github.com/a523959108/openclaw-proxy/internal/config"
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
	// Normalize rule type
	ruleType := strings.ToLower(rule.Type)
	
	switch ruleType {
	case "domain":
		return strings.EqualFold(domain, rule.Pattern)
	case "domain-suffix", "domain_suffix":
		return strings.HasSuffix(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "domain-keyword", "domain_keyword":
		return strings.Contains(strings.ToLower(domain), strings.ToLower(rule.Pattern))
	case "ip-cidr", "ip_cidr":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "ip-cidr6", "ip_cidr6":
		if ip == nil {
			return false
		}
		_, cidrNet, err := net.ParseCIDR(rule.Pattern)
		if err != nil {
			return false
		}
		return cidrNet.Contains(ip)
	case "geoip":
		// TODO: implement GeoIP matching, currently not supported
		// Return false to match next rule
		return false
	case "src-ip-cidr":
		// Source IP matching, not applicable here
		return false
	case "dst-port":
		// Destination port matching, not applicable here
		return false
	case "src-port":
		// Source port matching, not applicable here
		return false
	case "process-name":
		// Process name matching, not applicable here
		return false
	case "process-path":
		// Process path matching, not applicable here
		return false
	default:
		// Handle uppercase Clash format (DOMAIN, DOMAIN-SUFFIX, etc.)
		switch strings.ToLower(rule.Type) {
		case "domain":
			return strings.EqualFold(domain, rule.Pattern)
		case "domain-suffix", "domain_suffix", "suffix":
			return strings.HasSuffix(strings.ToLower(domain), strings.ToLower(rule.Pattern))
		case "domain-keyword", "domain_keyword", "keyword":
			return strings.Contains(strings.ToLower(domain), strings.ToLower(rule.Pattern))
		case "ip-cidr", "ip_cidr", "cidr":
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
