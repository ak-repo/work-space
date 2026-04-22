package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"auth-server/internal/model"
	"auth-server/internal/store"
	"auth-server/pkg/jwtutil"
	"auth-server/pkg/password"

	"github.com/google/uuid"
)

// OAuth errors.
var (
	ErrClientNotFound       = errors.New("oauth: client not found")
	ErrClientSecretInvalid  = errors.New("oauth: invalid client secret")
	ErrRedirectURIInvalid   = errors.New("oauth: redirect_uri not allowed")
	ErrGrantTypeNotAllowed  = errors.New("oauth: grant_type not allowed for client")
	ErrScopeNotAllowed      = errors.New("oauth: scope not allowed for client")
	ErrCodeInvalid          = errors.New("oauth: code invalid or expired")
	ErrCodeVerifierInvalid  = errors.New("oauth: code_verifier does not match challenge")
	ErrUnsupportedGrantType = errors.New("oauth: unsupported grant_type")
)

// OAuthService is the authorization server logic (RFC 6749 + PKCE RFC 7636).
type OAuthService struct {
	store   store.Store
	jwt     *jwtutil.Manager
	baseURL string
}

func NewOAuthService(s store.Store, j *jwtutil.Manager, baseURL string) *OAuthService {
	return &OAuthService{store: s, jwt: j, baseURL: baseURL}
}

// ── Authorization endpoint ────────────────────────────────────────────────────

// Authorize issues an authorization code after the user has consented.
// Call this after verifying the user's identity (via middleware or session).
//
//	responseType: "code" (only authorization_code flow supported)
//	scopes: requested scopes (space-separated string)
//	Returns the redirect URI with ?code=...&state=...
func (o *OAuthService) Authorize(
	ctx context.Context,
	userID, clientID, redirectURI, responseType, scopeStr, state,
	codeChallenge, codeChallengeMethod string,
) (string, error) {
	if responseType != "code" {
		return "", errors.New("oauth: only response_type=code is supported")
	}

	client, err := o.validateClient(ctx, clientID)
	if err != nil {
		return "", err
	}
	if err := o.validateRedirectURI(client, redirectURI); err != nil {
		return "", err
	}
	if !hasGrantType(client, "authorization_code") {
		return "", ErrGrantTypeNotAllowed
	}

	scopes := parseScopes(scopeStr)
	if err := o.validateScopes(client, scopes); err != nil {
		return "", err
	}

	rawCode := uuid.NewString()
	code := &model.AuthorizationCode{
		Code:                rawCode,
		ClientID:            clientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scopes:              scopes,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		CreatedAt:           time.Now(),
	}
	if err := o.store.OAuth.SaveCode(ctx, code); err != nil {
		return "", err
	}

	redirect := redirectURI + "?code=" + rawCode
	if state != "" {
		redirect += "&state=" + state
	}
	return redirect, nil
}

// ── Token endpoint ────────────────────────────────────────────────────────────

// TokenRequest carries the parameters from POST /oauth/token.
type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	RefreshToken string
	ClientID     string
	ClientSecret string
	Scopes       string
	CodeVerifier string // PKCE
}

// TokenResponse mirrors the RFC 6749 response body.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// Token processes the /oauth/token endpoint.
func (o *OAuthService) Token(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return o.authCodeGrant(ctx, req)
	case "client_credentials":
		return o.clientCredentialsGrant(ctx, req)
	case "refresh_token":
		return o.refreshTokenGrant(ctx, req)
	default:
		return nil, ErrUnsupportedGrantType
	}
}

func (o *OAuthService) authCodeGrant(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	client, err := o.authenticateClient(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !hasGrantType(client, "authorization_code") {
		return nil, ErrGrantTypeNotAllowed
	}

	code, err := o.store.OAuth.GetCode(ctx, req.Code)
	if err != nil {
		return nil, ErrCodeInvalid
	}
	if code.ClientID != client.ID {
		_ = o.store.OAuth.MarkCodeUsed(ctx, req.Code) // consume on mismatch
		return nil, ErrCodeInvalid
	}
	if code.RedirectURI != req.RedirectURI {
		return nil, ErrRedirectURIInvalid
	}

	// PKCE verification.
	if code.CodeChallenge != "" {
		if err := verifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, req.CodeVerifier); err != nil {
			return nil, err
		}
	}

	_ = o.store.OAuth.MarkCodeUsed(ctx, req.Code)

	user, err := o.store.Users.GetByID(ctx, code.UserID)
	if err != nil {
		return nil, fmt.Errorf("oauth: load user: %w", err)
	}

	return o.buildTokenResponse(ctx, client, user, code.Scopes)
}

func (o *OAuthService) clientCredentialsGrant(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	client, err := o.authenticateClient(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !hasGrantType(client, "client_credentials") {
		return nil, ErrGrantTypeNotAllowed
	}

	scopes := parseScopes(req.Scopes)
	if err := o.validateScopes(client, scopes); err != nil {
		return nil, err
	}

	// No user — access token Subject = clientID.
	access, err := o.jwt.AccessToken(client.ID, "", scopes)
	if err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(o.jwt.AccessTokenTTL().Seconds()),
		Scope:       strings.Join(scopes, " "),
	}, nil
}

func (o *OAuthService) refreshTokenGrant(ctx context.Context, req TokenRequest) (*TokenResponse, error) {
	client, err := o.authenticateClient(ctx, req.ClientID, req.ClientSecret)
	if err != nil {
		return nil, err
	}
	if !hasGrantType(client, "refresh_token") {
		return nil, ErrGrantTypeNotAllowed
	}

	hash := hashToken(req.RefreshToken)
	oauthToken, err := o.store.OAuth.GetOAuthToken(ctx, hash)
	if err != nil || oauthToken.ClientID != client.ID || oauthToken.TokenType != "refresh" {
		return nil, ErrTokenInvalid
	}

	// Rotate: revoke old, issue new.
	_ = o.store.OAuth.RevokeOAuthToken(ctx, hash)

	user, err := o.store.Users.GetByID(ctx, oauthToken.UserID)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	return o.buildTokenResponse(ctx, client, user, oauthToken.Scopes)
}

// ── Introspection (RFC 7662) ──────────────────────────────────────────────────

type IntrospectResponse struct {
	Active    bool   `json:"active"`
	Sub       string `json:"sub,omitempty"`
	ClientID  string `json:"client_id,omitempty"`
	Username  string `json:"username,omitempty"`
	Scope     string `json:"scope,omitempty"`
	ExpiresAt int64  `json:"exp,omitempty"`
	TokenType string `json:"token_type,omitempty"`
}

func (o *OAuthService) Introspect(ctx context.Context, rawToken string) *IntrospectResponse {
	claims, err := o.jwt.Parse(rawToken)
	if err != nil {
		return &IntrospectResponse{Active: false}
	}
	u, _ := o.store.Users.GetByID(ctx, claims.Subject)
	resp := &IntrospectResponse{
		Active:    true,
		Sub:       claims.Subject,
		Scope:     strings.Join(claims.Scopes, " "),
		ExpiresAt: claims.ExpiresAt.Unix(),
		TokenType: "Bearer",
	}
	if u != nil {
		resp.Username = u.Email
	}
	return resp
}

// ── Revoke (RFC 7009) ─────────────────────────────────────────────────────────

// Revoke revokes an OAuth refresh token.
func (o *OAuthService) Revoke(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)
	err := o.store.OAuth.RevokeOAuthToken(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return nil // idempotent
	}
	return err
}

// ── UserInfo (OIDC) ───────────────────────────────────────────────────────────

type UserInfoResponse struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Picture       string `json:"picture,omitempty"`
}

func (o *OAuthService) UserInfo(ctx context.Context, userID string) (*UserInfoResponse, error) {
	u, err := o.store.Users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserInfoResponse{
		Sub:           u.ID,
		Name:          u.Name,
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Picture:       u.Picture,
	}, nil
}

// ── Client management ─────────────────────────────────────────────────────────

// RegisterClient creates a new OAuth2 client.
// Call from an admin endpoint or a seed function at startup.
func (o *OAuthService) RegisterClient(ctx context.Context, c *model.OAuthClient) error {
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	c.CreatedAt = time.Now()
	return o.store.OAuth.SaveClient(ctx, c)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (o *OAuthService) validateClient(ctx context.Context, id string) (*model.OAuthClient, error) {
	c, err := o.store.OAuth.GetClient(ctx, id)
	if err != nil {
		return nil, ErrClientNotFound
	}
	return c, nil
}

func (o *OAuthService) authenticateClient(ctx context.Context, id, secret string) (*model.OAuthClient, error) {
	c, err := o.validateClient(ctx, id)
	if err != nil {
		return nil, err
	}
	if !c.Public {
		if err := password.Verify(c.Secret, secret); err != nil {
			return nil, ErrClientSecretInvalid
		}
	}
	return c, nil
}

func (o *OAuthService) validateRedirectURI(c *model.OAuthClient, uri string) error {
	for _, u := range c.RedirectURIs {
		if u == uri {
			return nil
		}
	}
	return ErrRedirectURIInvalid
}

func (o *OAuthService) validateScopes(c *model.OAuthClient, requested []string) error {
	allowed := make(map[string]bool, len(c.Scopes))
	for _, s := range c.Scopes {
		allowed[s] = true
	}
	for _, s := range requested {
		if !allowed[s] {
			return fmt.Errorf("%w: %s", ErrScopeNotAllowed, s)
		}
	}
	return nil
}

func (o *OAuthService) buildTokenResponse(
	ctx context.Context,
	client *model.OAuthClient,
	user *model.User,
	scopes []string,
) (*TokenResponse, error) {
	access, err := o.jwt.AccessToken(user.ID, user.Email, scopes)
	if err != nil {
		return nil, err
	}

	resp := &TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int(o.jwt.AccessTokenTTL().Seconds()),
		Scope:       strings.Join(scopes, " "),
	}

	// Issue refresh token if client supports it.
	if hasGrantType(client, "refresh_token") {
		rawRefresh := uuid.NewString()
		hash := hashToken(rawRefresh)
		rt := &model.OAuthToken{
			ID:        uuid.NewString(),
			ClientID:  client.ID,
			UserID:    user.ID,
			TokenHash: hash,
			Scopes:    scopes,
			TokenType: "refresh",
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
			CreatedAt: time.Now(),
		}
		if err := o.store.OAuth.SaveOAuthToken(ctx, rt); err != nil {
			return nil, err
		}
		resp.RefreshToken = rawRefresh
	}

	return resp, nil
}

func hasGrantType(c *model.OAuthClient, gt string) bool {
	for _, g := range c.GrantTypes {
		if g == gt {
			return true
		}
	}
	return false
}

func parseScopes(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

// verifyPKCE checks the code_verifier against the stored code_challenge.
func verifyPKCE(challenge, method, verifier string) error {
	if verifier == "" {
		return ErrCodeVerifierInvalid
	}
	switch method {
	case "S256":
		h := sha256.Sum256([]byte(verifier))
		computed := base64.RawURLEncoding.EncodeToString(h[:])
		if computed != challenge {
			return ErrCodeVerifierInvalid
		}
	case "plain", "":
		if verifier != challenge {
			return ErrCodeVerifierInvalid
		}
	default:
		return fmt.Errorf("oauth: unsupported code_challenge_method: %s", method)
	}
	return nil
}
