package repository

//import (
//	"fmt"
//	"sync"
//
//	"gitsec-backend/internal/models"
//)
//
//type IRepository interface {
//	CreateRepo(model *models.Repo) error
//
//	RepoByName(model *models.Repo) error
//
//	IsRepoExist(model *models.Repo) bool
//}
//
//type InMemoryRepository struct {
//	reposMap map[string]*models.Repo
//
//	mu sync.Mutex
//}
//
//func NewRepository() IRepository {
//	return &InMemoryRepository{reposMap: make(map[string]*models.Repo)}
//}
//
//func (r *InMemoryRepository) CreateRepo(model *models.Repo) error {
//	if r.IsRepoExist(model) {
//		return fmt.Errorf("repo with name %v already exist", model.Name)
//	}
//
//	defer r.mu.Unlock()
//	r.mu.Lock()
//
//	r.reposMap[model.Name] = model
//	return nil
//}
//
//func (r *InMemoryRepository) RepoByName(model *models.Repo) error {
//	defer r.mu.Unlock()
//	r.mu.Lock()
//
//	repo, ok := r.reposMap[model.Name]
//	if !ok {
//		return fmt.Errorf("repo with name %v doesn't exist", model.Name)
//	}
//
//	model = repo
//	return nil
//}
//
//func (r *InMemoryRepository) IsRepoExist(model *models.Repo) bool {
//	defer r.mu.Unlock()
//	r.mu.Lock()
//
//	if _, ok := r.reposMap[model.Name]; ok {
//		return true
//	}
//	return false
//}
