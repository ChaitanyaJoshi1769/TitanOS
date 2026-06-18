package server

import (
	"log"
	"net/http"

	pb "github.com/ChaitanyaJoshi1769/TitanOS/services/scheduler/pb"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// NewGRPCServer creates and configures a new gRPC server
func NewGRPCServer(scheduler *Scheduler) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * 60,
		}),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
	}

	server := grpc.NewServer(opts...)

	// Register Scheduler service
	pb.RegisterSchedulerServiceServer(server, scheduler)

	// Initialize Prometheus metrics
	grpc_prometheus.Register(server)

	return server
}

// StartMetricsServer starts the Prometheus metrics HTTP server
func StartMetricsServer(port string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// This will be populated by Prometheus middleware
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("# Metrics endpoint ready\n"))
	}))

	log.Printf("Metrics server listening on :%s", port)
	return http.ListenAndServe(":"+port, mux)
}
