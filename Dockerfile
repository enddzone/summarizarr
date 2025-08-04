# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /summarizarr ./cmd/summarizarr

# Final stage
FROM alpine:latest

WORKDIR /

COPY --from=builder /summarizarr /summarizarr
COPY schema.sql .

EXPOSE 8081

ENTRYPOINT ["/summarizarr"]