package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/harshc9/llm-service/internal/app/admin"
	"github.com/harshc9/llm-service/internal/domain/client"
	"github.com/harshc9/llm-service/internal/domain/keypool"
	"github.com/harshc9/llm-service/internal/infra/crypto"
	"github.com/labstack/echo/v4"
)

type AdminHandler struct {
	syncService  *admin.SyncService
	statsService *admin.StatsService
	keyRepo      keypool.Repository
	clientRepo   client.Repository
	masterKey    []byte
}

func NewAdminHandler(
	syncService *admin.SyncService,
	statsService *admin.StatsService,
	keyRepo keypool.Repository,
	clientRepo client.Repository,
	masterKey string,
) *AdminHandler {
	return &AdminHandler{
		syncService:  syncService,
		statsService: statsService,
		keyRepo:      keyRepo,
		clientRepo:   clientRepo,
		masterKey:    []byte(masterKey),
	}
}

// GetStats godoc
// @Summary Get usage stats
// @Description Retrieve aggregated usage statistics and model distribution
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /v1/admin/stats [get]
func (h *AdminHandler) GetStats(c echo.Context) error {
	summary, err := h.statsService.GetUsageSummary(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	modelUsage, err := h.statsService.GetModelUsage(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"summary":     summary,
		"model_usage": modelUsage,
	})
}

type CreateProjectRequest struct {
	Name     string `json:"name" validate:"required"`
	Provider string `json:"provider" validate:"required"`
}

// CreateProject godoc
// @Summary Create a new project
// @Description Register a new Google Cloud project in the system
// @Tags admin
// @Accept json
// @Produce json
// @Param project body CreateProjectRequest true "Project details"
// @Success 201 {object} keypool.ApiProject
// @Router /v1/admin/projects [post]
func (h *AdminHandler) CreateProject(c echo.Context) error {
	var req CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	project := &keypool.ApiProject{
		ID:        uuid.New(),
		Name:      req.Name,
		Provider:  req.Provider,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.keyRepo.CreateProject(c.Request().Context(), project); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, project)
}

type CreateKeyRequest struct {
	ProjectID uuid.UUID `json:"project_id" validate:"required"`
	Alias     string    `json:"alias" validate:"required"`
	ApiKey    string    `json:"api_key" validate:"required"`
	Priority  int       `json:"priority"`
}

// CreateKey godoc
// @Summary Add an API key
// @Description Encrypt and store a new Gemini API key for a project
// @Tags admin
// @Accept json
// @Produce json
// @Param key body CreateKeyRequest true "Key details"
// @Success 201 {object} keypool.ApiKey
// @Router /v1/admin/keys [post]
func (h *AdminHandler) CreateKey(c echo.Context) error {
	var req CreateKeyRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ciphertext, err := crypto.Encrypt(req.ApiKey, h.masterKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
	}

	key := &keypool.ApiKey{
		ID:               uuid.New(),
		ProjectID:        req.ProjectID,
		Alias:            req.Alias,
		SecretCiphertext: ciphertext,
		Priority:         req.Priority,
		Enabled:          true,
		HealthState:      "healthy",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := h.keyRepo.CreateKey(c.Request().Context(), key); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, key)
}

// SyncModels godoc
// @Summary Sync model catalog
// @Description Fetch and update available Gemini models from the provider
// @Tags admin
// @Produce json
// @Success 200 {object} map[string]string
// @Router /v1/admin/sync-models [post]
func (h *AdminHandler) SyncModels(c echo.Context) error {
	if err := h.syncService.SyncModels(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"message": "sync complete"})
}

type CreateClientRequest struct {
	Name string `json:"name" validate:"required"`
}

// CreateClient godoc
// @Summary Create a client
// @Description Register a new multi-tenant client and generate a bearer token
// @Tags admin
// @Accept json
// @Produce json
// @Param client body CreateClientRequest true "Client details"
// @Success 201 {object} map[string]interface{}
// @Router /v1/admin/clients [post]
func (h *AdminHandler) CreateClient(c echo.Context) error {
	var req CreateClientRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	token := uuid.New().String() // Simple token for now

	clientEntity := &client.Client{
		ID:        uuid.New(),
		Name:      req.Name,
		ApiToken:  token,
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.clientRepo.Create(c.Request().Context(), clientEntity); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Return token only once
	return c.JSON(http.StatusCreated, map[string]interface{}{
		"client":    clientEntity,
		"api_token": token,
	})
}
