package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/jev-ai-decision-center/internal/config"
	"github.com/example/jev-ai-decision-center/internal/decision"
	"github.com/example/jev-ai-decision-center/internal/httpapi"
	"github.com/example/jev-ai-decision-center/internal/policy"
	"github.com/example/jev-ai-decision-center/internal/provider/jev"
	"github.com/example/jev-ai-decision-center/internal/store"
	"github.com/example/jev-ai-decision-center/internal/stream"
)

//go:embed web/*
var webFS embed.FS

func main() {
	cfg := config.Load()
	ctx := context.Background()
	st, err := store.Open(ctx, cfg.DataFile)
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	var p decision.Provider
	if cfg.Jev.Mode == "live" {
		p = &jev.Client{BaseURL: cfg.Jev.BaseURL, APIKey: cfg.Jev.APIKey, Client: &http.Client{Timeout: cfg.Jev.Timeout}}
	} else {
		p = jev.Mock{Model: "mock-jev"}
	}
	sub, _ := fs.Sub(webFS, "web")
	app := &httpapi.Server{Store: st, Provider: p, Policy: policy.Engine{Cfg: cfg.Policy}, Broker: stream.New(), Model: cfg.Jev.Model, Timeout: cfg.Jev.Timeout, Web: sub}
	srv := &http.Server{Addr: cfg.HTTPAddr, Handler: app.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("Jev Decision Center on http://localhost%s (mode=%s)", cfg.HTTPAddr, cfg.Jev.Mode)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(cctx)
	fmt.Println("stopped")
}
