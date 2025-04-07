# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod ./
COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build server
RUN CGO_ENABLED=0 GOOS=linux go build -o thothnetwork-server ./cmd/server

# Build CLI
RUN CGO_ENABLED=0 GOOS=linux go build -o thothnetwork-cli ./cmd/cli

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/thothnetwork-server /app/
COPY --from=builder /app/thothnetwork-cli /app/

# Create config directory
RUN mkdir -p /etc/thothnetwork

# Expose ports
EXPOSE 8080
EXPOSE 9090

# Set entrypoint
ENTRYPOINT ["/app/thothnetwork-server"]
