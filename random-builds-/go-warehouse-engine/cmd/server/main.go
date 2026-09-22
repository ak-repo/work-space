package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"warehouse-engine/api/handler"
	"warehouse-engine/internal/config"
	"warehouse-engine/internal/db"
	"warehouse-engine/internal/repository"
	"warehouse-engine/internal/service"
	"warehouse-engine/internal/worker"

	"github.com/go-chi/chi"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()
	log.Println("connected to postgres")

	productRepo := repository.NewProductRepo(pool)
	reservationRepo := repository.NewReservationRepo(pool)
	inventorySvc := service.NewInventoryService(productRepo, reservationRepo, cfg.ReservationTTL)

	// background goroutines — stopped via ctx cancellation on SIGINT/SIGTERM
	go worker.StartTTLWatcher(ctx, inventorySvc, 30*time.Second)
	go worker.StartReorderWorker(ctx, inventorySvc.ReorderCh())

	r := chi.NewRouter()
	h := handler.New(inventorySvc)
	h.RegisterRoutes(r)

	// Serve static files
	r.Handle("/", http.FileServer(http.Dir("./static")))

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutdown signal received")
	cancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped cleanly")
}
