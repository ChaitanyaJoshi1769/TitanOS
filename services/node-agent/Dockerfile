# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy both node-agent and scheduler modules
COPY go.mod go.sum* ./
COPY ../scheduler ../scheduler/

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the node agent
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o node-agent main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /build/node-agent .

# Environment defaults
ENV NODE_ID=node-default
ENV SCHEDULER_ADDRESS=scheduler:50051
ENV CPU_CORES=4
ENV MEMORY_GB=8
ENV HEARTBEAT_INTERVAL=10s

# Run the node agent
CMD ["./node-agent"]
