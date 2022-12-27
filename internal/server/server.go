package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitsec-backend/config"
	"gitsec-backend/internal/server/handlers"
)

const gitDir = ".git/"

type HttpServer struct {
	handlers *handlers.Handlers
	*http.Server
}

// NewHttpServer create new Http Server
// instance and bind it with given port
func NewHttpServer(
	config *config.Scheme,
) *HttpServer {
	server := &HttpServer{
		handlers: handlers.NewHandlers(gitDir),
		Server: &http.Server{
			Addr: fmt.Sprintf(":%d", config.Http.Port),
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
