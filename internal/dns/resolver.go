package dns

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// Resolver handles DNS resolution with pollution protection
type Resolver struct {
	trustedDNS  []string
	client      *dns.Client
	cache       map[string]*dnsCacheEntry
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration
	timeout     time.Duration
	ctx         context.Context
	cancel      context.CancelFunc
}

type dnsCacheEntry struct {
	answer  *dns.Msg
	expires time.Time
}

// Config is DNS resolver configuration
type Config struct {
	TrustedDNS []string      `json:"trusted_dns" yaml:"trusted_dns"`
	CacheTTL   int           `json:"cache_ttl" yaml:"cache_ttl"` // cache TTL in minutes
	Timeout    int           `json:"timeout" yaml:"timeout"`     // timeout in seconds
}

// DefaultConfig returns default DNS configuration
func DefaultConfig() *Config {
	return &Config{
		TrustedDNS: []string{
			"8.8.8.8:53",
			"1.1.1.1:53",
		},
		CacheTTL: 30,
		Timeout:  5,
	}
}

// NewResolver creates a new DNS resolver
func NewResolver(cfg *Config) *Resolver {
	ctx, cancel := context.WithCancel(context.Background())

	cfg = fillDefaultConfig(cfg)

	r := &Resolver{
		trustedDNS: cfg.TrustedDNS,
		client: &dns.Client{
			Net:     "udp",
			Timeout: time.Duration(cfg.Timeout) * time.Second,
		},
		cache:    make(map[string]*dnsCacheEntry),
		cacheTTL: time.Duration(cfg.CacheTTL) * time.Minute,
		timeout:  time.Duration(cfg.Timeout) * time.Second,
		ctx:      ctx,
		cancel:   cancel,
	}

	// Start periodic cache cleanup
	go r.cleanupCache()

	return r
}

func fillDefaultConfig(cfg *Config) *Config {
	if cfg == nil {
		return DefaultConfig()
	}
	if len(cfg.TrustedDNS) == 0 {
		cfg.TrustedDNS = DefaultConfig().TrustedDNS
	}
	if cfg.CacheTTL <= 0 {
		cfg.CacheTTL = DefaultConfig().CacheTTL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultConfig().Timeout
	}
	return cfg
}

// LookupIP resolves a domain name to IP addresses using trusted DNS
func (r *Resolver) LookupIP(domain string) ([]net.IP, error) {
	// Check cache first
	if ips, ok := r.getFromCache(domain); ok {
		return ips, nil
	}

	// Query all trusted DNS in parallel, return the first valid response
	return r.parallelLookup(domain)
}

// LookupIPv4 resolves a domain name to IPv4 addresses only
func (r *Resolver) LookupIPv4(domain string) ([]net.IP, error) {
	ips, err := r.LookupIP(domain)
	if err != nil {
		return nil, err
	}

	var ipv4s []net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4s = append(ipv4s, ip)
		}
	}
	return ipv4s, nil
}

// IsPolluted checks if a domain is likely poisoned by checking against trusted DNS
func (r *Resolver) IsPolluted(domain string, checkIPs []net.IP) (bool, error) {
	trustedIPs, err := r.LookupIP(domain)
	if err != nil {
		return false, err
	}

	// If any of the check IPs is not in trusted IPs, it's likely polluted
	// Or if trusted IPs is empty, consider it polluted
	if len(trustedIPs) == 0 {
		return true, nil
	}

	for _, checkIP := range checkIPs {
		found := false
		for _, trustedIP := range trustedIPs {
			if checkIP.Equal(trustedIP) {
				found = true
				break
			}
		}
		if !found {
			return true, nil
		}
	}

	return false, nil
}

func (r *Resolver) parallelLookup(domain string) ([]net.IP, error) {
	type result struct {
		ips []net.IP
		err error
	}

	resultChan := make(chan result, len(r.trustedDNS))

	for _, dnsServer := range r.trustedDNS {
		go func(server string) {
			ips, err := r.queryDNS(server, domain)
			select {
			case resultChan <- result{ips, err}:
			case <-r.ctx.Done():
			}
		}(dnsServer)
	}

	// Return the first successful result with overall timeout
	var lastErr error
	timeout := time.After(r.timeout)
	
	for i := 0; i < len(r.trustedDNS); i++ {
		select {
		case res := <-resultChan:
			if res.err == nil && len(res.ips) > 0 {
				r.putToCache(domain, res.ips)
				return res.ips, nil
			}
			if res.err != nil {
				lastErr = res.err
			}
		case <-r.ctx.Done():
			return nil, context.Canceled
		case <-timeout:
			return nil, context.DeadlineExceeded
		}
	}

	return nil, lastErr
}

func (r *Resolver) queryDNS(dnsServer, domain string) ([]net.IP, error) {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)

	resp, _, err := r.client.Exchange(m, dnsServer)
	if err != nil {
		return nil, err
	}

	if resp.Rcode != dns.RcodeSuccess {
		return nil, nil
	}

	var ips []net.IP
	for _, ans := range resp.Answer {
		switch a := ans.(type) {
		case *dns.A:
			ips = append(ips, a.A)
		case *dns.AAAA:
			ips = append(ips, a.AAAA)
		}
	}

	return ips, nil
}

func (r *Resolver) getFromCache(domain string) ([]net.IP, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	entry, ok := r.cache[domain]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expires) {
		return nil, false
	}

	var ips []net.IP
	for _, answer := range entry.answer.Answer {
		switch a := answer.(type) {
		case *dns.A:
			ips = append(ips, a.A)
		case *dns.AAAA:
			ips = append(ips, a.AAAA)
		}
	}

	return ips, true
}

func (r *Resolver) putToCache(domain string, ips []net.IP) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	m := new(dns.Msg)
	m.Response = true
	for _, ip := range ips {
		if ip4 := ip.To4(); ip4 != nil {
			a := new(dns.A)
			a.Hdr = dns.RR_Header{Name: dns.Fqdn(domain), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 300}
			a.A = ip4
			m.Answer = append(m.Answer, a)
		} else {
			aaaa := new(dns.AAAA)
			aaaa.Hdr = dns.RR_Header{Name: dns.Fqdn(domain), Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 300}
			aaaa.AAAA = ip
			m.Answer = append(m.Answer, aaaa)
		}
	}

	r.cache[domain] = &dnsCacheEntry{
		answer:  m,
		expires: time.Now().Add(r.cacheTTL),
	}
}

func (r *Resolver) cleanupCache() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.doCleanup()
		case <-r.ctx.Done():
			return
		}
	}
}

func (r *Resolver) doCleanup() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	now := time.Now()
	for domain, entry := range r.cache {
		if now.After(entry.expires) {
			delete(r.cache, domain)
		}
	}
}

// Stop stops the resolver
func (r *Resolver) Stop() {
	r.cancel()
}

// ClearCache clears all cached DNS entries
func (r *Resolver) ClearCache() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[string]*dnsCacheEntry)
}