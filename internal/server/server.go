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

// HttpServer represents a HTTP server that handles incoming requests
// and routes them to the appropriate handler functions.
type HttpServer struct {
	// contains handler functions for handling different routes
	handlers *handlers.Handlers
	// underlying HTTP server instance
	*http.Server
}

// NewHttpServer creates a new HttpServer instance
// and binds it to the given port. It also initializes
// the routes and registers them to the server.
func NewHttpServer(
	cfg *config.Scheme,
	srv service.IGitService,
) *HttpServer {
	server := &HttpServer{
		handlers: handlers.NewHandlers(cfg.Git.Path, srv),
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.Http.Port),
		},
	}

	server.registerRoutes()

	return server
}

// registerRoutes registers the routes to
// the HttpServer instance.
func (s *HttpServer) registerRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.HandleFunc("/{repoName}/info/refs", s.handlers.InfoRef())
	r.HandleFunc("/{repoName}/git-upload-pack", s.handlers.GitUploadPack())
	r.HandleFunc("/{repoName}/git-receive-pack", s.handlers.GitReceivePack())

	s.Handler = r
}
