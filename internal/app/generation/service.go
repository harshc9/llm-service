package generation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/harshc9/llm-service/internal/domain/quota"
	"github.com/harshc9/llm-service/internal/domain/routing"
	"github.com/harshc9/llm-service/internal/domain/usage"
	"github.com/harshc9/llm-service/internal/infra/crypto"
	"github.com/harshc9/llm-service/internal/infra/gemini"
)

type Service struct {
	routingEngine *routing.Engine
	geminiClient  *gemini.Client
	usageService  *usage.Service
	quotaMgr      *quota.Manager
	masterKey     []byte
}

func NewService(
	routingEngine *routing.Engine,
	geminiClient *gemini.Client,
	usageService *usage.Service,
	quotaMgr *quota.Manager,
	masterKey string,
) *Service {
	return &Service{
		routingEngine: routingEngine,
		geminiClient:  geminiClient,
		usageService:  usageService,
		quotaMgr:      quotaMgr,
		masterKey:     []byte(masterKey),
	}
}

type GenerateResponse struct {
	Text         string `json:"text"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

func (s *Service) Generate(ctx context.Context, clientID uuid.UUID, prompt string) (*GenerateResponse, error) {
	requestID := uuid.New()
	startTime := time.Now()

	var lastErr error
	for retry := 0; retry < 3; retry++ {
		// 1. Select route
		candidate, err := s.routingEngine.SelectRoute(ctx, "text")
		if err != nil {
			return nil, err
		}

		apiKey, err := crypto.Decrypt(candidate.Key.SecretCiphertext, s.masterKey)
		if err != nil {
			return nil, err
		}

		// 2. Call Gemini
		resp, statusCode, err := s.geminiClient.GenerateText(ctx, apiKey, candidate.Model.ModelName, prompt)

		latency := int(time.Since(startTime).Milliseconds())

		if err != nil {
			// Handle quota/availability errors
			if statusCode == 429 || statusCode == 503 || statusCode == 500 {
				s.quotaMgr.MarkDegraded(ctx, candidate.Project.ID, 5*time.Minute)
				lastErr = err
				continue // Try next candidate
			}
			return nil, err
		}

		// 3. Record success
		tokens := resp.UsageMetadata.TotalTokenCount
		s.quotaMgr.IncrementUsage(ctx, candidate.Project.ID, tokens)

		usageEvent := &usage.UsageEvent{
			ID:           uuid.New(),
			RequestID:    requestID,
			ClientID:     clientID,
			ProjectID:    candidate.Project.ID,
			ApiKeyID:     candidate.Key.ID,
			ModelName:    candidate.Model.ModelName,
			Feature:      "text",
			Endpoint:     "/v1/generate",
			StatusCode:   statusCode,
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
			LatencyMs:    latency,
			RetryCount:   retry,
			CreatedAt:    time.Now(),
		}
		_ = s.usageService.RecordEvent(ctx, usageEvent)

		return &GenerateResponse{
			Text:         resp.Candidates[0].Content.Parts[0].Text,
			Model:        candidate.Model.ModelName,
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		}, nil
	}

	return nil, fmt.Errorf("failed after retries: %v", lastErr)
}

func (s *Service) StreamGenerate(ctx context.Context, clientID uuid.UUID, prompt string) (*http.Response, *routing.Candidate, error) {
	// 1. Select route (with simple retry for initial connection)
	var candidate *routing.Candidate
	var resp *http.Response
	var lastErr error

	for retry := 0; retry < 3; retry++ {
		c, err := s.routingEngine.SelectRoute(ctx, "text")
		if err != nil {
			return nil, nil, err
		}

		apiKey, err := crypto.Decrypt(c.Key.SecretCiphertext, s.masterKey)
		if err != nil {
			return nil, nil, err
		}

		r, err := s.geminiClient.StreamGenerateText(ctx, apiKey, c.Model.ModelName, prompt)
		if err != nil {
			lastErr = err
			continue
		}

		if r.StatusCode != http.StatusOK {
			if r.StatusCode == 429 || r.StatusCode == 503 || r.StatusCode == 500 {
				s.quotaMgr.MarkDegraded(ctx, c.Project.ID, 5*time.Minute)
				r.Body.Close()
				lastErr = fmt.Errorf("rate limited or unavailable: %d", r.StatusCode)
				continue
			}
			r.Body.Close()
			return nil, nil, fmt.Errorf("gemini error: %d", r.StatusCode)
		}

		candidate = c
		resp = r
		break
	}

	if resp == nil {
		return nil, nil, fmt.Errorf("failed after retries: %v", lastErr)
	}

	return resp, candidate, nil
}

func (s *Service) GenerateGrounded(ctx context.Context, clientID uuid.UUID, prompt string) (*GenerateResponse, error) {
	requestID := uuid.New()
	startTime := time.Now()

	var lastErr error
	for retry := 0; retry < 3; retry++ {
		candidate, err := s.routingEngine.SelectRoute(ctx, "grounding")
		if err != nil {
			// Fallback to text models if grounding is not available
			candidate, err = s.routingEngine.SelectRoute(ctx, "text")
			if err != nil {
				return nil, err
			}
		}

		apiKey, err := crypto.Decrypt(candidate.Key.SecretCiphertext, s.masterKey)
		if err != nil {
			return nil, err
		}

		resp, statusCode, err := s.geminiClient.GenerateGroundedText(ctx, apiKey, candidate.Model.ModelName, prompt)
		latency := int(time.Since(startTime).Milliseconds())

		if err != nil {
			if statusCode == 429 || statusCode == 503 || statusCode == 500 {
				s.quotaMgr.MarkDegraded(ctx, candidate.Project.ID, 5*time.Minute)
				lastErr = err
				continue
			}
			return nil, err
		}

		usageEvent := &usage.UsageEvent{
			ID:           uuid.New(),
			RequestID:    requestID,
			ClientID:     clientID,
			ProjectID:    candidate.Project.ID,
			ApiKeyID:     candidate.Key.ID,
			ModelName:    candidate.Model.ModelName,
			Feature:      "grounding",
			Endpoint:     "/v1/generate/grounded",
			StatusCode:   statusCode,
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
			LatencyMs:    latency,
			CreatedAt:    time.Now(),
		}
		_ = s.usageService.RecordEvent(ctx, usageEvent)

		return &GenerateResponse{
			Text:         resp.Candidates[0].Content.Parts[0].Text,
			Model:        candidate.Model.ModelName,
			InputTokens:  resp.UsageMetadata.PromptTokenCount,
			OutputTokens: resp.UsageMetadata.CandidatesTokenCount,
		}, nil
	}

	return nil, fmt.Errorf("failed after retries: %v", lastErr)
}

type EmbedResponse struct {
	Embedding []float32 `json:"embedding"`
	Model     string    `json:"model"`
}

func (s *Service) Embed(ctx context.Context, clientID uuid.UUID, text string) (*EmbedResponse, error) {
	requestID := uuid.New()
	startTime := time.Now()

	var lastErr error
	for retry := 0; retry < 3; retry++ {
		candidate, err := s.routingEngine.SelectRoute(ctx, "embeddings")
		if err != nil {
			return nil, err
		}

		apiKey, err := crypto.Decrypt(candidate.Key.SecretCiphertext, s.masterKey)
		if err != nil {
			return nil, err
		}

		resp, statusCode, err := s.geminiClient.EmbedText(ctx, apiKey, candidate.Model.ModelName, text)
		latency := int(time.Since(startTime).Milliseconds())

		if err != nil {
			if statusCode == 429 || statusCode == 503 || statusCode == 500 {
				s.quotaMgr.MarkDegraded(ctx, candidate.Project.ID, 5*time.Minute)
				lastErr = err
				continue
			}
			return nil, err
		}

		usageEvent := &usage.UsageEvent{
			ID:         uuid.New(),
			RequestID:  requestID,
			ClientID:   clientID,
			ProjectID:  candidate.Project.ID,
			ApiKeyID:   candidate.Key.ID,
			ModelName:  candidate.Model.ModelName,
			Feature:    "embeddings",
			Endpoint:   "/v1/embeddings",
			StatusCode: statusCode,
			LatencyMs:  latency,
			CreatedAt:  time.Now(),
		}
		_ = s.usageService.RecordEvent(ctx, usageEvent)

		return &EmbedResponse{
			Embedding: resp.Embedding.Values,
			Model:     candidate.Model.ModelName,
		}, nil
	}

	return nil, fmt.Errorf("failed after retries: %v", lastErr)
}
