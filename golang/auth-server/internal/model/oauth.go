package model

import "time"

// OAuthClient represents a registered OAuth 2.0 client application.
// Seed your clients in the store at startup or via an admin API.
type OAuthClient struct {
	ID           string    `db:"id"`
	Secret       string    `db:"secret"`       // hashed with bcrypt; empty for public clients
	Name         string    `db:"name"`
	RedirectURIs []string  `db:"redirect_uris"` // allowed redirect URIs
	Scopes       []string  `db:"scopes"`        // allowed scopes
	GrantTypes   []string  `db:"grant_types"`   // "authorization_code","client_credentials","refresh_token"
	Public       bool      `db:"public"`        // PKCE-only clients (no secret)
	CreatedAt    time.Time `db:"created_at"`
}

// AuthorizationCode is a short-lived code issued after user consent.
// Exchanged for tokens at the /oauth/token endpoint.
type AuthorizationCode struct {
	Code                string    `db:"code"`
	ClientID            string    `db:"client_id"`
	UserID              string    `db:"user_id"`
	RedirectURI         string    `db:"redirect_uri"`
	Scopes              []string  `db:"scopes"`
	CodeChallenge       string    `db:"code_challenge"`        // PKCE
	CodeChallengeMethod string    `db:"code_challenge_method"` // "S256" or "plain"
	ExpiresAt           time.Time `db:"expires_at"`
	Used                bool      `db:"used"`
	CreatedAt           time.Time `db:"created_at"`
}

// OAuthToken represents an issued OAuth access/refresh token (opaque or JWT).
// When using JWT access tokens, you may not need to store them —
// keep this for refresh tokens and token introspection.
type OAuthToken struct {
	ID           string    `db:"id"`
	ClientID     string    `db:"client_id"`
	UserID       string    `db:"user_id"` // empty for client_credentials
	TokenHash    string    `db:"token_hash"`
	Scopes       []string  `db:"scopes"`
	TokenType    string    `db:"token_type"` // "access" | "refresh"
	ExpiresAt    time.Time `db:"expires_at"`
	Revoked      bool      `db:"revoked"`
	CreatedAt    time.Time `db:"created_at"`
}
