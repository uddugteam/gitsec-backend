package models

import (
	"fmt"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

type Repo struct {
	Name       string
	BasePath   string
	fileSystem billy.Filesystem
	server     transport.Transport
	endpoint   *transport.Endpoint
}

func NewRepo(name, basePath string) (*Repo, error) {
	repo := &Repo{
		Name:     name,
		BasePath: basePath,
	}

	if err := repo.initRepo(); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	return repo, nil
}

func (r *Repo) FullPath() string {
	return fmt.Sprintf("%s/%s/", r.BasePath, r.Name)
}

func (r *Repo) initRepo() error {
	if err := r.initFileSystem(); err != nil {
		return fmt.Errorf("failed to init repo filesystem: %w", err)
	}

	r.initServer()

	if err := r.initEndpoint(); err != nil {
		return fmt.Errorf("failed to init repo endpoint: %w", err)
	}

	return nil
}

func (r *Repo) initFileSystem() error {
	r.fileSystem = osfs.New(r.FullPath())

	if !r.isRepoFSExists() {
		if _, err := git.Init(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), r.fileSystem); err != nil {
			return fmt.Errorf("failed to create new repo on fs: %w", err)
		}
	}

	return nil
}

func (r *Repo) initServer() {
	r.server = server.NewServer(server.NewFilesystemLoader(r.fileSystem))
}

func (r *Repo) initEndpoint() (err error) {
	r.endpoint, err = transport.NewEndpoint("/")
	if err != nil {
		return fmt.Errorf("failed to create new endpoint: %w", err)
	}
	return
}

func (r *Repo) NewUploadPackSession() (transport.UploadPackSession, error) {
	return r.server.NewUploadPackSession(r.endpoint, nil)
}

func (r *Repo) NewReceivePackSession() (transport.ReceivePackSession, error) {
	return r.server.NewReceivePackSession(r.endpoint, nil)
}

func (r *Repo) NewSessionFromType(sessionType GitSessionType) (transport.Session, error) {
	switch sessionType {
	case GitSessionReceivePack:
		return r.NewReceivePackSession()
	case GitSessionUploadPack:
		return r.NewUploadPackSession()
	default:
		return nil, fmt.Errorf("unsupported git session type %v", sessionType)
	}
}

func (r *Repo) isRepoFSExists() bool {
	if _, err := os.Stat(r.FullPath()); os.IsNotExist(err) {
		return false
	}
	return true
}
