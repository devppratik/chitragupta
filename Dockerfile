# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build server
RUN CGO_ENABLED=1 go build -o chitra-server ./cmd/chitra-server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/chitra-server .

# Create storage directory
RUN mkdir -p /app/storage/packages

# Expose port
EXPOSE 8080

# Environment defaults
ENV DB_TYPE=sqlite \
    DB_DSN=/app/chitragupta.db \
    STORAGE_PATH=/app/storage/packages \
    PORT=8080

CMD ["./chitra-server"]
