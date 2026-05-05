package usage

import (
	"time"

	"github.com/google/uuid"
)

type UsageEvent struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	RequestID    uuid.UUID `gorm:"index" json:"request_id"`
	ClientID     uuid.UUID `gorm:"index" json:"client_id"`
	ProjectID    uuid.UUID `gorm:"index" json:"project_id"`
	ApiKeyID     uuid.UUID `gorm:"index" json:"api_key_id"`
	ModelName    string    `gorm:"index" json:"model_name"`
	Feature      string    `gorm:"index" json:"feature"`
	Endpoint     string    `json:"endpoint"`
	StatusCode   int       `json:"status_code"`
	ErrorCode    string    `json:"error_code"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	LatencyMs    int       `json:"latency_ms"`
	RetryCount   int       `json:"retry_count"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

type DailyUsageAggregate struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Day          time.Time `gorm:"index" json:"day"`
	ClientID     uuid.UUID `gorm:"index" json:"client_id"`
	ProjectID    uuid.UUID `gorm:"index" json:"project_id"`
	ApiKeyID     uuid.UUID `gorm:"index" json:"api_key_id"`
	ModelName    string    `gorm:"index" json:"model_name"`
	Feature      string    `gorm:"index" json:"feature"`
	Requests     int       `json:"requests"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Errors       int       `json:"errors"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
