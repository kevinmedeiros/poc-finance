FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Set GOTOOLCHAIN to auto to allow downloading newer toolchain if needed
ENV GOTOOLCHAIN=auto

RUN go mod download

# Copy source code
COPY . .

# Build the application (CGO disabled - PostgreSQL driver doesn't need it)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Production stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/server .

# Copy templates
COPY --from=builder /app/internal/templates ./internal/templates

# Set environment variables
ENV TZ=America/Sao_Paulo
ENV ENV=production

# Expose port (Render will set PORT env var)
EXPOSE 8080

# Run the application
CMD ["./server"]
