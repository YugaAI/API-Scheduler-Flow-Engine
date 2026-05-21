# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk update && apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o flow-engine cmd/server/main.go

# Run stage
FROM alpine:3.21
WORKDIR /

# Security: non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Install shell dan timezone data
RUN apk add --no-cache bash tzdata ca-certificates

COPY --from=builder /app/flow-engine /flow-engine

USER appuser
ENTRYPOINT ["/flow-engine"]
