package api

import (
	"encoding/json"
	"net/http"

	"openclaw-mcp/internal/config"
	"openclaw-mcp/internal/lighthouse"
	"openclaw-mcp/internal/selection"
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
	router     *chi.Mux
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	sm *subscription.Manager,
	s *selection.Selector,
	lh *lighthouse.Lighthouse,
) *Server {
	srv := &Server{
		config:     cfg,
		subManager: sm,
		selector:   s,
		lighthouse: lh,
		router:     chi.NewRouter(),
	}

	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)

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
	})

	// Web UI
	s.router.Handle("/*", http.FileServer(http.Dir("./web")))
}

// Start starts the server
func (s *Server) Start() error {
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
	// TODO: implementation
	w.WriteHeader(http.StatusOK)
}

func (s *Server) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	// TODO: implementation
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
	// TODO: implementation
	w.WriteHeader(http.StatusOK)
}
