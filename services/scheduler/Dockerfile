# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git protobuf-dev

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate protobuf code (if needed)
# RUN protoc --go_out=. --go-grpc_out=. proto/*.proto

# Build the scheduler
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o scheduler main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /build/scheduler .

# Expose gRPC and metrics ports
EXPOSE 50051 8001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8001/metrics || exit 1

# Run the scheduler
CMD ["./scheduler"]
