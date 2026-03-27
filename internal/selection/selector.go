package selection

import (
	"context"
	"sort"
	"sync"
	"time"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/lighthouse"
	"openclaw-mcp/internal/subscription"
)

// Selector handles automatic node selection
type Selector struct {
	config     *config.Config
	subManager *subscription.Manager
	lighthouse *lighthouse.Lighthouse
	cancelCtx  context.CancelFunc
	ctx        context.Context
	wg         sync.WaitGroup
	current    *config.Node
	mu         sync.RWMutex
}

// New creates a new selector
func New(cfg *config.Config, lh *lighthouse.Lighthouse, sm *subscription.Manager) *Selector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Selector{
		config:     cfg,
		subManager: sm,
		lighthouse: lh,
		ctx:        ctx,
		cancelCtx:  cancel,
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
	for _, node := range nodes {
		if node.Available {
			if groupName == "" || node.Group == groupName {
				availableNodes = append(availableNodes, node)
			}
		}
	}

	if len(availableNodes) == 0 {
		// If no tested nodes available, filter untested by group
		for _, node := range nodes {
			if groupName == "" || node.Group == groupName {
				return node
			}
		}
		return nodes[0] // fallback to first node if no match
	}

	// Sort by average latency for more stable selection
	sort.Slice(availableNodes, func(i, j int) bool {
		if availableNodes[i].AverageLatency > 0 && availableNodes[j].AverageLatency > 0 {
			return availableNodes[i].AverageLatency < availableNodes[j].AverageLatency
		}
		return availableNodes[i].Latency < availableNodes[j].Latency
	})

	return availableNodes[0]
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

func detectRegion(ip string) string {
	// TODO: implement geolocation detection
	return "default"
}
