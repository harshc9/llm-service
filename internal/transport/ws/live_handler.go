package ws

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/harshc9/llm-service/internal/domain/routing"
	"github.com/harshc9/llm-service/internal/infra/crypto"
	"github.com/labstack/echo/v4"
)

type LiveHandler struct {
	routingEngine *routing.Engine
	upgrader      websocket.Upgrader
	masterKey     []byte
}

func NewLiveHandler(routingEngine *routing.Engine, masterKey string) *LiveHandler {
	return &LiveHandler{
		routingEngine: routingEngine,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true }, // Relax for MVP
		},
		masterKey: []byte(masterKey),
	}
}

// Proxy godoc
// @Summary Gemini Live WebSocket Proxy
// @Description Establish a bidirectional multimodal session with Gemini Live
// @Tags generation
// @Security BearerAuth
// @Router /v1/live [get]
func (h *LiveHandler) Proxy(c echo.Context) error {
	clientID := c.Get("client_id").(uuid.UUID)
	_ = clientID // Use for tracking later

	// 1. Select route
	candidate, err := h.routingEngine.SelectRoute(c.Request().Context(), "live")
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "no healthy live routes"})
	}

	apiKey, err := crypto.Decrypt(candidate.Key.SecretCiphertext, h.masterKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "config error"})
	}

	// 2. Upgrade client connection
	clientConn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer clientConn.Close()

	// 3. Connect to Gemini Live API
	// Note: Using the v1beta BidiGenerateContent endpoint as per current docs
	geminiURL := fmt.Sprintf("wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1beta.GenerativeService/BidiGenerateContent?key=%s", apiKey)

	geminiConn, _, err := websocket.DefaultDialer.Dial(geminiURL, nil)
	if err != nil {
		log.Printf("failed to connect to gemini ws: %v", err)
		return nil // Connection already upgraded, just close
	}
	defer geminiConn.Close()

	// 4. Proxy messages
	errChan := make(chan error, 2)

	// Client -> Gemini
	go func() {
		for {
			messageType, payload, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := geminiConn.WriteMessage(messageType, payload); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Gemini -> Client
	go func() {
		for {
			messageType, payload, err := geminiConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(messageType, payload); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for any side to close or error
	select {
	case <-c.Request().Context().Done():
		return nil
	case err := <-errChan:
		log.Printf("ws proxy ended: %v", err)
		return nil
	}
}
