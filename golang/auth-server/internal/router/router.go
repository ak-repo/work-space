package router

import (
	"net/http"

	"auth-server/internal/handler"
	"auth-server/internal/middleware"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// New builds and returns the fully wired chi router.
func New(
	authH *handler.AuthHandler,
	socialH *handler.SocialHandler,
	oauthH *handler.OAuthHandler,
	authn *middleware.Authenticator,
) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ────────────────────────────────────────────────────
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.StripSlashes)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // restrict in production
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Discovery ────────────────────────────────────────────────────────────
	r.Get("/.well-known/oauth-authorization-server", oauthH.Discovery)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// ── Email / Password auth ────────────────────────────────────────────────
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/logout", authH.Logout)
		r.Post("/forgot-password", authH.ForgotPassword)
		r.Post("/reset-password", authH.ResetPassword)
		r.Get("/verify-email", authH.VerifyEmail)

		// Authenticated routes
		r.Group(func(r chi.Router) {
			r.Use(authn.Required)
			r.Post("/logout-all", authH.LogoutAll)
			r.Post("/resend-verification", authH.ResendVerification)
		})

		// ── Social login ─────────────────────────────────────────────────────
		r.Route("/social/{provider}", func(r chi.Router) {
			r.Get("/", socialH.Redirect)
			r.Get("/callback", socialH.Callback)
		})
	})

	// ── OAuth 2.0 authorization server ───────────────────────────────────────
	r.Route("/oauth", func(r chi.Router) {
		// Authorization endpoint — user must be authenticated
		r.With(authn.Required).Get("/authorize", oauthH.Authorize)

		// Token, revoke, introspect are open (client authenticates in body/header)
		r.Post("/token", oauthH.Token)
		r.Post("/revoke", oauthH.Revoke)

		// Introspect — in production, protect this so only resource servers can call it
		r.Post("/introspect", oauthH.Introspect)

		// OIDC userinfo — requires user access token
		r.With(authn.Required).Get("/userinfo", oauthH.UserInfo)
	})

	return r
}
