package selection

import (
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/a523959108/openclaw-proxy/internal/config"
)

// SelectionStrategy defines different node selection strategies
type SelectionStrategy string

const (
	// StrategyLatency selects node based on lowest latency (default)
	StrategyLatency SelectionStrategy = "latency"
	// StrategyRoundRobin selects nodes in round-robin order
	StrategyRoundRobin SelectionStrategy = "round-robin"
	// StrategyLeastConnections selects node with least active connections
	StrategyLeastConnections SelectionStrategy = "least-connections"
	// StrategyFailover always selects first available node, only failover when down
	StrategyFailover SelectionStrategy = "failover"
	// StrategyRandom randomly selects an available node
	StrategyRandom SelectionStrategy = "random"
)

// ConnectionTracker tracks active connections for each node
type ConnectionTracker struct {
	connections map[string]int // node name -> active connections
	mu          sync.RWMutex
}

// NewConnectionTracker creates a new connection tracker
func NewConnectionTracker() *ConnectionTracker {
	return &ConnectionTracker{
		connections: make(map[string]int),
	}
}

// AddConnection increments connection count for a node
func (ct *ConnectionTracker) AddConnection(node *config.Node) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.connections[node.Name]++
}

// RemoveConnection decrements connection count for a node
func (ct *ConnectionTracker) RemoveConnection(node *config.Node) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	if count, ok := ct.connections[node.Name]; ok && count > 0 {
		ct.connections[node.Name]--
	}
}

// GetConnections gets current connection count for a node
func (ct *ConnectionTracker) GetConnections(node *config.Node) int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.connections[node.Name]
}

// GetAllConnections gets all connection counts
func (ct *ConnectionTracker) GetAllConnections() map[string]int {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	result := make(map[string]int, len(ct.connections))
	for k, v := range ct.connections {
		result[k] = v
	}
	return result
}

// Reset resets all connection counts
func (ct *ConnectionTracker) Reset() {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.connections = make(map[string]int)
}

// StrategySelector selects node based on different strategies
type StrategySelector struct {
	strategy      SelectionStrategy
	tracker       *ConnectionTracker
	roundRobinIdx int
	mu            sync.Mutex
	rand          *rand.Rand
	lastFailover  int // Last selected node index for failover
}

// NewStrategySelector creates a new strategy selector
func NewStrategySelector(strategy SelectionStrategy) *StrategySelector {
	return &StrategySelector{
		strategy: strategy,
		tracker:  NewConnectionTracker(),
		rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select selects a node from available nodes based on strategy
func (ss *StrategySelector) Select(availableNodes []*config.Node) (*config.Node, error) {
	if len(availableNodes) == 0 {
		return nil, errors.New("no available nodes")
	}

	switch ss.strategy {
	case StrategyLatency:
		return ss.selectByLatency(availableNodes), nil
	case StrategyRoundRobin:
		return ss.selectRoundRobin(availableNodes), nil
	case StrategyLeastConnections:
		return ss.selectLeastConnections(availableNodes), nil
	case StrategyFailover:
		return ss.selectFailover(availableNodes), nil
	case StrategyRandom:
		return ss.selectRandom(availableNodes), nil
	default:
		return ss.selectByLatency(availableNodes), nil
	}
}

// selectByLatency sorts by latency and selects the lowest
func (ss *StrategySelector) selectByLatency(nodes []*config.Node) *config.Node {
	// Sort by average latency, then by current latency
	sorted := make([]*config.Node, len(nodes))
	copy(sorted, nodes)
	// Sort based on existing sort order already done by caller
	if len(sorted) > 0 {
		return sorted[0]
	}
	return nil
}

// selectRoundRobin rotates through nodes
func (ss *StrategySelector) selectRoundRobin(nodes []*config.Node) *config.Node {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	idx := ss.roundRobinIdx % len(nodes)
	ss.roundRobinIdx = (ss.roundRobinIdx + 1) % len(nodes)
	return nodes[idx]
}

// selectLeastConnections selects node with least active connections
// If tie, break by latency
func (ss *StrategySelector) selectLeastConnections(nodes []*config.Node) *config.Node {
	var selected *config.Node
	minConns := int(^uint(0) >> 1) // Max int

	for _, node := range nodes {
		conns := ss.tracker.GetConnections(node)
		if selected == nil || conns < minConns {
			minConns = conns
			selected = node
		} else if conns == minConns && selected != nil {
			// Break tie by latency
			if node.AverageLatency > 0 && selected.AverageLatency > 0 {
				if node.AverageLatency < selected.AverageLatency {
					selected = node
				}
			} else if node.Latency > 0 && selected.Latency > 0 {
				if node.Latency < selected.Latency {
					selected = node
				}
			}
		}
	}

	return selected
}

// selectFailover always selects first available node starting from last known good
func (ss *StrategySelector) selectFailover(nodes []*config.Node) *config.Node {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	// Try starting from last selected, then wrap around
	for i := 0; i < len(nodes); i++ {
		idx := (ss.lastFailover + i) % len(nodes)
		if nodes[idx].Available {
			ss.lastFailover = idx
			return nodes[idx]
		}
	}

	// If none available, return first anyway
	ss.lastFailover = 0
	return nodes[0]
}

// selectRandom randomly selects an available node
func (ss *StrategySelector) selectRandom(nodes []*config.Node) *config.Node {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	idx := ss.rand.Intn(len(nodes))
	return nodes[idx]
}

// GetStrategy returns current selection strategy
func (ss *StrategySelector) GetStrategy() SelectionStrategy {
	return ss.strategy
}

// SetStrategy changes selection strategy
func (ss *StrategySelector) SetStrategy(strategy SelectionStrategy) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.strategy = strategy
}

// GetConnectionTracker returns the connection tracker
func (ss *StrategySelector) GetConnectionTracker() *ConnectionTracker {
	return ss.tracker
}