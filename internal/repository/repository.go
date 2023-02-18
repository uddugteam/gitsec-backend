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

	r.repositories[repo.Name] = repo
	return nil
}

func (r *Repository) GetRepo(repo *models.Repo) error {
	re, ok := r.repositories[repo.Name]
	if !ok {
		return fmt.Errorf("repo doesn't exist")
	}

	repo = re
	return nil
}
