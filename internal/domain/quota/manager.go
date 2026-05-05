package quota

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/harshc9/llm-service/internal/infra/config"
	"github.com/redis/go-redis/v9"
)

type Manager struct {
	rdb *redis.Client
	cfg *config.Config
}

func NewManager(rdb *redis.Client, cfg *config.Config) *Manager {
	return &Manager{
		rdb: rdb,
		cfg: cfg,
	}
}

func (m *Manager) CheckQuota(ctx context.Context, projectID uuid.UUID) (bool, error) {
	now := time.Now()
	minuteKey := fmt.Sprintf("quota:project:%s:min:%d", projectID, now.Unix()/60)
	dayKey := fmt.Sprintf("quota:project:%s:day:%d", projectID, now.YearDay())

	// Use pipeline for efficiency
	pipe := m.rdb.Pipeline()
	minCount := pipe.Get(ctx, minuteKey)
	dayCount := pipe.Get(ctx, dayKey)
	_, _ = pipe.Exec(ctx)

	minVal, _ := minCount.Int()
	dayVal, _ := dayCount.Int()

	if minVal >= m.cfg.MaxRPM || dayVal >= m.cfg.MaxRPD {
		return false, nil
	}

	return true, nil
}

func (m *Manager) IncrementUsage(ctx context.Context, projectID uuid.UUID, tokens int) error {
	now := time.Now()
	minuteKey := fmt.Sprintf("quota:project:%s:min:%d", projectID, now.Unix()/60)
	dayKey := fmt.Sprintf("quota:project:%s:day:%d", projectID, now.YearDay())

	pipe := m.rdb.Pipeline()
	pipe.Incr(ctx, minuteKey)
	pipe.Expire(ctx, minuteKey, 2*time.Minute)
	pipe.Incr(ctx, dayKey)
	pipe.Expire(ctx, dayKey, 25*time.Hour)
	_, err := pipe.Exec(ctx)
	
	return err
}

func (m *Manager) MarkDegraded(ctx context.Context, projectID uuid.UUID, duration time.Duration) error {
	key := fmt.Sprintf("health:project:%s:degraded", projectID)
	return m.rdb.Set(ctx, key, "true", duration).Err()
}

func (m *Manager) IsDegraded(ctx context.Context, projectID uuid.UUID) (bool, error) {
	key := fmt.Sprintf("health:project:%s:degraded", projectID)
	val, err := m.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	return val == "true", err
}
