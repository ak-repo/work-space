package store

import (
	"context"
	"errors"

	"auth-server/internal/model"
)

// Sentinel errors returned by store implementations.
var (
	ErrNotFound     = errors.New("store: record not found")
	ErrDuplicate    = errors.New("store: duplicate record")
	ErrTokenExpired = errors.New("store: token expired")
	ErrTokenUsed    = errors.New("store: token already used")
	ErrTokenRevoked = errors.New("store: token revoked")
)

// UserStore manages user persistence.
type UserStore interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByProvider(ctx context.Context, provider model.AuthProvider, providerID string) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
	Delete(ctx context.Context, id string) error
}

// TokenStore manages refresh tokens and one-time email tokens.
type TokenStore interface {
	// Refresh tokens
	SaveRefreshToken(ctx context.Context, t *model.RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*model.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error

	// One-time email tokens (verify + reset password)
	SaveEmailToken(ctx context.Context, t *model.EmailToken) error
	GetEmailToken(ctx context.Context, tokenHash string) (*model.EmailToken, error)
	MarkEmailTokenUsed(ctx context.Context, tokenHash string) error
	// Invalidate all pending tokens of a type for a user (e.g. on password reset)
	RevokeEmailTokens(ctx context.Context, userID string, tokenType model.EmailTokenType) error
}

// OAuthStore manages OAuth 2.0 clients, authorization codes, and tokens.
type OAuthStore interface {
	// Clients
	SaveClient(ctx context.Context, c *model.OAuthClient) error
	GetClient(ctx context.Context, id string) (*model.OAuthClient, error)
	ListClients(ctx context.Context) ([]*model.OAuthClient, error)
	DeleteClient(ctx context.Context, id string) error

	// Authorization codes
	SaveCode(ctx context.Context, code *model.AuthorizationCode) error
	GetCode(ctx context.Context, code string) (*model.AuthorizationCode, error)
	MarkCodeUsed(ctx context.Context, code string) error

	// OAuth tokens (primarily for refresh tokens + introspection)
	SaveOAuthToken(ctx context.Context, t *model.OAuthToken) error
	GetOAuthToken(ctx context.Context, tokenHash string) (*model.OAuthToken, error)
	RevokeOAuthToken(ctx context.Context, tokenHash string) error
	RevokeClientTokens(ctx context.Context, clientID string) error
}

// Store bundles all sub-stores. Pass this struct around as a dependency.
type Store struct {
	Users  UserStore
	Tokens TokenStore
	OAuth  OAuthStore
}
