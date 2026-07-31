// Command gateway is the reverse-proxy entrypoint for OmniMed. It runs
// two servers concurrently, mirroring the AI inference service's own
// split:
//   - HTTP: image-upload diagnosis endpoints (needs multipart/form-data)
//     plus health/readiness checks. This is the only thing HTTP serves.
//   - gRPC: reserved for every other client-facing operation (auth,
//     session, medical logs) as those are built; nothing registers onto
//     it yet beyond the standard health service.
//
// Both forward diagnosis requests via gRPC to the Python AI inference
// service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	deliverygrpc "github.com/fanyicharllson/omnimed-backend/internal/gateway/delivery/grpc"
	deliveryhttp "github.com/fanyicharllson/omnimed-backend/internal/gateway/delivery/http"
	"github.com/fanyicharllson/omnimed-backend/internal/gateway/delivery/http/middleware"
	triageclient "github.com/fanyicharllson/omnimed-backend/internal/gateway/repository/grpc"
	"github.com/fanyicharllson/omnimed-backend/internal/gateway/usecase"

	"github.com/fanyicharllson/omnimed-backend/internal/config"
	applogger "github.com/fanyicharllson/omnimed-backend/internal/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := applogger.New(cfg.LogLevel, cfg.Environment)

	client, err := triageclient.NewTriageClient(
		cfg.InferenceAddr(),
		time.Duration(cfg.InferenceTimeoutS)*time.Second,
	)
	if err != nil {
		log.Error("failed to create inference client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	policy := usecase.NewDecisionPolicy(cfg.MalignantFlagThreshold, cfg.MinConfidenceFloor)
	triageUsecase := usecase.NewTriageUsecase(client, policy, log)
	handler := deliveryhttp.NewTriageHandler(triageUsecase, log)
	authenticator := middleware.NewStubAuthenticator(log)
	router := deliveryhttp.NewRouter(handler, authenticator, log)

	httpServer := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	grpcServer := deliverygrpc.NewServer(log)
	grpcListener, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("failed to listen for grpc", "port", cfg.GRPCPort, "error", err)
		os.Exit(1)
	}

	go func() {
		log.Info("gateway http listening", "port", cfg.HTTPPort, "inference_addr", cfg.InferenceAddr())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		log.Info("gateway grpc listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			log.Error("grpc server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error("http graceful shutdown failed", "error", err)
	}
	grpcServer.GracefulStop()
}
