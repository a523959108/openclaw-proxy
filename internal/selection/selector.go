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
	subManager *subscription.Manager
	lighthouse *lighthouse.Lighthouse
	cancelCtx  context.CancelFunc
	ctx        context.Context
	wg         sync.WaitGroup
	current    *config.Node
	mu         sync.RWMutex
}

// New creates a new selector
func New(lh *lighthouse.Lighthouse, sm *subscription.Manager) *Selector {
	ctx, cancel := context.WithCancel(context.Background())
	return &Selector{
		subManager: sm,
		lighthouse: lh,
		ctx:        ctx,
		cancelCtx:  cancel,
	}
}

// GetBestNode selects the best available node based on latency
func (s *Selector) GetBestNode() *config.Node {
	nodes := s.subManager.GetAllNodes()
	if len(nodes) == 0 {
		return nil
	}

	availableNodes := make([]*config.Node, 0)
	for _, node := range nodes {
		if node.Available {
			availableNodes = append(availableNodes, node)
		}
	}

	if len(availableNodes) == 0 {
		// If no tested nodes available, return first node
		return nodes[0]
	}

	// Sort by latency
	sort.Slice(availableNodes, func(i, j int) bool {
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
		ticker := time.NewTicker(30 * time.Minute)
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

// StopAutoSelection stops the automatic selection
func (s *Selector) StopAutoSelection() {
	s.cancelCtx()
	s.wg.Wait()
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
