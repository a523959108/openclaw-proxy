package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/dns"
	"openclaw-mcp/internal/lighthouse"
	"openclaw-mcp/internal/selection"
	"openclaw-mcp/internal/stats"
	"openclaw-mcp/internal/subscription"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server is the API server
type Server struct {
	config     *config.Config
	subManager *subscription.Manager
	selector   *selection.Selector
	lighthouse *lighthouse.Lighthouse
	dnsResolver *dns.Resolver
	statsCollector *stats.StatsCollector
	router     *chi.Mux
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	sm *subscription.Manager,
	s *selection.Selector,
	lh *lighthouse.Lighthouse,
	dr *dns.Resolver,
	sc *stats.StatsCollector,
) *Server {
	srv := &Server{
		config:         cfg,
		subManager:     sm,
		selector:       s,
		lighthouse:     lh,
		dnsResolver:    dr,
		statsCollector: sc,
		router:         chi.NewRouter(),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

	// Basic auth if enabled
	if s.config.EnableAuth {
		s.router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				username, password, ok := r.BasicAuth()
				if !ok || username != s.config.Username || password != s.config.Password {
					w.Header().Set("WWW-Authenticate", `Basic realm="Openclaw MCP"`)
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
			})
		})
	}

	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		// Subscriptions
		r.Get("/subscriptions", s.GetSubscriptions)
		r.Post("/subscriptions", s.AddSubscription)
		r.Delete("/subscriptions/{id}", s.DeleteSubscription)
		r.Post("/subscriptions/{id}/update", s.UpdateSubscription)

		// Nodes
		r.Get("/nodes", s.GetNodes)
		r.Post("/nodes/test", s.TestAllNodes)
		r.Get("/nodes/best", s.GetBestNode)

		// Selection
		r.Get("/current", s.GetCurrent)
		r.Post("/select/best", s.SelectBest)

		// Config
		r.Get("/config", s.GetConfig)
		r.Put("/config", s.UpdateConfig)

		// Groups
		r.Get("/groups", s.GetGroups)
		r.Post("/groups", s.CreateGroup)
		r.Post("/groups/{name}/select", s.SelectGroupNode)

		// Statistics
		r.Get("/stats", s.GetStats)
		r.Get("/stats/nodes", s.GetNodeStats)
		r.Post("/stats/reset", s.ResetStats)
	})

	// Web UI
	s.router.Handle("/*", http.FileServer(http.Dir("./web")))

	// MCP endpoints for openclaw
	s.router.Route("/mcp", func(r chi.Router) {
		mcpServer := NewMCPServer(s.selector)
		r.Get("/current", mcpServer.GetCurrentNode)
		r.Get("/status", mcpServer.Status)
	})
})

// Start starts the server
func (s *Server) Start() error {
	if s.config.EnableHTTPS && s.config.CertFile != "" && s.config.KeyFile != "" {
		return http.ListenAndServeTLS(s.config.ListenAddr, s.config.CertFile, s.config.KeyFile, s.router)
	}
	return http.ListenAndServe(s.config.ListenAddr, s.router)
}

// API Handlers
func (s *Server) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.config.Subscriptions)
}

func (s *Server) AddSubscription(w http.ResponseWriter, r *http.Request) {
	var sub config.Subscription
	if err := json.NewDecoder(r.Body).Decode(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	sub.Enabled = true
	if err := s.subManager.AddSubscription(&sub); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}
	s.subManager.RemoveSubscription(id)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid subscription id", http.StatusBadRequest)
		return
	}
	if id < 0 || id >= len(s.config.Subscriptions) {
		http.Error(w, "subscription not found", http.StatusNotFound)
		return
	}
	sub := s.config.Subscriptions[id]
	if err := s.subManager.Update(sub); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.subManager.GetAllNodes()
	json.NewEncoder(w).Encode(nodes)
}

func (s *Server) TestAllNodes(w http.ResponseWriter, r *http.Request) {
	nodes := s.subManager.GetAllNodes()
	s.lighthouse.TestAll(nodes)
	json.NewEncoder(w).Encode(nodes)
}

func (s *Server) GetBestNode(w http.ResponseWriter, r *http.Request) {
	node := s.selector.GetBestNode()
	json.NewEncoder(w).Encode(node)
}

func (s *Server) GetCurrent(w http.ResponseWriter, r *http.Request) {
	node := s.selector.GetCurrent()
	json.NewEncoder(w).Encode(node)
}

func (s *Server) SelectBest(w http.ResponseWriter, r *http.Request) {
	node := s.selector.SelectBest()
	json.NewEncoder(w).Encode(node)
}

func (s *Server) GetConfig(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.config)
}

func (s *Server) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newConfig config.Config
	if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	*s.config = newConfig
	if err := config.SaveConfig("config.yaml", s.config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// GetGroups returns all groups
func (s *Server) GetGroups(w http.ResponseWriter, r *http.Request) {
	// Auto generate groups from subscription tags if no manual groups
	groups := s.selector.GetAllGroups()
	json.NewEncoder(w).Encode(groups)
}

// CreateGroup creates a new group
func (s *Server) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var group config.Group
	if err := json.NewDecoder(r.Body).Decode(&group); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.config.Groups = append(s.config.Groups, &group)
	if err := config.SaveConfig("config.yaml", s.config); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// SelectGroupNode selects best node for a group
func (s *Server) SelectGroupNode(w http.ResponseWriter, r *http.Request) {
	groupName := chi.URLParam(r, "name")
	node := s.selector.SelectBestForGroup(groupName)
	if node == nil {
		http.Error(w, "group not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(node)
}

// GetStats returns overall connection statistics
func (s *Server) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := s.statsCollector.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// GetNodeStats returns statistics per node
func (s *Server) GetNodeStats(w http.ResponseWriter, r *http.Request) {
	nodeStats := s.statsCollector.GetNodeStats()
	json.NewEncoder(w).Encode(nodeStats)
}

// ResetStats resets all statistics
func (s *Server) ResetStats(w http.ResponseWriter, r *http.Request) {
	s.statsCollector.ResetStats()
	w.WriteHeader(http.StatusOK)
}
