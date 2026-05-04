# Gemini LLM Service

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Swagger Docs](https://img.shields.io/badge/Swagger-Documentation-green)](http://localhost:8080/swagger/index.html)

A high-availability, multi-tenant API proxy for Google's Gemini LLMs. Designed to bypass rate limits through multi-project routing, provide automatic failover, and track detailed usage across projects and API keys.

## 🚀 Key Features

- **Multi-Project Routing:** Distribute traffic across multiple Google Cloud projects to aggregate rate limits.
- **Automatic Failover:** Detects `429 RESOURCE_EXHAUSTED` errors and instantly switches to the next available healthy project/key.
- **Key Pooling:** Support for up to 100 API keys with priority-based selection.
- **Multi-Tenant Authentication:** Register multiple clients, each with their own secure bearer token.
- **Dynamic Model Catalog:** Syncs available models directly from Google's API to maintain an up-to-date registry.
- **Encrypted Secrets:** API keys are encrypted at rest using AES-256 with a master key.
- **Real-time Monitoring:** Distributed quota tracking (RPM/TPM) using Redis and a built-in admin dashboard.
- **Comprehensive API:** Supports standard generation, streaming (SSE), embeddings, search grounding, and the Live (WebSocket) API.

## 🏗 Architecture

The project follows **Domain-Driven Design (DDD)** principles for a clean, maintainable structure:

- `internal/domain`: Core business logic (Routing, Quota, Key Management, Usage Tracking).
- `internal/app`: Use cases (Text Generation, Live Sessions, Admin Services).
- `internal/infra`: Infrastructure adapters (PostgreSQL, Redis, Gemini API Client).
- `internal/transport`: API layer (HTTP handlers, WebSocket proxy, Auth Middleware).

## 🛠 Tech Stack

- **Language:** Go 1.25
- **Web Framework:** [Echo](https://echo.labstack.com/)
- **Database:** [PostgreSQL 15](https://www.postgresql.org/)
- **ORM:** [GORM](https://gorm.io/)
- **Caching/Coordination:** [Redis 7](https://redis.io/)
- **Documentation:** [Swagger/swag](https://github.com/swaggo/swag)
- **Containerization:** Docker & Docker Compose

## 🚦 Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.25 (for local development without Docker)

### Local Setup

1. **Clone the repository:**
   ```bash
   git clone https://github.com/user/llm-service.git
   cd llm-service
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Update AES_MASTER_KEY with a 32-byte string
   # Example: openssl rand -base64 32
   ```

3. **Start infrastructure:**
   ```bash
   docker-compose up -d
   ```

4. **Run the application:**
   ```bash
   go run cmd/api/main.go
   ```
   The API will be available at `http://localhost:8080`.

## ⚙️ Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENVIRONMENT` | `development` or `production` | `development` |
| `AES_MASTER_KEY` | 32-byte key for API key encryption | *(Required)* |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://...` |
| `REDIS_URL` | Redis connection string | `redis://...` |

## 📖 API Reference

### Public API Endpoints
All generation endpoints require `Authorization: Bearer <token>` header.

- `POST /v1/generate`: Standard text generation.
- `POST /v1/generate/stream`: Streaming text generation (Server-Sent Events).
- `POST /v1/generate/grounded`: Text generation with Google Search grounding.
- `POST /v1/embeddings`: Generate text embeddings.
- `GET /v1/live`: WebSocket proxy for Gemini Live API.

### Admin API Endpoints
- `POST /v1/admin/projects`: Register a new Google Project.
- `POST /v1/admin/keys`: Add an encrypted API key to a project.
- `POST /v1/admin/clients`: Register a new client and get an API token.
- `POST /v1/admin/sync-models`: Sync the local model catalog with Google.
- `GET /v1/admin/stats`: Get system-wide usage statistics.

### Documentation & Dashboard
- **Swagger Docs:** `http://localhost:8080/swagger/index.html`
- **Admin Dashboard:** `http://localhost:8080/dashboard`

## 🛠 Admin Tasks

### 1. Register a Client
To use the API, first register a client to receive a token:
```bash
curl -X POST http://localhost:8080/v1/admin/clients \
  -H "Content-Type: application/json" \
  -d '{"name": "Internal App"}'
```

### 2. Add API Keys
Projects must be created before adding keys:
```bash
# 1. Create Project
curl -X POST http://localhost:8080/v1/admin/projects \
  -d '{"name": "My Project", "provider": "google"}'

# 2. Add Key (using Project ID from above)
curl -X POST http://localhost:8080/v1/admin/keys \
  -d '{"project_id": "<uuid>", "alias": "key-1", "api_key": "AIza...", "priority": 1}'
```

### 3. Sync Models
Refresh the available Gemini models:
```bash
curl -X POST http://localhost:8080/v1/admin/sync-models
```

## 📝 License

Distributed under the Apache 2.0 License. See `LICENSE` for more information (Not found in repository, assuming Apache 2.0 based on `main.go` headers).
