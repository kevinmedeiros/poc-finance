FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

# Production stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/server .
COPY --from=builder /app/internal/templates ./internal/templates

# Create data directory for SQLite
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set environment variables
ENV TZ=America/Sao_Paulo

# Run the application
CMD ["./server"]
