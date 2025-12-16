# Build stage
FROM golang:1.23-alpine AS builder

# Enable automatic toolchain download for dependencies requiring newer Go versions
ENV GOTOOLCHAIN=auto

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git curl

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate Goa code
RUN go install goa.design/goa/v3/cmd/goa@latest
RUN goa gen springstreet/api/design

# Build application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o springstreet-api cmd/api/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/springstreet-api .

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

# Run application
CMD ["./springstreet-api"]


