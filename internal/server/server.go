package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitsec-backend/config"
	"gitsec-backend/internal/server/handlers"
	"gitsec-backend/internal/service"
)

const gitDir = ".git/"

type HttpServer struct {
	handlers *handlers.Handlers
	*http.Server
}

// NewHttpServer create new Http Server
// instance and bind it with given port
func NewHttpServer(
	cfg *config.Http,
	srv service.IGitService,
) *HttpServer {
	server := &HttpServer{
		handlers: handlers.NewHandlers(gitDir, srv),
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", cfg.Port),
		},
	}

	server.registerRoutes()

	return server
}

// registerRoutes register HttpServer instance routes
func (s *HttpServer) registerRoutes() {
	r := chi.NewRouter()

	r.HandleFunc("/info/refs", s.handlers.InfoRef())
	r.HandleFunc("/git-upload-pack", s.handlers.GitUploadPack())
	r.HandleFunc("/git-receive-pack", s.handlers.GitReceivePack())

	s.Handler = r
}
