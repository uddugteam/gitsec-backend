package models

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/viper"
)

// Repo represents a local git repository.
type Repo struct {
	// Name is the name of the repository.
	Name string
	// BasePath is the base directory path where the repository is stored.

	Description string

	BasePath string

	ID int

	Owner common.Address

	Metadata string

	ForkFrom string

	// fileSystem is the filesystem where the repository is stored.
	fileSystem billy.Filesystem
	// server is the transport server used to handle git sessions.
	server transport.Transport
	// endpoint is the transport endpoint used to handle git sessions.
	endpoint *transport.Endpoint

	Repocore *git.Repository
}

// NewRepo creates a new Repo instance.
func NewRepo(name, description, basePath, forkFrom string, id int, owner common.Address, fs billy.Filesystem) (*Repo, error) {
	repo := &Repo{
		Name:        name,
		Description: description,
		BasePath:    basePath,
		ID:          id,
		Owner:       owner,
		ForkFrom:    forkFrom,
	}

	if err := repo.InitRepo(fs); err != nil {
		return nil, fmt.Errorf("failed to init Repocore: %w", err)
	}

	return repo, nil
}

// FullPath returns the full path to the repository directory.
func (r *Repo) FullPath() string {
	return fmt.Sprintf("%s/%s/", r.BasePath, r.Name)
}

// InitRepo initializes the repository
// by creating the filesystem, server, and endpoint.
func (r *Repo) InitRepo(fs billy.Filesystem) (err error) {
	r.fileSystem, err = fs.Chroot(r.Name)
	if err != nil {
		return fmt.Errorf("init chroot filesystem: %w", err)
	}

	if err := r.initFileSystem(); err != nil {
		return fmt.Errorf("failed to init Repocore filesystem: %w", err)
	}

	r.initServer()

	if err := r.initEndpoint(); err != nil {
		return fmt.Errorf("failed to init Repocore endpoint: %w", err)
	}

	return nil
}

// initFileSystem initializes the file system for the repository.
// If the file system does not already exist, a new repository is created.
func (r *Repo) initFileSystem() (err error) {
	if !r.isRepoFSExists() {
		if r.ForkFrom != "" {
			r.Repocore, err = git.Clone(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), nil, &git.CloneOptions{
				URL:               r.ForkFrom,
				RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			})
			if err != nil {
				return fmt.Errorf("failed to clone Repocore on fs: %w", err)
			}
		} else {
			r.Repocore, err = git.Init(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), nil)
			if err != nil {
				if errors.Is(err, git.ErrRepositoryAlreadyExists) {
					return nil
				}
				return fmt.Errorf("failed to create new Repocore on fs: %w", err)
			}
		}
	} else {
		r.Repocore, err = git.Open(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), nil)
		if err != nil {
			return fmt.Errorf("failed to open Repocore on fs: %w", err)
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
		return false
	}
	return true
}

func (r *Repo) Head() (p *plumbing.Reference, err error) {
	r.Repocore, err = git.Open(filesystem.NewStorage(r.fileSystem, cache.NewObjectLRU(500)), r.fileSystem)
	if err != nil {
		return nil, fmt.Errorf("failed to open Repocore on fs: %w", err)
	}

	// FIXME this way doesn't work ¯\_(ツ)_/¯
	/*ref, err := r.Repocore.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get repository head: %w", err)
	}*/

	ref, err := r.Repocore.Reference(plumbing.NewBranchReferenceName("main"), false)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository head: %w", err)
	}

	return ref, nil
}

func (r *Repo) Commit(hash plumbing.Hash) (*object.Commit, error) {
	commit, err := r.Repocore.CommitObject(hash)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Repocore commit %s: %w", hash, err)
	}

	return commit, nil
}

func (r *Repo) Tree(hash plumbing.Hash) (*object.Tree, error) {
	commit, err := r.Commit(hash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commit %s Tree: %w", hash, err)
	}

	return tree, nil
}

func (r *Repo) GenMeta() (*RepoMetadata, error) {
	meta := &RepoMetadata{
		Name:         r.Name,
		Description:  r.Description,
		ExternalUrl:  viper.GetString("baseurl") + r.Name,
		Tree:         []*RepoFile{},
		Commit:       "repository created",
		Timestamp:    time.Now().Unix(),
		CommitsCount: 0,
	}

	commit, err := r.LastCommit()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return meta, nil
		}
		return nil, fmt.Errorf("failed to retrieve last commit: %w", err)
	}

	commitsCount, err := r.CommitsCount()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commits count: %w", err)
	}

	meta.CommitsCount = commitsCount
	meta.Timestamp = commit.Author.When.Unix()
	meta.Commit = commit.Hash.String()

	return meta, nil
}

func (r *Repo) CommitsCount() (int, error) {
	cIter, err := r.Repocore.Log(&git.LogOptions{All: true})
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve repository logs: %w", err)
	}

	var cCount int
	if err := cIter.ForEach(func(c *object.Commit) error {
		cCount++
		return nil
	}); err != nil {
		return 0, fmt.Errorf("failed to iter over repository commits: %w", err)
	}

	return cCount, nil
}

func (r *Repo) LastCommit() (*object.Commit, error) {
	logs, err := r.Repocore.Log(&git.LogOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve repository logs: %w", err)
	}

	return logs.Next()
}

func (r *Repo) FileLastCommit(fileName string) (*object.Commit, error) {
	logs, err := r.Repocore.Log(&git.LogOptions{FileName: &fileName, All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve logs for file %s: %w", fileName, err)
	}

	return logs.Next()
}
