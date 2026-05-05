package modelcatalog

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ModelCatalog struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	ModelName      string         `gorm:"uniqueIndex" json:"model_name"`
	Capability     string         `json:"capability"` // text, live, embeddings, grounding
	Provider       string         `json:"provider"`
	IsPreview      bool           `json:"is_preview"`
	IsFreeEligible bool           `json:"is_free_eligible"`
	Priority       int            `json:"priority"`
	Enabled        bool           `json:"enabled"`
	MetaJSON       datatypes.JSON `json:"meta"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}
