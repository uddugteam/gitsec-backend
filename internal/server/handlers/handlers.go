package handlers

import "gitsec-backend/internal/service"

const (
	repoNamePath = "repoName"
)

type Handlers struct {
	dir string
	srv service.IGitService
}

func NewHandlers(dir string, srv service.IGitService) *Handlers {
	return &Handlers{
		dir: dir,
		srv: srv,
	}
}
