package admin

import (
	"context"

	"github.com/harshc9/llm-service/internal/domain/usage"
	"gorm.io/gorm"
)

type StatsService struct {
	db *gorm.DB
}

func NewStatsService(db *gorm.DB) *StatsService {
	return &StatsService{db: db}
}

type UsageSummary struct {
	TotalRequests int     `json:"total_requests"`
	TotalTokens   int     `json:"total_tokens"`
	ErrorRate     float64 `json:"error_rate"`
}

func (s *StatsService) GetUsageSummary(ctx context.Context) (*UsageSummary, error) {
	var summary UsageSummary
	err := s.db.WithContext(ctx).Model(&usage.UsageEvent{}).
		Select("count(*) as total_requests, sum(input_tokens + output_tokens) as total_tokens").
		Scan(&summary).Error

	if err != nil {
		return nil, err
	}

	var totalErrors int64
	s.db.WithContext(ctx).Model(&usage.UsageEvent{}).Where("status_code != ?", 200).Count(&totalErrors)

	if summary.TotalRequests > 0 {
		summary.ErrorRate = float64(totalErrors) / float64(summary.TotalRequests)
	}

	return &summary, nil
}

type ModelUsage struct {
	ModelName string `json:"model_name"`
	Requests  int    `json:"requests"`
}

func (s *StatsService) GetModelUsage(ctx context.Context) ([]ModelUsage, error) {
	var results []ModelUsage
	err := s.db.WithContext(ctx).Model(&usage.UsageEvent{}).
		Select("model_name, count(*) as requests").
		Group("model_name").
		Order("requests desc").
		Scan(&results).Error
	return results, err
}
