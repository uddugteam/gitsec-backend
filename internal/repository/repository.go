package repository

import (
	"fmt"

	"gitsec-backend/internal/models"
)

type IRepository interface {
	CreateRepo(repo *models.Repo) error

	GetRepo(repo *models.Repo) error
}

type Repository struct {
	repositories map[string]*models.Repo
}

func NewRepository() IRepository {
	return &Repository{repositories: make(map[string]*models.Repo)}
}

func (r *Repository) CreateRepo(repo *models.Repo) error {
	if _, ok := r.repositories[repo.Name]; ok {
		return fmt.Errorf("repo already exist")
	}

	newRepo := &models.Repo{
		Name:        repo.Name,
		Description: repo.Description,
		BasePath:    repo.BasePath,
		ID:          repo.ID,
		Owner:       repo.Owner,
		Metadata:    repo.Metadata,
		Repocore:    repo.Repocore,
	}

	r.repositories[repo.Name] = newRepo
	return nil
}

func (r *Repository) GetRepo(repo *models.Repo) error {
	re, ok := r.repositories[repo.Name]
	if !ok {
		return fmt.Errorf("repo doesn't exist")
	}

	repo.Metadata = re.Metadata
	repo.ID = re.ID
	repo.Description = re.Description
	repo.Owner = re.Owner
	repo.BasePath = re.BasePath
	return nil
}
