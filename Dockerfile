# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for downloading dependencies
RUN apk update && apk add --no-cache git

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o flow-engine cmd/server/main.go

# Run stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy binary from builder
COPY --from=builder /app/flow-engine /flow-engine

# Use non-root user
USER 65532:65532

ENTRYPOINT ["/flow-engine"]
