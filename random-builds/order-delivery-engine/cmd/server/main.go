package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ak-repo/order-delivery-engine/internal/config"
	"github.com/ak-repo/order-delivery-engine/internal/handlers"
	"github.com/ak-repo/order-delivery-engine/internal/observability"
	"github.com/ak-repo/order-delivery-engine/internal/repository"
	"github.com/ak-repo/order-delivery-engine/internal/router"
	"github.com/ak-repo/order-delivery-engine/internal/service"
	"github.com/ak-repo/order-delivery-engine/internal/websocket"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", slog.Any("error", err))
		os.Exit(1)
	}
	logger := observability.NewLogger(cfg.Observability, cfg.Environment, os.Stdout)
	slog.SetDefault(logger)

	db, err := repository.Connect(cfg.Database, logger, cfg.Observability)
	if err != nil {
		logger.Error("db connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	driverRepo := repository.NewDriverRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	hub := websocket.NewHub(logger, cfg.Observability)

	assignSvc := service.NewAssignmentService(driverRepo, orderRepo, hub, cfg.Assignment, logger, cfg.Observability)
	orderSvc := service.NewOrderService(orderRepo, driverRepo, assignSvc, hub, cfg.Defaults, logger, cfg.Observability)

	driverHandler := handlers.NewDriverHandler(driverRepo, orderRepo, hub, cfg.Defaults, logger, cfg.Observability)
	orderHandler := handlers.NewOrderHandler(orderSvc, orderRepo, assignSvc, logger)
	sseHandler := handlers.NewSSEHandler(hub, cfg.SSE, logger, cfg.Observability)

	r := router.New(driverHandler, orderHandler, sseHandler, logger, cfg.Observability)

	addr := cfg.Server.Host
	if addr == "" {
		addr = ":" + cfg.Server.Port
	} else {
		addr = addr + ":" + cfg.Server.Port
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server started", slog.String("addr", addr), slog.String("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("server shutdown started")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", slog.Any("error", err))
	}
	logger.Info("server stopped")
}
