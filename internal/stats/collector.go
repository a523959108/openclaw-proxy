package stats

import (
	"sync"
	"time"

	"github.com/a523959108/openclaw-proxy/internal/config"
)

// ConnectionStats tracks connection statistics
type ConnectionStats struct {
	TotalConnections     int64   `json:"total_connections"`
	ActiveConnections    int     `json:"active_connections"`
	TotalBytesReceived   int64   `json:"total_bytes_received"`
	TotalBytesSent       int64   `json:"total_bytes_sent"`
	TotalBytes           int64   `json:"total_bytes"`
	StartTime            time.Time `json:"start_time"`
	LastReset            time.Time `json:"last_reset"`
	ConnectionsPerSecond float64 `json:"connections_per_second"`
}

// NodeStats tracks per-node statistics
type NodeStats struct {
	NodeName           string `json:"node_name"`
	TotalConnections   int64  `json:"total_connections"`
	ActiveConnections  int    `json:"active_connections"`
	BytesReceived      int64  `json:"total_bytes_received"`
	BytesSent          int64  `json:"total_bytes_sent"`
	TotalBytes         int64  `json:"total_bytes"`
	LastUsed           time.Time `json:"last_used"`
}

// StatsCollector collects and tracks statistics
type StatsCollector struct {
	stats       *ConnectionStats
	nodeStats   map[string]*NodeStats
	mu          sync.RWMutex
}

// NewStatsCollector creates a new stats collector
func NewStatsCollector() *StatsCollector {
	now := time.Now()
	return &StatsCollector{
		stats: &ConnectionStats{
			StartTime: now,
			LastReset: now,
		},
		nodeStats: make(map[string]*NodeStats),
	}
}

// GetStats gets overall connection statistics
func (sc *StatsCollector) GetStats() *ConnectionStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	// Calculate connections per second
	elapsed := time.Since(sc.stats.LastReset).Seconds()
	if elapsed > 0 {
		sc.stats.ConnectionsPerSecond = float64(sc.stats.TotalConnections) / elapsed
	}

	copy := *sc.stats
	return &copy
}

// GetNodeStats gets statistics for all nodes
func (sc *StatsCollector) GetNodeStats() map[string]*NodeStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	result := make(map[string]*NodeStats, len(sc.nodeStats))
	for k, v := range sc.nodeStats {
		copy := *v
		result[k] = &copy
	}
	return result
}

// GetNodeStatsByName gets statistics for a specific node
func (sc *StatsCollector) GetNodeStatsByName(nodeName string) *NodeStats {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if ns, ok := sc.nodeStats[nodeName]; ok {
		copy := *ns
		return &copy
	}
	return nil
}

// OnConnectionOpen records a new connection
func (sc *StatsCollector) OnConnectionOpen(node *config.Node) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Update global stats
	sc.stats.TotalConnections++
	sc.stats.ActiveConnections++

	// Update node stats
	ns, ok := sc.nodeStats[node.Name]
	if !ok {
		ns = &NodeStats{
			NodeName: node.Name,
		}
		sc.nodeStats[node.Name] = ns
	}
	ns.TotalConnections++
	ns.ActiveConnections++
	ns.LastUsed = time.Now()
}

// OnConnectionClose records connection closed with byte counts
func (sc *StatsCollector) OnConnectionClose(node *config.Node, bytesReceived, bytesSent int64) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	// Update global stats
	if sc.stats.ActiveConnections > 0 {
		sc.stats.ActiveConnections--
	}
	sc.stats.TotalBytesReceived += bytesReceived
	sc.stats.TotalBytesSent += bytesSent
	sc.stats.TotalBytes += bytesReceived + bytesSent

	// Update node stats
	if ns, ok := sc.nodeStats[node.Name]; ok {
		if ns.ActiveConnections > 0 {
			ns.ActiveConnections--
		}
		ns.BytesReceived += bytesReceived
		ns.BytesSent += bytesSent
		ns.TotalBytes += bytesReceived + bytesSent
		ns.LastUsed = time.Now()
	}
}

// ResetStats resets all statistics
func (sc *StatsCollector) ResetStats() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	now := time.Now()
	sc.stats = &ConnectionStats{
		StartTime: sc.stats.StartTime,
		LastReset: now,
	}
	sc.nodeStats = make(map[string]*NodeStats)
}

// TotalConnections returns total connections since start
func (sc *StatsCollector) TotalConnections() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats.TotalConnections
}

// ActiveConnections returns current active connections
func (sc *StatsCollector) ActiveConnections() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats.ActiveConnections
}

// TotalBytes returns total bytes transferred
func (sc *StatsCollector) TotalBytes() int64 {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.stats.TotalBytes
}