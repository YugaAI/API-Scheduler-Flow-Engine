# =========================
# Build Stage
# =========================
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o flow-engine cmd/server/main.go


# =========================
# Runtime Stage
# =========================
FROM alpine:3.22

WORKDIR /

RUN apk add --no-cache \
    bash \
    git \
    curl \
    ca-certificates \
    docker-cli \
    tzdata

COPY --from=builder /app/flow-engine /flow-engine

RUN addgroup -S app && adduser -S app -G app

RUN mkdir -p /app/logs && \
    chown -R app:app /app

USER app

ENTRYPOINT ["/flow-engine"]