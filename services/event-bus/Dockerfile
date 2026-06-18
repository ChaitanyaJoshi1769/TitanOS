FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o event-bus .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/event-bus .
EXPOSE 50053 8002
ENV KAFKA_HOST=kafka:9092
ENV GRPC_PORT=50053
CMD ["./event-bus"]
