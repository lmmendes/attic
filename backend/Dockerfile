# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/attic ./cmd/server

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates su-exec tzdata

# Copy binary from builder
COPY --from=builder /app/attic /app/attic
COPY --chmod=755 docker-entrypoint.sh /app/docker-entrypoint.sh

# Create non-root user
RUN adduser -D -g '' appuser \
    && mkdir -p /data/uploads \
    && chown appuser:appuser /data/uploads

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/docker-entrypoint.sh"]
CMD ["/app/attic"]
