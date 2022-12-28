package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/andskur/go-git/v5/plumbing/format/pktline"
	"github.com/andskur/go-git/v5/plumbing/protocol/packp"

	"gitsec-backend/config"
	"gitsec-backend/internal/models"
)

type IGitService interface {
	UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error)

	ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error)

	InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error)
}

type GitService struct {
	baseGitPath string
}

func NewGitService(cfg *config.Git) *GitService {
	return &GitService{baseGitPath: cfg.Path}
}

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

	res, err := sess.UploadPack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload pack to git: %w", err)
	}

	return res, nil

	/*ep, err := transport.NewEndpoint("/")
	if err != nil {
		return nil, fmt.Errorf("failed to create new endpoint: %w", err)

	}

	svr := server.NewServer(server.NewFilesystemLoader(osfs.New(g.baseGitPath + repositoryName + "/")))

	sess, err := svr.NewUploadPackSession(ep, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new upload pack session to git: %w", err)
	}

	res, err := sess.UploadPack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload pack to git: %w", err)
	}

	return res, nil*/
}

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

	res, err := sess.ReceivePack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to recieve pack to git: %w", err)
	}

	return res, nil
}

func (g *GitService) InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error) {
	repo, err := models.NewRepo(repositoryName, g.baseGitPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create new repo: %w", err)
	}

	sess, err := repo.NewSessionFromType(infoRefRequestType)
	if err != nil {
		return nil, fmt.Errorf("failed to create new git session: %w", err)
	}

	ar, err := sess.AdvertisedReferencesContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve context: %w", err)
	}

	ar.Prefix = [][]byte{
		[]byte(fmt.Sprintf("# service=%s", infoRefRequestType)),
		pktline.Flush,
	}

	return ar, nil
}

func (g *GitService) isRepoExist(repositoryName string) bool {
	if _, err := os.Stat(g.baseGitPath + repositoryName + "/"); os.IsNotExist(err) {
		return false
	}
	return true
}
