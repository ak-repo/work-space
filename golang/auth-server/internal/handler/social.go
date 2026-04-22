package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"auth-server/internal/model"
	"auth-server/internal/service"

	"github.com/go-chi/chi/v5"
)

// SocialHandler manages social OAuth2 login (Google, GitHub).
type SocialHandler struct {
	social      *service.SocialService
	frontendURL string // where to redirect after successful login
}

func NewSocialHandler(s *service.SocialService, frontendURL string) *SocialHandler {
	return &SocialHandler{social: s, frontendURL: frontendURL}
}

// GET /auth/social/{provider}
// Redirects the browser to the provider's OAuth consent page.
func (h *SocialHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	provider := model.AuthProvider(chi.URLParam(r, "provider"))

	// Generate CSRF state token and store in a short-lived cookie.
	state, err := randomState()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", "could not generate state")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600, // 10 minutes
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	authURL, err := h.social.AuthURL(provider, state)
	if err != nil {
		if errors.Is(err, service.ErrProviderNotConfigured) {
			writeError(w, http.StatusBadRequest, "provider_not_configured", string(provider)+" is not enabled")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", "failed to build auth URL")
		return
	}

	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

// GET /auth/social/{provider}/callback
// Exchanges the authorization code, upserts the user, returns tokens.
func (h *SocialHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := model.AuthProvider(chi.URLParam(r, "provider"))

	// Validate CSRF state.
	stateCookie, err := r.Cookie("oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusBadRequest, "state_mismatch", "CSRF state mismatch")
		return
	}
	// Clear the state cookie.
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", MaxAge: -1, Path: "/"})

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "missing_code", "authorization code not present")
		return
	}

	_, pair, err := h.social.HandleCallback(r.Context(), provider, code)
	if err != nil {
		if errors.Is(err, service.ErrProviderNotConfigured) {
			writeError(w, http.StatusBadRequest, "provider_not_configured", string(provider)+" is not enabled")
			return
		}
		writeError(w, http.StatusUnauthorized, "callback_failed", "social login failed: "+err.Error())
		return
	}

	// Option A: Redirect the browser to the frontend with tokens in the URL
	// (suitable for SPA; use fragment # to keep tokens out of server logs).
	redirectURL := h.frontendURL +
		"?access_token=" + pair.AccessToken +
		"&refresh_token=" + pair.RefreshToken

	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)

	// Option B: Return JSON (suitable for mobile / server-side apps).
	// Comment out the redirect above and uncomment this:
	// writeJSON(w, http.StatusOK, pair)
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
