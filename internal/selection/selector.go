package selection

import (
	"context"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/a523959108/openclaw-proxy/internal/config"
	"github.com/a523959108/openclaw-proxy/internal/dns"
	"github.com/a523959108/openclaw-proxy/internal/lighthouse"
	"github.com/a523959108/openclaw-proxy/internal/subscription"
)

// Selector handles automatic node selection
type Selector struct {
	config     *config.Config
	subManager *subscription.Manager
	lighthouse *lighthouse.Lighthouse
	dnsResolver *dns.Resolver
	strategySelector *StrategySelector
	cancelCtx  context.CancelFunc
	ctx        context.Context
	wg         sync.WaitGroup
	current    *config.Node
	mu         sync.RWMutex
}

// New creates a new selector
func New(cfg *config.Config, lh *lighthouse.Lighthouse, sm *subscription.Manager, dr *dns.Resolver) *Selector {
	ctx, cancel := context.WithCancel(context.Background())
	strategy := SelectionStrategy(cfg.SelectionStrategy)
	return &Selector{
		config:           cfg,
		subManager:       sm,
		lighthouse:       lh,
		dnsResolver:      dr,
		strategySelector: NewStrategySelector(strategy),
		ctx:              ctx,
		cancelCtx:        cancel,
	}
}

// GetBestNode selects the best available node based on average latency
func (s *Selector) GetBestNode() *config.Node {
	return s.GetBestNodeForGroup("")
}

// GetBestNodeForGroup selects the best node for a specific group
func (s *Selector) GetBestNodeForGroup(groupName string) *config.Node {
	nodes := s.subManager.GetAllNodes()
	if len(nodes) == 0 {
		return nil
	}

	availableNodes := make([]*config.Node, 0)
	// Check DNS pollution if enabled
	availableNodes = s.filterPollutedNodes(nodes, groupName)

	if len(availableNodes) == 0 {
		// If no tested/clean nodes available, filter untested by group
		for _, node := range nodes {
			if groupName == "" || node.Group == groupName {
				// Still check pollution if enabled
				if s.dnsResolver != nil && s.config.DNS != nil && s.config.DNS.CheckPollution {
					if ips, err := s.resolveNodeServer(node); err == nil && !s.checkPolluted(node.Server, ips) {
						return node
					}
				} else {
					return node
				}
			}
		}
		// If all nodes are polluted, fallback to first node
		for _, node := range nodes {
			if groupName == "" || node.Group == groupName {
				return node
			}
		}
		return nodes[0] // fallback to first node if no match
	}

	// Sort by average latency for more stable selection for latency-based strategy
	if s.strategySelector.GetStrategy() == StrategyLatency {
		sort.Slice(availableNodes, func(i, j int) bool {
			if availableNodes[i].AverageLatency > 0 && availableNodes[j].AverageLatency > 0 {
				return availableNodes[i].AverageLatency < availableNodes[j].AverageLatency
			}
			return availableNodes[i].Latency < availableNodes[j].Latency
		})
	}

	// Use strategy selector to pick the final node
	node, err := s.strategySelector.Select(availableNodes)
	if err != nil && len(availableNodes) > 0 {
		return availableNodes[0]
	}
	return node
}

// filterPollutedNodes filters out nodes that have polluted DNS results
func (s *Selector) filterPollutedNodes(nodes []*config.Node, groupName string) []*config.Node {
	filtered := make([]*config.Node, 0)

	// If DNS pollution check is not enabled, just do original filtering
	if s.dnsResolver == nil || s.config.DNS == nil || !s.config.DNS.Enable || !s.config.DNS.CheckPollution {
		for _, node := range nodes {
			if node.Available {
				if groupName == "" || node.Group == groupName {
					filtered = append(filtered, node)
				}
			}
		}
		return filtered
	}

	// Check each node
	for _, node := range nodes {
		if node.Available {
			if groupName != "" && node.Group != groupName {
				continue
			}

			// Resolve domain and check if it's polluted
			ips, err := s.resolveNodeServer(node)
			if err != nil {
				// If resolution fails, keep the node but warn
				filtered = append(filtered, node)
				continue
			}

			if !s.checkPolluted(node.Server, ips) {
				filtered = append(filtered, node)
			}
		}
	}

	return filtered
}

// resolveNodeServer resolves node server domain to IPs
func (s *Selector) resolveNodeServer(node *config.Node) ([]net.IP, error) {
	// If node.Server is already an IP, no need to resolve
	if net.ParseIP(node.Server) != nil {
		return []net.IP{net.ParseIP(node.Server)}, nil
	}

	// Use trusted DNS to resolve
	return s.dnsResolver.LookupIP(node.Server)
}

// checkPolluted checks if any IP is mismatched with trusted DNS
func (s *Selector) checkPolluted(domain string, ips []net.IP) bool {
	isPolluted, err := s.dnsResolver.IsPolluted(domain, ips)
	if err != nil {
		return false // If check fails, assume not polluted
	}
	return isPolluted
}

// SelectBest forces a new selection
func (s *Selector) SelectBest() *config.Node {
	s.mu.Lock()
	defer s.mu.Unlock()

	// First test all nodes
	nodes := s.subManager.GetAllNodes()
	s.lighthouse.TestAll(nodes)

	// Then select best
	s.current = s.GetBestNode()
	return s.current
}

// SelectBestForGroup selects best node for a specific group
func (s *Selector) SelectBestForGroup(groupName string) *config.Node {
	s.mu.Lock()
	defer s.mu.Unlock()

	nodes := s.subManager.GetAllNodesByGroup(groupName)
	s.lighthouse.TestAll(nodes)
	node := s.GetBestNodeForGroup(groupName)
	return node
}

// GetCurrent returns the currently selected node
func (s *Selector) GetCurrent() *config.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// StartAutoSelection starts automatic periodic selection
func (s *Selector) StartAutoSelection() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		interval := time.Duration(s.config.AutoSelectInterval) * time.Minute
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Do initial selection
		s.SelectBest()

		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.SelectBest()
			}
		}
	}()
}

// StopAutoSelection stops automatic periodic selection
func (s *Selector) StopAutoSelection() {
	s.cancelCtx()
	s.wg.Wait()

	// Create new context for potential restart
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
}

// GetAllGroups returns all configured groups plus auto groups based on node tags
func (s *Selector) GetAllGroups() []*config.Group {
	groups := make([]*config.Group, len(s.config.Groups))
	copy(groups, s.config.Groups)

	// Auto add group 'auto' if not exists
	hasAuto := false
	for _, g := range groups {
		if g.Name == "auto" {
			hasAuto = true
			break
		}
	}
	if !hasAuto {
		groups = append(groups, &config.Group{
			Name:   "auto",
			IsAuto: true,
		})
	}

	// Auto populate nodes for automatic groups
	for _, group := range groups {
		if group.IsAuto {
			group.Nodes = s.subManager.GetAllNodesByGroup(group.Name)
			group.Selected = s.GetBestNodeForGroup(group.Name)
		}
	}

	return groups
}

// SelectBestForGroup selects and sets current node for a group
func (s *Selector) SelectBestForGroup(groupName string) *config.Node {
	return s.GetBestNodeForGroup(groupName)
}

// CountAvailableNodes counts all available nodes
func (s *Selector) CountAvailableNodes() int {
	nodes := s.subManager.GetAllNodes()
	count := 0
	for _, node := range nodes {
		if node.Available {
			count++
		}
	}
	return count
}

// SelectByRegion selects nodes from specific region
func (s *Selector) SelectByRegion(region string) *config.Node {
	// TODO: implement region-based selection
	return s.GetBestNode()
}

// GroupByRegion groups nodes by detected region
func (s *Selector) GroupByRegion() map[string][]*config.Node {
	groups := make(map[string][]*config.Node)
	nodes := s.subManager.GetAllNodes()

	for _, node := range nodes {
		// TODO: implement IP geolocation
		region := detectRegion(node.Server)
		groups[region] = append(groups[region], node)
	}
	return groups
}

// UpdateStrategy updates selection strategy from config
func (s *Selector) UpdateStrategy() {
	s.strategySelector.SetStrategy(SelectionStrategy(s.config.SelectionStrategy))
}

// GetConnectionTracker returns connection tracker for stats
func (s *Selector) GetConnectionTracker() *ConnectionTracker {
	return s.strategySelector.GetConnectionTracker()
}

// GetStrategy returns current selection strategy
func (s *Selector) GetStrategy() string {
	return string(s.strategySelector.GetStrategy())
}

// SetStrategy changes selection strategy dynamically
func (s *Selector) SetStrategy(strategy string) {
	s.config.SelectionStrategy = strategy
	s.strategySelector.SetStrategy(SelectionStrategy(strategy))
}

func detectRegion(ip string) string {
	// TODO: implement geolocation detection
	return "default"
}
