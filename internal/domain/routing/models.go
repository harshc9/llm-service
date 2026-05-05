package routing

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type RoutingPolicy struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Feature       string         `json:"feature"` // generate, live, embed, grounded
	Strategy      string         `json:"strategy"` // priority, round_robin, cost_aware
	ModelChainJSON datatypes.JSON `json:"model_chain"`
	Enabled       bool           `json:"enabled"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}
