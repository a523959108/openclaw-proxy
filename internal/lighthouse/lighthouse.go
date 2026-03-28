package lighthouse

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/a523959108/openclaw-proxy/internal/config"
	"github.com/a523959108/openclaw-proxy/internal/dns"
)

// Lighthouse handles latency testing
type Lighthouse struct {
	config      *config.Config
	cancelCtx   context.CancelFunc
	ctx         context.Context
	wg          sync.WaitGroup
	testURL     string
	timeout     time.Duration
	dnsResolver *dns.Resolver
}

// New creates a new lighthouse service
func New(cfg *config.Config, dnsResolver *dns.Resolver) *Lighthouse {
	ctx, cancel := context.WithCancel(context.Background())
	return &Lighthouse{
		config:  cfg,
		ctx:     ctx,
		cancelCtx: cancel,
		testURL: "http://www.gstatic.com/generate_204",
		timeout: 10 * time.Second,
		dnsResolver: dnsResolver,
	}
}
}

// SetNodeSource sets the function to get all nodes for periodic testing
func (l *Lighthouse) SetNodeSource(nodeSource func() []*config.Node) {
	// Run test every 30 minutes
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()
		
		// Run initial test after 1 minute
		time.Sleep(1 * time.Minute)
		nodes := nodeSource()
		l.TestAll(nodes)
		
		for {
			select {
			case <-l.ctx.Done():
				return
			case <-ticker.C:
				nodes := nodeSource()
				l.TestAll(nodes)
			}
		}
	}()
}

// Stop stops the service
func (l *Lighthouse) Stop() {
	l.cancelCtx()
	l.wg.Wait()
}

// TestNode tests a single node's latency
func (l *Lighthouse) TestNode(node *config.Node) (int64, error) {
	start := time.Now()

	// Check DNS pollution if enabled
	if l.config.DNS != nil && l.config.DNS.Enable && l.config.DNS.CheckPollution && l.dnsResolver != nil {
		// Resolve IP using system resolver first
		systemIPs, err := net.LookupIP(node.Server)
		if err == nil && len(systemIPs) > 0 {
			// Check if polluted
			polluted, err := l.dnsResolver.IsPolluted(node.Server, systemIPs)
			if err == nil && polluted {
				node.Available = false
				node.Latency = -1
				return -1, fmt.Errorf("node domain is polluted")
			}
		}
	}

	// TODO: implement proper connection test through the proxy
	dialer := &net.Dialer{
		Timeout: l.timeout,
	}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", node.Server, node.Port))
	if err != nil {
		node.Available = false
		node.Latency = -1
		return -1, err
	}
	defer conn.Close()

	latency := time.Since(start).Milliseconds()
	node.Latency = latency
	// Keep last 5 latency measurements for average calculation
	node.LatencyHistory = append(node.LatencyHistory, latency)
	if len(node.LatencyHistory) > 5 {
		node.LatencyHistory = node.LatencyHistory[1:]
	}
	// Calculate average latency
	var sum int64
	for _, lat := range node.LatencyHistory {
		sum += lat
	}
	node.AverageLatency = float64(sum) / float64(len(node.LatencyHistory))
	node.Available = true
	node.LastCheck = time.Now()

	return latency, nil
}

// TestAll tests all nodes
func (l *Lighthouse) TestAll(nodes []*config.Node) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent tests

	for _, node := range nodes {
		wg.Add(1)
		go func(n *config.Node) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			select {
			case <-l.ctx.Done():
				return
			default:
			}

			l.TestNode(n)
		}(node)
	}

	wg.Wait()
}
