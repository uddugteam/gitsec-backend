package service

import (
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"

	"gitsec-backend/config"
	"gitsec-backend/internal/models"
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

	InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error)
}

// GitService is a Git service implementation
type GitService struct {
	// baseGitPath is the base path for the Git
	// repositories on the file system.
	baseGitPath string
}

// NewGitService creates a new GitService instance with
// the given configuration.
func NewGitService(cfg *config.Git) *GitService {
	return &GitService{baseGitPath: cfg.Path}
}

// UploadPack handles Git "git-upload-pack" command
// and returns UploadPackResponse
func (g *GitService) UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error) {
	upr := packp.NewUploadPackRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	repo, err := models.NewRepo(repositoryName, g.baseGitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	sess, err := repo.NewUploadPackSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new upload pack session to git: %w", err)
	}

	defer sess.Close()

	res, err := sess.UploadPack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload pack to git: %w", err)
	}

	return res, nil
}

// ReceivePack handles Git "git-receive-pack" command
// and returns ReportStatus
func (g *GitService) ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error) {
	upr := packp.NewReferenceUpdateRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)

	}

	repo, err := models.NewRepo(repositoryName, g.baseGitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	sess, err := repo.NewReceivePackSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create new recieve pack session to git: %w", err)
	}

	defer sess.Close()

	res, err := sess.ReceivePack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to recieve pack to git: %w", err)
	}

	return res, nil
}

// InfoRef retrieves advertised refs for given repository
// and GitSessionType
func (g *GitService) InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error) {
	repo, err := models.NewRepo(repositoryName, g.baseGitPath)
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
