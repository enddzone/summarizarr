# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /summarizarr ./cmd/summarizarr

# Final stage - use Ubuntu for glibc compatibility
FROM ubuntu:24.04

# Install required packages for Ollama
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /

COPY --from=builder /summarizarr /summarizarr
COPY schema.sql .

EXPOSE 8081

ENTRYPOINT ["/summarizarr"]