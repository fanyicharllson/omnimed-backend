// Command gateway is the reverse-proxy entrypoint for OmniMed: it
// accepts image uploads over HTTP and forwards them via gRPC to the
// Python AI inference service.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	triageUsecase := usecase.NewTriageUsecase(client, log)
	handler := deliveryhttp.NewTriageHandler(triageUsecase, log)
	authenticator := middleware.NewStubAuthenticator(log)
	router := deliveryhttp.NewRouter(handler, authenticator, log)

	server := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		log.Info("gateway listening", "port", cfg.HTTPPort, "inference_addr", cfg.InferenceAddr())
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("shutting down gateway")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
}
