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

type HttpServer struct {
	handlers *handlers.Handlers
	*http.Server
}

// NewHttpServer create new Http Server
// instance and bind it with given port
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

// registerRoutes register HttpServer instance routes
func (s *HttpServer) registerRoutes() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.HandleFunc("/{repoName}/info/refs", s.handlers.InfoRef())
	r.HandleFunc("/{repoName}/git-upload-pack", s.handlers.GitUploadPack())
	r.HandleFunc("/{repoName}/git-receive-pack", s.handlers.GitReceivePack())

	s.Handler = r
}
