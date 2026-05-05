package client

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Create(ctx context.Context, c *Client) error
	GetByToken(ctx context.Context, token string) (*Client, error)
	List(ctx context.Context) ([]Client, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, c *Client) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) GetByToken(ctx context.Context, token string) (*Client, error) {
	var c Client
	err := r.db.WithContext(ctx).Where("api_token = ?", token).First(&c).Error
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) List(ctx context.Context) ([]Client, error) {
	var clients []Client
	err := r.db.WithContext(ctx).Find(&clients).Error
	return clients, err
}
