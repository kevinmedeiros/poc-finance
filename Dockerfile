FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO (needed for SQLite)
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Production stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Copy binary from builder
COPY --from=builder /app/server .

# Copy templates
COPY --from=builder /app/internal/templates ./internal/templates

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Set environment variables
ENV TZ=America/Sao_Paulo
ENV DATABASE_PATH=/app/data/finance.db
ENV ENV=production

# Expose port (Render will set PORT env var)
EXPOSE 8080

# Run the application
CMD ["./server"]
