package keypool

import (
	"time"

	"github.com/google/uuid"
)

type ApiProject struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"` // "google"
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Keys      []ApiKey  `gorm:"foreignKey:ProjectID" json:"keys,omitempty"`
}

type ApiKey struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ProjectID        uuid.UUID  `gorm:"type:uuid;index" json:"project_id"`
	Alias            string     `json:"alias"`
	SecretCiphertext string     `json:"-"` // Never expose in JSON
	Priority         int        `json:"priority"`
	Enabled          bool       `json:"enabled"`
	HealthState      string     `json:"health_state"` // healthy, degraded, cooldown, disabled
	LastUsedAt       *time.Time `json:"last_used_at"`
	FailCount        int        `json:"fail_count"`
	CooldownUntil    *time.Time `json:"cooldown_until"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
