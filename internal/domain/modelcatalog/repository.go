package modelcatalog

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository interface {
	Upsert(ctx context.Context, model *ModelCatalog) error
	ListEnabled(ctx context.Context) ([]ModelCatalog, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Upsert(ctx context.Context, model *ModelCatalog) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "model_name"}},
		UpdateAll: true,
	}).Create(model).Error
}

func (r *repository) ListEnabled(ctx context.Context) ([]ModelCatalog, error) {
	var models []ModelCatalog
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("priority asc").Find(&models).Error
	return models, err
}
