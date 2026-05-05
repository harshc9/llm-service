package keypool

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	CreateProject(ctx context.Context, project *ApiProject) error
	CreateKey(ctx context.Context, key *ApiKey) error
	ListProjects(ctx context.Context) ([]ApiProject, error)
	GetAnyKey(ctx context.Context) (*ApiKey, error)
	GetKeyByID(ctx context.Context, id uuid.UUID) (*ApiKey, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) CreateProject(ctx context.Context, project *ApiProject) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *repository) CreateKey(ctx context.Context, key *ApiKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *repository) ListProjects(ctx context.Context) ([]ApiProject, error) {
	var projects []ApiProject
	err := r.db.WithContext(ctx).Preload("Keys").Find(&projects).Error
	return projects, err
}

func (r *repository) GetAnyKey(ctx context.Context) (*ApiKey, error) {
	var key ApiKey
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority asc").First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (r *repository) GetKeyByID(ctx context.Context, id uuid.UUID) (*ApiKey, error) {
	var key ApiKey
	err := r.db.WithContext(ctx).First(&key, id).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}
