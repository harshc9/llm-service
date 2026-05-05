package http

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/harshc9/llm-service/internal/app/generation"
	"github.com/labstack/echo/v4"
)

type GenerationHandler struct {
	genService *generation.Service
}

func NewGenerationHandler(genService *generation.Service) *GenerationHandler {
	return &GenerationHandler{genService: genService}
}

type GenerateRequest struct {
	Prompt string `json:"prompt" validate:"required"`
}

// Generate godoc
// @Summary Generate text
// @Description Execute a text completion request with automatic routing and failover
// @Tags generation
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "Generation parameters"
// @Success 200 {object} generation.GenerateResponse
// @Router /v1/generate [post]
func (h *GenerationHandler) Generate(c echo.Context) error {
	var req GenerateRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	clientID := c.Get("client_id").(uuid.UUID)

	resp, err := h.genService.Generate(c.Request().Context(), clientID, req.Prompt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

// StreamGenerate godoc
// @Summary Stream text generation
// @Description Execute a streaming text completion request (SSE)
// @Tags generation
// @Security BearerAuth
// @Accept json
// @Produce text/event-stream
// @Param request body GenerateRequest true "Generation parameters"
// @Success 200 {string} string "SSE stream of chunks"
// @Router /v1/generate/stream [post]
func (h *GenerationHandler) StreamGenerate(c echo.Context) error {
	var req GenerateRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	clientID := c.Get("client_id").(uuid.UUID)

	resp, _, err := h.genService.StreamGenerate(c.Request().Context(), clientID, req.Prompt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	// Set headers for SSE
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		if line == "[" || line == "]" || line == "," {
			continue
		}
		if strings.HasPrefix(line, ",") {
			line = line[1:]
		}

		if line == "" {
			continue
		}

		fmt.Fprintf(c.Response().Writer, "data: %s\n\n", line)
		c.Response().Flush()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(c.Response().Writer, "event: error\ndata: %s\n\n", err.Error())
		c.Response().Flush()
	}

	return nil
}

// GenerateGrounded godoc
// @Summary Generate grounded text
// @Description Execute a text completion request with Google Search grounding
// @Tags generation
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body GenerateRequest true "Generation parameters"
// @Success 200 {object} generation.GenerateResponse
// @Router /v1/generate/grounded [post]
func (h *GenerationHandler) GenerateGrounded(c echo.Context) error {
	var req GenerateRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	clientID := c.Get("client_id").(uuid.UUID)

	resp, err := h.genService.GenerateGrounded(c.Request().Context(), clientID, req.Prompt)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

type EmbedRequest struct {
	Text  string `json:"text" validate:"required"`
}

// Embed godoc
// @Summary Generate embeddings
// @Description Execute a text embedding request
// @Tags generation
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body EmbedRequest true "Embedding parameters"
// @Success 200 {object} generation.EmbedResponse
// @Router /v1/embeddings [post]
func (h *GenerationHandler) Embed(c echo.Context) error {
	var req EmbedRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	clientID := c.Get("client_id").(uuid.UUID)

	resp, err := h.genService.Embed(c.Request().Context(), clientID, req.Text)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, resp)
}

