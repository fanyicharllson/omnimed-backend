// Package grpc wraps the gRPC connection to the Python AI inference
// service, presenting the TriageService contract as a plain Go
// interface to the usecase layer.
package grpc

import (
	"context"
	"fmt"
	"time"

	triagepb "github.com/fanyicharllson/omnimed-backend/pb/triage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// TriageClient talks to the AI inference service over gRPC.
type TriageClient struct {
	conn        *grpc.ClientConn
	client      triagepb.TriageServiceClient
	healthCheck grpc_health_v1.HealthClient
	timeout     time.Duration
}

// NewTriageClient dials the inference service at addr. The dial is
// non-blocking; connection failures surface on the first RPC call.
func NewTriageClient(addr string, timeout time.Duration) (*TriageClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("triage client: dial %s: %w", addr, err)
	}

	return &TriageClient{
		conn:        conn,
		client:      triagepb.NewTriageServiceClient(conn),
		healthCheck: grpc_health_v1.NewHealthClient(conn),
		timeout:     timeout,
	}, nil
}

// Close releases the underlying gRPC connection.
func (c *TriageClient) Close() error {
	return c.conn.Close()
}

// DiagnoseBreastCancer forwards image bytes to the inference service's
// breast cancer classifier and returns its raw response.
func (c *TriageClient) DiagnoseBreastCancer(ctx context.Context, req *triagepb.ImageRequest) (*triagepb.DiagnosisResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.DiagnoseBreastCancer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("triage client: DiagnoseBreastCancer: %w", err)
	}

	return resp, nil
}

// CheckHealth asks the inference service's standard gRPC health
// endpoint (grpc.health.v1) whether it is serving. It uses a short,
// fixed timeout independent of the usual inference timeout, since a
// health probe should fail fast rather than wait as long as a real
// diagnosis call would.
func (c *TriageClient) CheckHealth(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := c.healthCheck.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return fmt.Errorf("triage client: health check: %w", err)
	}

	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("triage client: inference service status is %s", resp.GetStatus())
	}

	return nil
}
