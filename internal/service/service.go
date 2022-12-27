package service

import (
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/server"

	"gitsec-backend/config"
)

type IGitService interface {
	UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error)

	ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error)

	InfoRef(ctx context.Context, repositoryName, infoRefRequestType string) (*packp.AdvRefs, error)
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

	ep, err := transport.NewEndpoint("/")
	if err != nil {
		return nil, fmt.Errorf("failed to create new endpoint: %w", err)

	}

	svr := server.NewServer(server.NewFilesystemLoader(osfs.New(g.baseGitPath)))

	sess, err := svr.NewUploadPackSession(ep, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new upload pack session to git: %w", err)
	}

	res, err := sess.UploadPack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to upload pack to git: %w", err)
	}

	return res, nil
}

func (g *GitService) ReceivePack(ctx context.Context, req io.Reader, repositoryName string) (*packp.ReportStatus, error) {
	upr := packp.NewReferenceUpdateRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)

	}

	ep, err := transport.NewEndpoint("/")
	if err != nil {
		return nil, fmt.Errorf("failed to create new endpoint: %w", err)
	}

	svr := server.NewServer(server.NewFilesystemLoader(osfs.New(g.baseGitPath)))

	sess, err := svr.NewReceivePackSession(ep, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create new recieve pack session to git: %w", err)
	}

	res, err := sess.ReceivePack(ctx, upr)
	if err != nil {
		return nil, fmt.Errorf("failed to recieve pack to git: %w", err)
	}

	return res, nil
}

const (
	gitUploadPackType  = "git-upload-pack"
	gitReceivePackType = "git-receive-pack"
)

func (g *GitService) InfoRef(ctx context.Context, repositoryName, infoRefRequestType string) (*packp.AdvRefs, error) {
	ep, err := transport.NewEndpoint("/")
	if err != nil {
		return nil, fmt.Errorf("failed to create new endpoint: %w", err)
	}

	svr := server.NewServer(server.NewFilesystemLoader(osfs.New(g.baseGitPath)))

	var sess transport.Session

	switch infoRefRequestType {
	case gitUploadPackType:
		sess, err = svr.NewUploadPackSession(ep, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new upload pack session to git: %w", err)
		}
	case gitReceivePackType:
		sess, err = svr.NewReceivePackSession(ep, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create new recieve pack session to git: %w", err)
		}
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
