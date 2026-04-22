package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"auth-server/internal/middleware"
	"auth-server/internal/service"
)

// AuthHandler handles email/password authentication endpoints.
type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(a *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: a}
}

// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}

	user, err := h.auth.Register(r.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email_taken", "email already registered")
		case errors.Is(err, service.ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, "password_too_short", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "server_error", "registration failed")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"id":             user.ID,
		"email":          user.Email,
		"name":           user.Name,
		"email_verified": user.EmailVerified,
	})
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}

	pair, err := h.auth.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid_credentials", "email or password incorrect")
		case errors.Is(err, service.ErrAccountDisabled):
			writeError(w, http.StatusForbidden, "account_disabled", "account has been disabled")
		case errors.Is(err, service.ErrEmailNotVerified):
			writeError(w, http.StatusForbidden, "email_not_verified", "please verify your email first")
		default:
			writeError(w, http.StatusInternalServerError, "server_error", "login failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}

	pair, err := h.auth.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid_token", "refresh token invalid or expired")
		return
	}

	writeJSON(w, http.StatusOK, pair)
}

// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	_ = h.auth.Logout(r.Context(), req.RefreshToken)
	w.WriteHeader(http.StatusNoContent)
}

// POST /auth/logout-all  (requires auth middleware)
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	_ = h.auth.LogoutAll(r.Context(), userID)
	w.WriteHeader(http.StatusNoContent)
}

// POST /auth/forgot-password
func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}

	// Always respond 200 to avoid email enumeration.
	_ = h.auth.ForgotPassword(r.Context(), req.Email)
	writeJSON(w, http.StatusOK, map[string]string{
		"message": "if that email is registered, a reset link has been sent",
	})
}

// POST /auth/reset-password
func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "malformed JSON")
		return
	}

	if err := h.auth.ResetPassword(r.Context(), req.Token, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrTokenInvalid):
			writeError(w, http.StatusBadRequest, "invalid_token", "reset token invalid or expired")
		case errors.Is(err, service.ErrPasswordTooShort):
			writeError(w, http.StatusBadRequest, "password_too_short", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "server_error", "reset failed")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// GET /auth/verify-email?token=...
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		writeError(w, http.StatusBadRequest, "missing_token", "token is required")
		return
	}

	if err := h.auth.VerifyEmail(r.Context(), token); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_token", "verification token invalid or expired")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "email verified"})
}

// POST /auth/resend-verification  (requires auth middleware)
func (h *AuthHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	_ = h.auth.ResendVerification(r.Context(), userID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "if your email is unverified, a new link has been sent"})
}
