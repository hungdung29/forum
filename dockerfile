# Stage 1: Build the Go binary
FROM golang:1.22.3-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
# CGO_ENABLED=1 is required for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -o forum ./cmd/main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install runtime dependencies (ca-certificates for HTTPS, wget for healthcheck)
RUN apk --no-cache add ca-certificates sqlite-libs wget

# Create non-root user for security
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/forum .

# Copy web assets and templates
COPY --chown=appuser:appuser web ./web

# Copy database migrations
COPY --chown=appuser:appuser server/database/migrations ./server/database/migrations

# Create directory for database with proper permissions
RUN mkdir -p server/database && \
    chown -R appuser:appuser server

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV PORT=8080 \
    ENV=production \
    DB_PATH=server/database/database.db \
    BASE_PATH=/app/

# Run the application
CMD ["./forum"]