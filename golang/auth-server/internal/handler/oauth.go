package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"auth-server/internal/middleware"
	"auth-server/internal/service"
)

// OAuthHandler implements the OAuth 2.0 authorization server endpoints.
//
// Endpoints:
//
//	GET  /oauth/authorize       – authorization endpoint (requires user auth)
//	POST /oauth/token           – token endpoint
//	POST /oauth/revoke          – token revocation (RFC 7009)
//	POST /oauth/introspect      – token introspection (RFC 7662)
//	GET  /oauth/userinfo        – OIDC userinfo endpoint (requires user auth)
//	GET  /.well-known/oauth-authorization-server – discovery metadata
type OAuthHandler struct {
	oauth   *service.OAuthService
	baseURL string
}

func NewOAuthHandler(o *service.OAuthService, baseURL string) *OAuthHandler {
	return &OAuthHandler{oauth: o, baseURL: baseURL}
}

// GET /oauth/authorize
// The user MUST be authenticated (via Required middleware) before reaching this.
// Typical flow:
//  1. Client redirects user to GET /oauth/authorize?client_id=...
//  2. If user is not logged in, middleware returns 401 → SPA redirects to login.
//  3. After login, SPA calls this endpoint again with the access token in the header.
//  4. This handler issues a code and redirects to redirect_uri.
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	userID := middleware.UserIDFromCtx(r.Context())

	redirectURL, err := h.oauth.Authorize(
		r.Context(),
		userID,
		q.Get("client_id"),
		q.Get("redirect_uri"),
		q.Get("response_type"),
		q.Get("scope"),
		q.Get("state"),
		q.Get("code_challenge"),
		q.Get("code_challenge_method"),
	)
	if err != nil {
		writeError(w, http.StatusBadRequest, "authorize_failed", err.Error())
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// POST /oauth/token
// Supports: authorization_code, client_credentials, refresh_token
func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}

	// Support both Basic auth and body params for client credentials.
	clientID, clientSecret, ok := r.BasicAuth()
	if !ok {
		clientID = r.FormValue("client_id")
		clientSecret = r.FormValue("client_secret")
	}

	req := service.TokenRequest{
		GrantType:    r.FormValue("grant_type"),
		Code:         r.FormValue("code"),
		RedirectURI:  r.FormValue("redirect_uri"),
		RefreshToken: r.FormValue("refresh_token"),
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       r.FormValue("scope"),
		CodeVerifier: r.FormValue("code_verifier"),
	}

	resp, err := h.oauth.Token(r.Context(), req)
	if err != nil {
		code := http.StatusBadRequest
		errCode := "invalid_request"
		switch {
		case errors.Is(err, service.ErrClientNotFound), errors.Is(err, service.ErrClientSecretInvalid):
			code = http.StatusUnauthorized
			errCode = "invalid_client"
		case errors.Is(err, service.ErrCodeInvalid):
			errCode = "invalid_grant"
		case errors.Is(err, service.ErrUnsupportedGrantType):
			errCode = "unsupported_grant_type"
		case errors.Is(err, service.ErrScopeNotAllowed):
			errCode = "invalid_scope"
		case errors.Is(err, service.ErrGrantTypeNotAllowed):
			errCode = "unauthorized_client"
		}
		writeOAuthError(w, code, errCode, err.Error())
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	writeJSON(w, http.StatusOK, resp)
}

// POST /oauth/revoke
func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}
	_ = h.oauth.Revoke(r.Context(), r.FormValue("token"))
	// RFC 7009: always 200 even if token not found.
	w.WriteHeader(http.StatusOK)
}

// POST /oauth/introspect  (should be protected — only resource servers call this)
func (h *OAuthHandler) Introspect(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, http.StatusBadRequest, "invalid_request", "cannot parse form")
		return
	}
	resp := h.oauth.Introspect(r.Context(), r.FormValue("token"))
	writeJSON(w, http.StatusOK, resp)
}

// GET /oauth/userinfo  (requires auth middleware)
func (h *OAuthHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	resp, err := h.oauth.UserInfo(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// GET /.well-known/oauth-authorization-server
// OAuth 2.0 Authorization Server Metadata (RFC 8414).
func (h *OAuthHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	base := h.baseURL
	writeJSON(w, http.StatusOK, map[string]any{
		"issuer":                                base,
		"authorization_endpoint":                base + "/oauth/authorize",
		"token_endpoint":                        base + "/oauth/token",
		"revocation_endpoint":                   base + "/oauth/revoke",
		"introspection_endpoint":                base + "/oauth/introspect",
		"userinfo_endpoint":                     base + "/oauth/userinfo",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "client_credentials", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
	})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// writeOAuthError writes a JSON error in RFC 6749 format.
func writeOAuthError(w http.ResponseWriter, status int, errCode, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             errCode,
		"error_description": desc,
	})
}

// tokenFromBearer is a helper to extract a bearer token from Authorization header.
func tokenFromBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return h[7:]
	}
	return ""
}
