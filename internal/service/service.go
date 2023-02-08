package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/misnaged/annales/logger"

	"gitsec-backend/config"
	"gitsec-backend/internal/models"
	"gitsec-backend/pkg/fs-ipfs"
)

// IGitService defines the interface for Git Service
// that allows to interact with repositories
type IGitService interface {
	// UploadPack handles Git "git-upload-pack" command
	// and returns UploadPackResponse
	UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error)

	// ReceivePack handles Git "git-receive-pack" command
	// and returns ReportStatus
	ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error)

	// InfoRef retrieves advertised refs for given repository
	// and GitSessionType
	InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error)

	Close()
}

// GitService is a Git service implementation
type GitService struct {
	// baseGitPath is the base path for the Git
	// repositories on the file system.
	baseGitPath string

	fs billy.Filesystem

	stop chan struct{}
}

// NewGitService creates a new GitService instance with
// the given configuration.
func NewGitService(cfg *config.Scheme) (*GitService, error) {
	stop := make(chan struct{})

	fileSystem, err := fs.NewIPFSFilesystem(cfg.Ipfs.Address, stop)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs filesystem: %w", err)
	}

	//fileSystem = osfs.New(r.FullPath())
	//fileSystem := memfs.New()

	return &GitService{
		baseGitPath: cfg.Git.Path,
		fs:          fileSystem,
		stop:        stop,
	}, nil
}

// UploadPack handles Git "git-upload-pack" command
// and returns UploadPackResponse
func (g *GitService) UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error) {
	start := time.Now()

	upr := packp.NewUploadPackRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	repo, err := models.NewRepo(repositoryName, g.baseGitPath, g.fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	logger.Log().Infof("repo created in %s", time.Since(start))

	sess, err := repo.NewUploadPackSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new upload pack session to git: %w", err)
	}
	defer sess.Close()

	logger.Log().Infof("session created in %s", time.Since(start))

	res, err := sess.UploadPack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload pack to git: %w", err)
	}

	logger.Log().Infof("upload pack handled in %s", time.Since(start))

	return res, nil
}

// ReceivePack handles Git "git-receive-pack" command
// and returns ReportStatus
func (g *GitService) ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error) {
	start := time.Now()

	upr := packp.NewReferenceUpdateRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)

	}

	repo, err := models.NewRepo(repositoryName, g.baseGitPath, g.fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	logger.Log().Infof("repo created in %s", time.Since(start))

	sess, err := repo.NewReceivePackSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new recieve pack session to git: %w", err)
	}
	defer sess.Close()

	logger.Log().Infof("session created in %s", time.Since(start))

	res, err := sess.ReceivePack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to recieve pack to git: %w", err)
	}

	logger.Log().Infof("recieve pack handled in %s", time.Since(start))

	return res, nil
}

// InfoRef retrieves advertised refs for given repository
// and GitSessionType
func (g *GitService) InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error) {
	repo, err := models.NewRepo(repositoryName, g.baseGitPath, g.fs)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	sess, err := repo.NewSessionFromType(infoRefRequestType)
	if err != nil {
		return nil, fmt.Errorf("failed to create new git session: %w", err)
	}

	defer sess.Close()

	ar, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve context: %w", err)
	}

	ar.Prefix = [][]byte{
		[]byte(fmt.Sprintf("# service=%s", infoRefRequestType)),
		pktline.Flush,
	}

	if err := ar.Capabilities.Add("no-thin"); err != nil {
		return nil, fmt.Errorf("failed to add no-thin capability: %w", err)

	}

	return ar, nil
}

func (g *GitService) Close() {
	close(g.stop)
}
