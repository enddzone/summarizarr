# Stage 1: Build Next.js frontend
FROM node:24-alpine AS frontend-builder

WORKDIR /app/web

# Install minimal runtime libs for native modules on musl
RUN apk add --no-cache libc6-compat

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
FROM ubuntu:24.04 AS backend-builder

ARG GO_VERSION=1.25.1
ENV DEBIAN_FRONTEND=noninteractive
ENV PATH=/usr/local/go/bin:$PATH

# Install toolchain, SQLCipher dev headers, and download Go
RUN set -eux; \
    rm -rf /var/lib/apt/lists/*; \
    for attempt in 1 2 3; do \
      if apt-get update; then \
        break; \
      fi; \
      if [ "$attempt" = "3" ]; then \
        echo "apt-get update failed after $attempt attempts" >&2; \
        exit 1; \
      fi; \
      sleep 5; \
    done; \
    apt-get install -y --no-install-recommends \
      build-essential pkg-config curl ca-certificates libsqlcipher-dev libssl-dev; \
    rm -rf /var/lib/apt/lists/*; \
    arch=$(dpkg --print-architecture); \
    case "$arch" in \
      amd64) GOARCH=amd64 ;; \
      arm64) GOARCH=arm64 ;; \
      *) echo "Unsupported arch: $arch" && exit 1 ;; \
    esac && \
    curl -fsSL https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz -o /tmp/go.tgz && \
    tar -C /usr/local -xzf /tmp/go.tgz && \
    rm /tmp/go.tgz && \
    # Ensure go is available
    go version && \
    # Align lib name expected by go-sqlite3 when using libsqlite3 tag
    bash -lc 'libdir=$(dpkg-architecture -qDEB_HOST_MULTIARCH 2>/dev/null || echo aarch64-linux-gnu); \
      if [ -e /usr/lib/$libdir/libsqlcipher.so ] && [ ! -e /usr/lib/$libdir/libsqlite3.so ]; then \
        ln -s /usr/lib/$libdir/libsqlcipher.so /usr/lib/$libdir/libsqlite3.so; \
      fi'

# Enable CGO globally; actual flags are set at build invocation time below
ENV CGO_ENABLED=1

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

RUN CGO_CFLAGS="-I/usr/include/sqlcipher -DSQLITE_HAS_CODEC" \
    CGO_LDFLAGS="-lsqlcipher -lssl -lcrypto" \
    go build \
    -tags="sqlite_crypt libsqlite3" \
    -ldflags="-w -s \
    -X 'summarizarr/internal/version.Version=${VERSION}' \
    -X 'summarizarr/internal/version.GitCommit=${GIT_COMMIT}' \
    -X 'summarizarr/internal/version.BuildTime=${BUILD_TIME}'" \
    -o summarizarr \
    ./cmd/summarizarr

# Stage 3: Runtime (Debian/Ubuntu, glibc)
FROM ubuntu:24.04

RUN set -eux; \
    rm -rf /var/lib/apt/lists/*; \
    for attempt in 1 2 3; do \
      if apt-get update; then \
        break; \
      fi; \
      if [ "$attempt" = "3" ]; then \
        echo "apt-get update failed after $attempt attempts" >&2; \
        exit 1; \
      fi; \
      sleep 5; \
    done; \
    apt-get install -y --no-install-recommends ca-certificates wget sqlcipher; \
    rm -rf /var/lib/apt/lists/*; \
    bash -lc 'libdir=$(dpkg-architecture -qDEB_HOST_MULTIARCH 2>/dev/null || echo aarch64-linux-gnu); \
      if [ -e /usr/lib/$libdir/libsqlcipher.so ] && [ ! -e /usr/lib/$libdir/libsqlite3.so ]; then \
        ln -s /usr/lib/$libdir/libsqlcipher.so /usr/lib/$libdir/libsqlite3.so; \
      fi'; \
    groupadd -g 1001 summarizarr; \
    useradd -u 1001 -g 1001 -M -s /bin/sh summarizarr

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
