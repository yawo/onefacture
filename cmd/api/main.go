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

	"github.com/yawo/onefacture/internal/adapters/registry"
	"github.com/yawo/onefacture/internal/config"
	"github.com/yawo/onefacture/internal/events"
	"github.com/yawo/onefacture/internal/gateway"
	"github.com/yawo/onefacture/internal/gateway/middleware"
	"github.com/yawo/onefacture/internal/storage"
	"github.com/yawo/onefacture/internal/validation"
	"github.com/yawo/onefacture/internal/webhooks"
	"github.com/yawo/onefacture/internal/workers"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	store, err := storage.New(ctx, cfg.Database)
	if err != nil {
		logger.Error("storage init", "err", err)
		os.Exit(1)
	}
	defer store.Close()

	bus, err := events.New(ctx, cfg.Redis)
	if err != nil {
		logger.Error("event bus init", "err", err)
		os.Exit(1)
	}
	defer bus.Close()

	validator := validation.NewClient(cfg.Sidecar)
	reg := registry.NewDefault(logger)

	deliverer := webhooks.NewDeliverer(logger, bus, store)
	go deliverer.Run(ctx)

	poller := workers.NewStatusPoller(logger, store, reg, bus)
	go poller.Run(ctx)

	srv := gateway.New(gateway.Options{
		Config:    cfg,
		Logger:    logger,
		Store:     store,
		Validator: validator,
		Registry:  reg,
		Events:    bus,
		AuthN:     middleware.NewAPIKeyAuth(store),
	})

	httpSrv := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           srv.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("server listening", "addr", cfg.HTTP.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown requested")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", "err", err)
	}
}
