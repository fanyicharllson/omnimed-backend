// Package grpc is the gateway's own gRPC delivery layer — reserved for
// client-facing business services other than image-upload diagnosis
// (which stays on HTTP because it needs multipart/form-data). Auth,
// session, and medical-log services register onto the *grpc.Server
// returned by NewServer as they're built; nothing does yet.
package grpc

import (
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// NewServer constructs the gateway's gRPC server. It currently exposes
// only the standard gRPC health checking protocol (grpc.health.v1) —
// the same mechanism the Python inference service and the gateway's
// own /readyz check use — so it's already correctly probeable the
// moment the first real service is added to it.
func NewServer(logger *slog.Logger) *grpc.Server {
	server := grpc.NewServer()

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	logger.Debug("gateway grpc server initialized", "services", []string{"grpc.health.v1.Health"})

	return server
}
