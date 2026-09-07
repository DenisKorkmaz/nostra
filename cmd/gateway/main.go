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

	"github.com/DenisKorkmaz/nostra/internal/gateway"
	"github.com/DenisKorkmaz/nostra/internal/platform/config"
	"github.com/DenisKorkmaz/nostra/internal/platform/logging"
	"github.com/DenisKorkmaz/nostra/internal/platform/postgres"
)

var version = "dev"

const readHeaderTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("gateway stopped with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.IsProduction())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           gateway.NewHandler(pool, logger, version),
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serverErr := make(chan error, 1)

	go func() {
		logger.Info("gateway listening",
			slog.String("addr", cfg.HTTPAddr),
			slog.String("env", cfg.Env),
			slog.String("version", version),
		)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		logger.Info("shutdown requested")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("gateway stopped cleanly")

	return nil
}
