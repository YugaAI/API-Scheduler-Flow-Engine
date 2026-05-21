<<<<<<< HEAD
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
=======
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

# Install runtime dependencies
RUN apk add --no-cache \
    bash \
    git \
    curl \
    ca-certificates \
    docker-cli \
    tzdata

# Copy binary
COPY --from=builder /app/flow-engine /flow-engine

# Non-root user
RUN addgroup -S app && adduser -S app -G app

USER app

>>>>>>> 0927876 (fixing)
ENTRYPOINT ["/flow-engine"]
