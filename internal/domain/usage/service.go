package usage

import (
	"context"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) RecordEvent(ctx context.Context, event *UsageEvent) error {
	return s.db.WithContext(ctx).Create(event).Error
}
