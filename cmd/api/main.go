package main

import (
	"context"
	"embed"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"gorm.io/datatypes"

	_ "github.com/harshc9/llm-service/docs"
	"github.com/harshc9/llm-service/internal/app/admin"
	"github.com/harshc9/llm-service/internal/app/generation"
	"github.com/harshc9/llm-service/internal/domain/client"
	"github.com/harshc9/llm-service/internal/domain/keypool"
	"github.com/harshc9/llm-service/internal/domain/modelcatalog"
	"github.com/harshc9/llm-service/internal/domain/quota"
	"github.com/harshc9/llm-service/internal/domain/routing"
	"github.com/harshc9/llm-service/internal/domain/usage"
	"github.com/harshc9/llm-service/internal/infra/cache"
	"github.com/harshc9/llm-service/internal/infra/config"
	"github.com/harshc9/llm-service/internal/infra/gemini"
	"github.com/harshc9/llm-service/internal/infra/postgres"
	transport_http "github.com/harshc9/llm-service/internal/transport/http"
	app_middleware "github.com/harshc9/llm-service/internal/transport/middleware"
	transport_ws "github.com/harshc9/llm-service/internal/transport/ws"
)

// @title Gemini LLM Service API
// @version 1.0
// @description High-availability Gemini proxy with multi-project routing and failover.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
//
//go:embed dashboard.html
var dashboardEmbed embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Infra
	db, err := postgres.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	rdb, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	geminiClient := gemini.NewClient()

	// Repositories
	clientRepo := client.NewRepository(db)
	keyRepo := keypool.NewRepository(db)
	catalogRepo := modelcatalog.NewRepository(db)
	policyRepo := routing.NewPolicyRepository(db)

	seedDefaultPolicies(policyRepo)

	// Managers/Services
	quotaMgr := quota.NewManager(rdb, cfg)
	usageService := usage.NewService(db)
	routingEngine := routing.NewEngine(keyRepo, catalogRepo, policyRepo, quotaMgr)

	syncService := admin.NewSyncService(geminiClient, catalogRepo, keyRepo, cfg.AESMasterKey)
	statsService := admin.NewStatsService(db)
	genService := generation.NewService(routingEngine, geminiClient, usageService, quotaMgr, cfg.AESMasterKey)

	// Handlers
	adminHandler := transport_http.NewAdminHandler(syncService, statsService, keyRepo, clientRepo, cfg.AESMasterKey)
	genHandler := transport_http.NewGenerationHandler(genService)
	liveHandler := transport_ws.NewLiveHandler(routingEngine, cfg.AESMasterKey)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Public routes
	e.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/dashboard")
	})
	e.GET("/healthz", func(c echo.Context) error {
		return c.String(200, "OK")
	})
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Admin routes
	adminGroup := e.Group("/v1/admin")
	adminGroup.POST("/projects", adminHandler.CreateProject)
	adminGroup.POST("/keys", adminHandler.CreateKey)
	adminGroup.POST("/clients", adminHandler.CreateClient)
	adminGroup.POST("/sync-models", adminHandler.SyncModels)
	adminGroup.GET("/stats", adminHandler.GetStats)

	// Dashboard
	e.GET("/dashboard", func(c echo.Context) error {
		content, _ := dashboardEmbed.ReadFile("dashboard.html")
		return c.HTML(http.StatusOK, string(content))
	})

	// Protected routes
	apiGroup := e.Group("/v1")
	apiGroup.Use(app_middleware.ClientAuth(clientRepo))
	apiGroup.POST("/generate", genHandler.Generate)
	apiGroup.POST("/generate/stream", genHandler.StreamGenerate)
	apiGroup.POST("/generate/grounded", genHandler.GenerateGrounded)
	apiGroup.POST("/embeddings", genHandler.Embed)
	apiGroup.GET("/live", liveHandler.Proxy)

	log.Printf("Starting server on port %s", cfg.Port)
	if err := e.Start(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func seedDefaultPolicies(repo routing.PolicyRepository) {
	ctx := context.Background()
	policies := []struct {
		Feature string
		Models  []string
	}{
		{
			Feature: "text",
			Models:  []string{"models/gemini-3.1-flash-lite-preview", "models/gemini-3-flash-preview", "models/gemini-2.5-flash-lite", "models/gemini-2.5-flash"},
		},
		{
			Feature: "live",
			Models:  []string{"models/gemini-3.1-flash-live-preview", "models/gemini-2.5-flash-native-audio-preview-12-2025"},
		},
		{
			Feature: "embeddings",
			Models:  []string{"models/gemini-embedding-2", "models/gemini-embedding-001"},
		},
		{
			Feature: "grounding",
			Models:  []string{"models/gemini-2.5-flash-lite", "models/gemini-2.5-flash", "models/gemini-2.5-pro"},
		},
	}

	for _, p := range policies {
		_, err := repo.GetByFeature(ctx, p.Feature)
		if err == nil {
			continue // Policy already exists
		}

		chainBytes, _ := json.Marshal(p.Models)
		err = repo.Create(ctx, &routing.RoutingPolicy{
			ID:             uuid.New(),
			Feature:        p.Feature,
			Strategy:       "priority",
			ModelChainJSON: datatypes.JSON(chainBytes),
			Enabled:        true,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
		if err != nil {
			log.Printf("Failed to seed policy for %s: %v", p.Feature, err)
		} else {
			log.Printf("Seeded default policy for %s", p.Feature)
		}
	}
}
