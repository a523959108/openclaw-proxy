package api

import (
	"encoding/json"
	"net/http"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/selection"
)

// MCP is the Module Control Protocol interface for openclaw integration
type MCP interface {
	// GetProxy returns the current proxy configuration for openclaw
	GetProxy() *config.Node
	// Reload reloads all subscriptions
	Reload() error
}

// MCPServer implements the MCP endpoints for openclaw
type MCPServer struct {
	selector *selection.Selector
}

// NewMCPServer creates a new MCP server
func NewMCPServer(s *selection.Selector) *MCPServer {
	return &MCPServer{
		selector: s,
	}
}

// GetCurrentNode returns the current selected node - this is the main endpoint openclaw calls
func (m *MCPServer) GetCurrentNode(w http.ResponseWriter, r *http.Request) {
	node := m.selector.GetCurrent()
	if node == nil {
		node = m.selector.GetBestNode()
	}
	json.NewEncoder(w).Encode(node)
}

// Status returns the current status
func (m *MCPServer) Status(w http.ResponseWriter, r *http.Request) {
	type StatusResponse struct {
		CurrentNode *config.Node `json:"current_node"`
		Available   int          `json:"available_nodes"`
	}
	current := m.selector.GetCurrent()
	available := m.selector.CountAvailableNodes()
	json.NewEncoder(w).Encode(&StatusResponse{
		CurrentNode: current,
		Available:   available,
	})
}
