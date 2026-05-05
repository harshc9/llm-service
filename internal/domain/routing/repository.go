package routing

import (
	"context"

	"gorm.io/gorm"
)

type PolicyRepository interface {
	GetByFeature(ctx context.Context, feature string) (*RoutingPolicy, error)
	Create(ctx context.Context, policy *RoutingPolicy) error
}

type policyRepository struct {
	db *gorm.DB
}

func NewPolicyRepository(db *gorm.DB) PolicyRepository {
	return &policyRepository{db: db}
}

func (r *policyRepository) GetByFeature(ctx context.Context, feature string) (*RoutingPolicy, error) {
	var policy RoutingPolicy
	err := r.db.WithContext(ctx).Where("feature = ? AND enabled = ?", feature, true).First(&policy).Error
	if err != nil {
		return nil, err
	}
	return &policy, nil
}

func (r *policyRepository) Create(ctx context.Context, policy *RoutingPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}
