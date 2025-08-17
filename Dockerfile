# Stage 1: Build Next.js frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies (include dev deps for build)
RUN npm ci

# Copy source code
COPY web/ ./

# Build for production
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.24-alpine AS backend-builder

WORKDIR /app

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy frontend build output to embed location
COPY --from=frontend-builder /app/web/out internal/frontend/static/

# Build static Go binary with version information
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
ARG GOARCH

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH:-amd64} go build \
    -ldflags="-w -s -extldflags '-static' \
             -X 'summarizarr/internal/version.Version=${VERSION}' \
             -X 'summarizarr/internal/version.GitCommit=${GIT_COMMIT}' \
             -X 'summarizarr/internal/version.BuildTime=${BUILD_TIME}'" \
    -a -installsuffix cgo \
    -o summarizarr \
    ./cmd/summarizarr

# Stage 3: Runtime
FROM alpine:latest

# Update package index and add ca certificates for HTTPS and basic utilities
RUN apk update && \
    apk --no-cache add ca-certificates wget && \
    addgroup -g 1001 summarizarr && \
    adduser -D -s /bin/sh -u 1001 -G summarizarr summarizarr

WORKDIR /app

# Copy binary and schema
COPY --from=backend-builder /app/summarizarr .
COPY --from=backend-builder /app/schema.sql .

# Create data directory and set permissions
RUN mkdir -p /data /config && \
    chown -R summarizarr:summarizarr /app /data /config

# Switch to non-root user
USER summarizarr

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Container metadata labels (OCI compliant)
LABEL org.opencontainers.image.title="Summarizarr"
LABEL org.opencontainers.image.description="AI-powered Signal message summarizer"
LABEL org.opencontainers.image.vendor="EnddZone"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/enddzone/summarizarr"

# Expose unified port
EXPOSE 8080

# Default environment variables
ENV DATABASE_PATH=/data/summarizarr.db
ENV AI_PROVIDER=local
ENV SUMMARIZATION_INTERVAL=1h

ENTRYPOINT ["./summarizarr"]