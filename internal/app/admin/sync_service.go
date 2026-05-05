package admin

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/harshc9/llm-service/internal/domain/keypool"
	"github.com/harshc9/llm-service/internal/domain/modelcatalog"
	"github.com/harshc9/llm-service/internal/infra/crypto"
	"github.com/harshc9/llm-service/internal/infra/gemini"
	"gorm.io/datatypes"
)

type SyncService struct {
	geminiClient *gemini.Client
	catalogRepo  modelcatalog.Repository
	keyRepo      keypool.Repository
	masterKey    []byte
}

func NewSyncService(
	geminiClient *gemini.Client,
	catalogRepo modelcatalog.Repository,
	keyRepo keypool.Repository,
	masterKey string,
) *SyncService {
	return &SyncService{
		geminiClient: geminiClient,
		catalogRepo:  catalogRepo,
		keyRepo:      keyRepo,
		masterKey:    []byte(masterKey),
	}
}

func (s *SyncService) SyncModels(ctx context.Context) error {
	// 1. Get an API key for syncing
	keyEntity, err := s.keyRepo.GetAnyKey(ctx)
	if err != nil {
		return err
	}

	apiKey, err := crypto.Decrypt(keyEntity.SecretCiphertext, s.masterKey)
	if err != nil {
		return err
	}

	// 2. Fetch models from Gemini
	models, err := s.geminiClient.ListModels(apiKey)
	if err != nil {
		return err
	}

	// 3. Upsert into catalog
	for _, m := range models {
		metaJSON, _ := json.Marshal(m)

		// Map capabilities based on supported methods or name
		capability := "text"
		// Simple heuristic, can be improved
		for _, method := range m.SupportedGenerationMethods {
			if method == "embedContent" {
				capability = "embeddings"
			}
		}

		catalogModel := &modelcatalog.ModelCatalog{
			ID:             uuid.New(),
			ModelName:      m.Name,
			Capability:     capability,
			Provider:       "google",
			IsPreview:      false, // Could parse from name
			IsFreeEligible: true,  // Default to true for now
			Priority:       100,   // Default priority
			Enabled:        true,
			MetaJSON:       datatypes.JSON(metaJSON),
		}

		if err := s.catalogRepo.Upsert(ctx, catalogModel); err != nil {
			log.Printf("failed to upsert model %s: %v", m.Name, err)
		}
	}

	return nil
}
