package middleware

import (
	"net/http"
	"strings"

	"github.com/harshc9/llm-service/internal/domain/client"
	"github.com/labstack/echo/v4"
)

func ClientAuth(repo client.Repository) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization format"})
			}

			token := parts[1]
			clientEntity, err := repo.GetByToken(c.Request().Context(), token)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
			}

			if !clientEntity.Enabled {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "client disabled"})
			}

			// Store client info in context
			c.Set("client_id", clientEntity.ID)
			c.Set("client_name", clientEntity.Name)

			return next(c)
		}
	}
}
