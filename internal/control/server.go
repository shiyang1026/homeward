package control

import (
	"context"
	"net/http"
)

// Server is the control plane HTTP server.
// It owns the node registry and exposes the registration API.
type Server struct {
	httpServer *http.Server
	handler    *Handler
}

// NewServer creates a control plane server that listens on the given address.
func NewServer(addr string) *Server {
	store := NewNodeStore()
	handler := NewHandler(store)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/nodes", handler.RegisterNode)
	mux.HandleFunc("GET /api/v1/nodes", handler.ListNodes)

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		handler: handler,
	}
}

// Start begins serving and blocks until the server stops.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server, waiting for in-flight requests to finish.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
