package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"example.com/server-monitor/internal/config"
	"example.com/server-monitor/internal/httpapi"
	"example.com/server-monitor/internal/monitor"
	"example.com/server-monitor/internal/webui"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	db, configured, err := openDatabase(cfg)
	if err != nil {
		log.Fatalf("configure database: %v", err)
	}
	if db != nil {
		defer db.Close()
	}

	systemCollector := monitor.NewSystemCollector(cfg.SystemMetricInterval)
	systemCollector.Start(ctx)

	databaseMonitor := monitor.NewDatabaseMonitor(db, configured, cfg.DBPingTimeout)
	monitorService := monitor.NewService(systemCollector, databaseMonitor)
	handler := httpapi.NewHandler(monitorService)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpapi.NewRouter(handler, webui.Handler()),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		displayAddr := cfg.Addr
		if strings.HasPrefix(displayAddr, ":") {
			displayAddr = "localhost" + displayAddr
		}
		log.Printf("monitor dashboard listening on http://%s", displayAddr)
		if configured {
			log.Printf("postgres monitoring enabled")
		} else {
			log.Printf("DATABASE_URL is empty; database monitoring is disabled")
		}

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutdown requested")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func openDatabase(cfg config.Config) (*sql.DB, bool, error) {
	if cfg.DatabaseURL == "" {
		return nil, false, nil
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, true, err
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	return db, true, nil
}
