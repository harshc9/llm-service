package routing

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/harshc9/llm-service/internal/domain/keypool"
	"github.com/harshc9/llm-service/internal/domain/modelcatalog"
	"github.com/harshc9/llm-service/internal/domain/quota"
)

type Engine struct {
	keyRepo     keypool.Repository
	catalogRepo modelcatalog.Repository
	policyRepo  PolicyRepository
	quotaMgr    *quota.Manager
}

func NewEngine(
	keyRepo keypool.Repository,
	catalogRepo modelcatalog.Repository,
	policyRepo PolicyRepository,
	quotaMgr *quota.Manager,
) *Engine {
	return &Engine{
		keyRepo:     keyRepo,
		catalogRepo: catalogRepo,
		policyRepo:  policyRepo,
		quotaMgr:    quotaMgr,
	}
}

type Candidate struct {
	Project *keypool.ApiProject
	Key     *keypool.ApiKey
	Model   *modelcatalog.ModelCatalog
}

func (e *Engine) SelectRoute(ctx context.Context, feature string) (*Candidate, error) {
	// 1. Get the routing policy for the feature
	policy, err := e.policyRepo.GetByFeature(ctx, feature)
	if err != nil {
		log.Printf("SelectRoute error loading policy for %s: %v", feature, err)
		return nil, errors.New("routing policy not found for feature")
	}

	var modelChain []string
	if err := json.Unmarshal(policy.ModelChainJSON, &modelChain); err != nil {
		log.Printf("SelectRoute error parsing model chain: %v", err)
		return nil, errors.New("invalid routing policy configuration")
	}

	// 2. Get all models to verify availability and capability
	models, err := e.catalogRepo.ListEnabled(ctx)
	if err != nil {
		return nil, err
	}
	modelMap := make(map[string]*modelcatalog.ModelCatalog)
	for i := range models {
		modelMap[models[i].ModelName] = &models[i]
	}

	// 3. Get projects and keys
	projects, err := e.keyRepo.ListProjects(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("SelectRoute: Processing feature '%s' with chain %v", feature, modelChain)

	// 4. Strict Routing Logic based on Model Chain
	for _, targetModelName := range modelChain {
		m, ok := modelMap[targetModelName]
		if !ok {
			log.Printf("SelectRoute: Model %s not found in enabled catalog, skipping", targetModelName)
			continue
		}

		if m.Capability != feature {
			log.Printf("SelectRoute: Model %s capability mismatch (expected %s, got %s)", targetModelName, feature, m.Capability)
			continue
		}

		log.Printf("SelectRoute: Attempting to route via model %s", targetModelName)

		for _, p := range projects {
			if !p.Enabled {
				continue
			}

			// Check project health in Redis
			degraded, _ := e.quotaMgr.IsDegraded(ctx, p.ID)
			if degraded {
				log.Printf("SelectRoute: Project %s degraded, skipping", p.ID)
				continue
			}

			// Check quota
			allowed, _ := e.quotaMgr.CheckQuota(ctx, p.ID)
			if !allowed {
				log.Printf("SelectRoute: Project %s quota exceeded, skipping", p.ID)
				continue
			}

			for _, k := range p.Keys {
				if !k.Enabled || k.HealthState != "healthy" {
					continue
				}

				log.Printf("SelectRoute: SUCCESS - Selected Model=%s, Project=%s, Key=%s", m.ModelName, p.ID, k.ID)
				return &Candidate{
					Project: &p,
					Key:     &k,
					Model:   m,
				}, nil
			}
		}
		log.Printf("SelectRoute: Exhausted all projects for model %s, falling back to next model", targetModelName)
	}

	log.Printf("SelectRoute: Exhausted entire model chain, returning error")
	return nil, errors.New("no healthy routes available")
}

