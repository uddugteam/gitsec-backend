package models

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// Repo represents a local git repository.
type Repo struct {
	// Name is the name of the repository.
	Name string
	// BasePath is the base directory path where the repository is stored.
	BasePath string
	// fileSystem is the filesystem where the repository is stored.
	fileSystem billy.Filesystem
	// server is the transport server used to handle git sessions.
	server transport.Transport
	// endpoint is the transport endpoint used to handle git sessions.
	endpoint *transport.Endpoint
}

// NewRepo creates a new Repo instance.
func NewRepo(name, basePath string, fs billy.Filesystem) (*Repo, error) {
	repo := &Repo{
		Name:       name,
		BasePath:   basePath,
		fileSystem: fs,
	}

	if err := repo.initRepo(); err != nil {
		return nil, fmt.Errorf("failed to init repo: %w", err)
	}

	return repo, nil
}

// FullPath returns the full path to the repository directory.
func (r *Repo) FullPath() string {
	return fmt.Sprintf("%s/%s/", r.BasePath, r.Name)
}

// initRepo initializes the repository
// by creating the filesystem, server, and endpoint.
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

// initFileSystem initializes the file system for the repository.
// If the file system does not already exist, a new repository is created.
func (r *Repo) initFileSystem() (err error) {
	/*r.fileSystem, err = fs.NewIPFSFilesystem("http://127.0.0.1:5001")
	if err != nil {
		return fmt.Errorf("failed to create ipfs filesystem: %w", err)
	}*/
	//r.fileSystem = osfs.New(r.FullPath())
	//r.fileSystem = memfs.New()

	if !r.isRepoFSExists() {
		if _, err := git.Init(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), r.fileSystem); err != nil {
			if errors.Is(err, git.ErrRepositoryAlreadyExists) {
				return nil
			}
			return fmt.Errorf("failed to create new repo on fs: %w", err)
		}
	}

	return nil
}

// initServer initializes the server for the repository.
func (r *Repo) initServer() {
	r.server = server.NewServer(server.NewFilesystemLoader(r.fileSystem))
}

// initEndpoint initializes the endpoint for the repository.
func (r *Repo) initEndpoint() (err error) {
	r.endpoint, err = transport.NewEndpoint("/")
	if err != nil {
		return fmt.Errorf("failed to create new endpoint: %w", err)
	}
	return
}

// NewUploadPackSession creates a new UploadPackSession for the repository.
// The UploadPackSession can be used to perform a git fetch operation.
func (r *Repo) NewUploadPackSession() (transport.UploadPackSession, error) {
	return r.server.NewUploadPackSession(r.endpoint, nil)
}

// NewReceivePackSession creates a new ReceivePackSession for the repository.
// The ReceivePackSession can be used to perform a git push operation.
func (r *Repo) NewReceivePackSession() (transport.ReceivePackSession, error) {
	return r.server.NewReceivePackSession(r.endpoint, nil)
}

// NewSessionFromType creates a new session of the specified type for the repository.
// Supported session types are GitSessionUploadPack and GitSessionReceivePack.
// Returns an error if the session type is not supported.
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

// isRepoFSExists checks if the filesystem for the repository exists.
// Returns true if the filesystem exists, false otherwise.
func (r *Repo) isRepoFSExists() bool {
	if _, err := r.fileSystem.Stat(r.FullPath()); os.IsNotExist(err) {
		//if _, err := r.fileSystem.Stat(r.Name); os.IsNotExist(err) {
		return false
	}
	return true
}
