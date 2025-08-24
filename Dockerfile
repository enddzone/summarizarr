# Stage 1: Build Next.js frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /app/web

# Copy package files
COPY web/package*.json ./

# Install dependencies (include dev deps for build)
RUN npm ci

# Copy source code
# Add build arg to bust cache when frontend changes
ARG FRONTEND_CACHE_BUST=1
COPY web/ ./

# Build for production (static export)
RUN npm run build

# Stage 2: Build Go backend
FROM golang:1.25-alpine AS backend-builder

# Install SQLCipher and build dependencies, including static library
RUN apk add --no-cache gcc musl-dev sqlite-dev sqlite-static sqlcipher-dev pkgconf openssl-dev openssl-libs-static

WORKDIR /app

# Copy Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (excluding static assets to prevent cache conflicts)
COPY cmd/ cmd/
COPY internal/ internal/
COPY schema.sql .

# Remove any existing static assets to ensure clean state
RUN rm -rf internal/frontend/static

# Copy fresh frontend build output to embed location - ensures latest assets
COPY --from=frontend-builder /app/web/out internal/frontend/static/

# Build Go binary with SQLCipher support (CGO enabled)
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
ARG GOARCH

RUN CGO_ENABLED=1 \
    CGO_CFLAGS="$(pkg-config --cflags sqlcipher) -DSQLITE_HAS_CODEC" \
    CGO_LDFLAGS="$(pkg-config --libs sqlcipher) $(pkg-config --libs libcrypto libssl)" \
    go build \
    -tags="sqlite_crypt" \
    -ldflags="-w -s \
    -X 'summarizarr/internal/version.Version=${VERSION}' \
    -X 'summarizarr/internal/version.GitCommit=${GIT_COMMIT}' \
    -X 'summarizarr/internal/version.BuildTime=${BUILD_TIME}'" \
    -o summarizarr \
    ./cmd/summarizarr

# Stage 3: Runtime
FROM alpine:latest

# Update package index and add ca certificates, utilities, and SQLCipher runtime libraries
RUN apk update && \
    apk --no-cache add ca-certificates wget sqlcipher-libs openssl && \
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