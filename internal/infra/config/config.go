package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	AESMasterKey   string
	Environment    string // development, production
	MaxRPM         int
	MaxRPD         int
	MaxTPM         int
}

func Load() (*Config, error) {
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	cfg := &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/llm_service?sslmode=disable"),
		RedisURL:     getEnv("REDIS_URL", "redis://localhost:6379/0"),
		AESMasterKey: getEnv("AES_MASTER_KEY", "default-32-byte-master-key-here!"), // Exactly 32 bytes
		Environment:  getEnv("ENVIRONMENT", "development"),
		MaxRPM:       getEnvAsInt("MAX_RPM", 15),
		MaxRPD:       getEnvAsInt("MAX_RPD", 1500),
		MaxTPM:       getEnvAsInt("MAX_TPM", 1000000),
	}

	// Validate AES key length (must be 16, 24, or 32 bytes)
	keyLen := len(cfg.AESMasterKey)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, fmt.Errorf("invalid AES_MASTER_KEY length: %d bytes (must be 16, 24, or 32)", keyLen)
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
