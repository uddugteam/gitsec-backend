package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitsec-backend/config"
	"gitsec-backend/internal/server/handlers"
	"gitsec-backend/internal/service"
)

// HTTPServer represents a HTTP server that handles incoming requests
// and routes them to the appropriate handler functions.
type HTTPServer struct {
	// contains handler functions for handling different routes
	handlers *handlers.Handlers
	// underlying HTTP server instance
	*http.Server
}

// NewHTTPServer creates a new HTTPServer instance
// and binds it to the given port. It also initializes
// the routes and registers them to the server.
func NewHTTPServer(
	cfg *config.Scheme,
	srv service.IGitService,
) *HTTPServer {
	server := &HTTPServer{
		handlers: handlers.NewHandlers(cfg.Git.Path, srv),
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.HTTP.Port),
		},
	}

	server.registerRoutes()

	return server
}

// registerRoutes registers the routes to
// the HTTPServer instance.
func (s *HTTPServer) registerRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.HandleFunc("/{repoName}/info/refs", s.handlers.InfoRef())
	r.HandleFunc("/{repoName}/git-upload-pack", s.handlers.GitUploadPack())
	r.HandleFunc("/{repoName}/git-receive-pack", s.handlers.GitReceivePack())

	s.Handler = r
}
