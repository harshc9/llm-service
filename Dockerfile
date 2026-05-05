# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate swagger docs (optional, but ensures they are fresh)
RUN go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go

# Build the application
RUN go build -o llm-service ./cmd/api/main.go

# Final stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Copy binary and embedded files from builder
COPY --from=builder /app/llm-service .
COPY --from=builder /app/cmd/api/dashboard.html .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./llm-service"]
