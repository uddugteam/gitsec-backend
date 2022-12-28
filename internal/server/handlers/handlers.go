package handlers

import "gitsec-backend/internal/service"

const (
	// repoNamePath is the path parameter key for the repository name
	repoNamePath = "repoName"
)

// Handlers represents a set of HTTP handlers for handling
// Git requests.
type Handlers struct {
	// dir is the base directory for the repositories
	dir string
	// srv is the service for interacting with the git repositories
	srv service.IGitService
}

// NewHandlers returns a new instance of Handlers
// with the given directory and service.
func NewHandlers(dir string, srv service.IGitService) *Handlers {
	return &Handlers{
		dir: dir,
		srv: srv,
	}
}
