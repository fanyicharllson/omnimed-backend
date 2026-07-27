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
)

// TriageClient talks to the AI inference service over gRPC.
type TriageClient struct {
	conn    *grpc.ClientConn
	client  triagepb.TriageServiceClient
	timeout time.Duration
}

// NewTriageClient dials the inference service at addr. The dial is
// non-blocking; connection failures surface on the first RPC call.
func NewTriageClient(addr string, timeout time.Duration) (*TriageClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("triage client: dial %s: %w", addr, err)
	}

	return &TriageClient{
		conn:    conn,
		client:  triagepb.NewTriageServiceClient(conn),
		timeout: timeout,
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
