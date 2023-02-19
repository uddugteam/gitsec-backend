package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp"
	"github.com/misnaged/annales/logger"

	"gitsec-backend/config"
	"gitsec-backend/internal/models"
	"gitsec-backend/internal/repository"
	"gitsec-backend/pkg/contract"
	"gitsec-backend/pkg/pinner"
	"gitsec-backend/pkg/signer"
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

	StartListener()

	Close()
}

// GitService is a Git service implementation
type GitService struct {
	// baseGitPath is the base path for the Git
	// repositories on the file system.
	baseGitPath string

	fs billy.Filesystem

	pinner pinner.IPinner

	blockchain *ethclient.Client

	repository repository.IRepository

	contractAddress common.Address
	contract        *contract.Contract

	signer *signer.Signer

	chainId *big.Int

	stop chan struct{}
}

// NewGitService creates a new GitService instance with
// the given configuration.
func NewGitService(cfg *config.Scheme, blockchain *ethclient.Client) (*GitService, error) {
	stop := make(chan struct{})

	/*fileSystem, err := fs.NewIPFSFilesystem(cfg.Ipfs.Address, stop)
	if err != nil {
		return nil, fmt.Errorf("failed to create ipfs filesystem: %w", err)
	}*/

	fileSystem := osfs.New(cfg.Git.Path)
	//fileSystem := memfs.New()

	contractAddress := common.HexToAddress(cfg.Blockchain.Contract)

	gitSecContract, err := contract.NewContract(contractAddress, blockchain)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gitsec contract at %s: %w", contractAddress.Hex(), err)
	}

	chainId, err := blockchain.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Chain ID: %w", err)
	}

	sig, err := signer.NewSigner(cfg.Signer, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	var pinnerService pinner.IPinner

	switch cfg.Pinner {
	case "pinata":
		pinnerService = pinner.NewPinataPinner(cfg.Pinata.Jwt)
	case "ipfs":
		pinnerService = pinner.NewIpfsPinner(cfg.Ipfs.Address)
	default:
		return nil, fmt.Errorf("unsupported pinner %s", cfg.Pinner)
	}

	return &GitService{
		baseGitPath:     cfg.Git.Path,
		fs:              fileSystem,
		pinner:          pinnerService,
		blockchain:      blockchain,
		contract:        gitSecContract,
		repository:      repository.NewRepository(),
		contractAddress: contractAddress,
		signer:          sig,
		stop:            stop,
		chainId:         chainId,
	}, nil
}

func (g *GitService) StartListener() {
	if err := g.ListenRepositoryCreation(); err != nil {
		logger.Log().Error(err)
	}
}

func (g *GitService) ListenRepositoryCreation() error {
	repos := make(chan *contract.ContractRepositoryCreated)
	opts := &bind.WatchOpts{Context: context.Background()}

Subscribe:
	repositoryCreationSubscriptions, err := g.contract.WatchRepositoryCreated(opts, repos)
	if err != nil {
		return fmt.Errorf("failed subscribe to watch transfers event: %w", err)
	}
	defer repositoryCreationSubscriptions.Unsubscribe()

	logger.Log().Infof("listen contract repository creation events on %s", g.contractAddress.Hex())

	for {
		select {
		case <-g.stop:
			logger.Log().Warning("stop listen contract transfers events")
			close(repos)
			return nil
		case err := <-repositoryCreationSubscriptions.Err():
			logger.Log().Error(fmt.Errorf("repository creation subscription error: %w", err))
			goto Subscribe
		case r := <-repos:
			logger.Log().Infof("catch repository creation event: repository %s with ID %d created with owner %s", r.RepName, r.RepId, r.Owner.Hex())

			if err := g.CreateRepo(r.RepName, r.Description, int(r.RepId.Int64()), r.Owner); err != nil {
				logger.Log().Error(fmt.Errorf("error to create repository: %w", err))
			}
		}
	}
}

func (g *GitService) CreateRepo(name, description string, id int, owner common.Address) error {
	repo, err := models.NewRepo(name, description, g.baseGitPath, id, owner, g.fs)
	if err != nil {
		return fmt.Errorf("failed to create new repo: %w", err)
	}

	meta, err := repo.GenMeta()
	if err != nil {
		return fmt.Errorf("failed to generate repository meta: %w", err)
	}

	metaJson, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal repository metadata: %w", err)
	}

	hash, err := g.pinner.Pin(name+"-meta.json", bytes.NewReader(metaJson))
	if err != nil {
		return fmt.Errorf("pin repository metadata to ipfs: %w", err)
	}

	logger.Log().Infof("repository %s metadata %s pinned to IPFS", name, hash)

	repo.Metadata = hash

	if err := g.repository.CreateRepo(repo); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	logger.Log().Infof("repository %s ID %d created", repo.Name, repo.ID)

	sign, err := g.signer.Sign(g.chainId)
	if err != nil {
		return fmt.Errorf("prepare tx signing: %w", err)
	}

	tx, err := g.contract.UpdateIPFS(sign, big.NewInt(int64(repo.ID)), hash)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	logger.Log().Infof("transaction %s to update repository %s ID %d metadata %s send to blockchan", tx.Hash().Hex(), name, repo.ID, hash)

	rewrewr := &models.Repo{Name: name}

	if err := g.repository.GetRepo(rewrewr); err != nil {
		return err
	}

	return nil
}

// UploadPack handles Git "git-upload-pack" command
// and returns UploadPackResponse
func (g *GitService) UploadPack(ctx context.Context, req io.Reader, repositoryName string) (*packp.UploadPackResponse, error) {
	start := time.Now()

	upr := packp.NewUploadPackRequest()

	if err := upr.Decode(req); err != nil {
		return nil, fmt.Errorf("failed to decode request: %w", err)
	}

	repo := &models.Repo{Name: repositoryName}

	if err := g.repository.GetRepo(repo); err != nil {
		return nil, fmt.Errorf("failed to get repo %s: %w", repositoryName, err)
	}

	if err := repo.InitRepo(g.fs); err != nil {
		return nil, fmt.Errorf("failed to init repo %s: %w", repositoryName, err)
	}

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

	repo := &models.Repo{Name: repositoryName}

	if err := g.repository.GetRepo(repo); err != nil {
		return nil, fmt.Errorf("failed to get repo %s: %w", repositoryName, err)
	}

	if err := repo.InitRepo(g.fs); err != nil {
		return nil, fmt.Errorf("failed to init repo %s: %w", repositoryName, err)
	}

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

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get repo head: %w", err)
	}

	tree, err := repo.Tree(head.Hash())
	if err != nil {
		return nil, fmt.Errorf("failed to get repo tree: %w", err)
	}

	meta, err := repo.GenMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to generate repository meta: %w", err)
	}

	if err := meta.FillContent(tree); err != nil {
		return nil, fmt.Errorf("failed to fill metadata content: %w", err)
	}

	if err := meta.FillCommit(repo); err != nil {
		return nil, fmt.Errorf("failed to fill metadata tree commits: %w", err)
	}

	if err := g.StoreMetaTree(meta, repo); err != nil {
		return nil, fmt.Errorf("failed to store metadata content: %w", err)
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal repository metadata: %w", err)
	}

	hash, err := g.pinner.Pin(fmt.Sprintf("%s-%d-meta.json", repositoryName, time.Now().Unix()), bytes.NewReader(metaBytes))
	if err != nil {
		return nil, fmt.Errorf("pin repository metadata to ipfs: %w", err)
	}

	logger.Log().Infof("repository %s metadata %s pinned to IPFS", repositoryName, hash)

	repo.Metadata = hash

	sign, err := g.signer.Sign(g.chainId)
	if err != nil {
		return nil, fmt.Errorf("prepare tx signing: %w", err)
	}

	tx, err := g.contract.UpdateIPFS(sign, big.NewInt(int64(repo.ID)), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	logger.Log().Infof("transaction %s to update repository %s ID %d metadata %s send to blockchan", tx.Hash().Hex(), repositoryName, repo.ID, hash)

	return res, nil
}

func (g *GitService) StoreMetaTree(meta *models.RepoMetadata, repo *models.Repo) error {
	for _, f := range meta.Tree {
		hash, err := g.pinner.Pin(fmt.Sprintf("%s-%d.json", meta.Name, time.Now().Unix()), bytes.NewReader([]byte(f.Content)))
		if err != nil {
			return fmt.Errorf("pin file %s to ipfs: %w", f.Name, err)
		}

		logger.Log().Infof("file %s %s pinned to IPFS", f.Name, hash)

		fileMeta := &models.RepoFile{
			Name:      f.Name,
			Author:    f.Author,
			Commit:    f.Commit,
			Hash:      hash,
			Timestamp: f.Timestamp,
		}

		fileJson, err := json.Marshal(fileMeta)
		if err != nil {
			return fmt.Errorf("failed to marshal file meta: %w", err)
		}

		metaHash, err := g.pinner.Pin(fmt.Sprintf("%s-%d-meta.json", meta.Name, time.Now().Unix()), bytes.NewReader(fileJson))
		if err != nil {
			return fmt.Errorf("pin file %s metadata to ipfs: %w", f.Name, err)
		}

		logger.Log().Infof("file %s metadata %s pinned to IPFS", f.Name, metaHash)

		f.Hash = metaHash
	}

	return nil
}

// InfoRef retrieves advertised refs for given repository
// and GitSessionType
func (g *GitService) InfoRef(ctx context.Context, repositoryName string, infoRefRequestType models.GitSessionType) (*packp.AdvRefs, error) {
	logger.Log().Infof("handling InfoRef request for repo %s", repositoryName)

	repo := &models.Repo{Name: repositoryName}

	if err := g.repository.GetRepo(repo); err != nil {
		return nil, fmt.Errorf("failed to get repo %s: %w", repositoryName, err)
	}

	if err := repo.InitRepo(g.fs); err != nil {
		return nil, fmt.Errorf("failed to init repo %s: %w", repositoryName, err)
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
