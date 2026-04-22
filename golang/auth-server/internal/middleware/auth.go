package middleware

import (
	"context"
	"net/http"
	"strings"

	"auth-server/pkg/jwtutil"
)

type contextKey string

const (
	ctxUserID contextKey = "userID"
	ctxEmail  contextKey = "email"
	ctxScopes contextKey = "scopes"
)

// Authenticator is the JWT middleware.
type Authenticator struct {
	jwt *jwtutil.Manager
}

func NewAuthenticator(j *jwtutil.Manager) *Authenticator {
	return &Authenticator{jwt: j}
}

// Required rejects unauthenticated requests with 401.
func (a *Authenticator) Required(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := a.extractClaims(r)
		if err != nil || claims.TokenType != jwtutil.TypeAccess {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
		ctx = context.WithValue(ctx, ctxEmail, claims.Email)
		ctx = context.WithValue(ctx, ctxScopes, claims.Scopes)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Optional loads user context if a valid token is present, but never blocks.
func (a *Authenticator) Optional(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, err := a.extractClaims(r)
		if err == nil && claims.TokenType == jwtutil.TypeAccess {
			ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
			ctx = context.WithValue(ctx, ctxEmail, claims.Email)
			ctx = context.WithValue(ctx, ctxScopes, claims.Scopes)
			r = r.WithContext(ctx)
		}
		next.ServeHTTP(w, r)
	})
}

// RequireScope middleware ensures the token carries a specific scope.
func (a *Authenticator) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, _ := r.Context().Value(ctxScopes).([]string)
			for _, s := range scopes {
				if s == scope {
					next.ServeHTTP(w, r)
					return
				}
			}
			http.Error(w, `{"error":"insufficient_scope"}`, http.StatusForbidden)
		})
	}
}

// ── Context helpers ───────────────────────────────────────────────────────────

// UserIDFromCtx returns the authenticated user's ID from the request context.
func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

func EmailFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxEmail).(string)
	return v
}

func ScopesFromCtx(ctx context.Context) []string {
	v, _ := ctx.Value(ctxScopes).([]string)
	return v
}

// ── Internal ──────────────────────────────────────────────────────────────────

func (a *Authenticator) extractClaims(r *http.Request) (*jwtutil.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, jwtutil.ErrNoToken
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return nil, jwtutil.ErrNoToken
	}
	return a.jwt.Parse(parts[1])
}
