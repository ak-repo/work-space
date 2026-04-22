package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"auth-server/internal/config"
	"auth-server/internal/email"
	"auth-server/internal/handler"
	"auth-server/internal/middleware"
	"auth-server/internal/router"
	"auth-server/internal/service"
	"auth-server/internal/store/memory"
	"auth-server/pkg/jwtutil"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		slog.Warn("error loading .env file", slog.String("err", err.Error()))
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()

	// ── Store ─────────────────────────────────────────────────────────────────
	// Swap memory.New() with your Postgres/Redis implementation.
	st := memory.New()

	// ── JWT ───────────────────────────────────────────────────────────────────
	jwtMgr := jwtutil.New(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// ── Email sender ──────────────────────────────────────────────────────────
	// Swap email.ConsoleSender{} with your SMTP/SendGrid implementation.
	var emailSender email.Sender = email.ConsoleSender{}

	// ── Services ──────────────────────────────────────────────────────────────
	authSvc := service.NewAuthService(st, jwtMgr, emailSender, cfg.BCryptCost, cfg.BaseURL)
	socialSvc := service.NewSocialService(
		st, jwtMgr,
		cfg.GoogleClientID, cfg.GoogleClientSecret,
		cfg.GitHubClientID, cfg.GitHubClientSecret,
		cfg.BaseURL,
	)
	oauthSvc := service.NewOAuthService(st, jwtMgr, cfg.BaseURL)

	// ── Handlers ──────────────────────────────────────────────────────────────
	authH := handler.NewAuthHandler(authSvc)
	socialH := handler.NewSocialHandler(socialSvc, cfg.FrontendURL)
	oauthH := handler.NewOAuthHandler(oauthSvc, cfg.BaseURL)

	// ── Middleware ────────────────────────────────────────────────────────────
	authn := middleware.NewAuthenticator(jwtMgr)

	// ── Router ────────────────────────────────────────────────────────────────
	h := router.New(authH, socialH, oauthH, authn)

	// ── HTTP server ───────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("auth-server starting", slog.String("addr", srv.Addr), slog.String("base_url", cfg.BaseURL))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", slog.String("err", err.Error()))
	}
	fmt.Println("server stopped")
}
