package model

import "time"

// RefreshToken is a long-lived token stored server-side.
// On every refresh, the old token is rotated (revoked + new one issued).
type RefreshToken struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	TokenHash string    `db:"token_hash"` // store hash, not raw token
	ExpiresAt time.Time `db:"expires_at"`
	Revoked   bool      `db:"revoked"`
	CreatedAt time.Time `db:"created_at"`
}

// EmailTokenType describes the purpose of a one-time email token.
type EmailTokenType string

const (
	EmailTokenVerify        EmailTokenType = "verify_email"
	EmailTokenResetPassword EmailTokenType = "reset_password"
)

// EmailToken is a single-use token sent via email.
type EmailToken struct {
	ID        string         `db:"id"`
	UserID    string         `db:"user_id"`
	TokenHash string         `db:"token_hash"` // store hash, not raw token
	Type      EmailTokenType `db:"type"`
	ExpiresAt time.Time      `db:"expires_at"`
	UsedAt    *time.Time     `db:"used_at"` // nil = not yet used
	CreatedAt time.Time      `db:"created_at"`
}

// TokenPair is the response returned after a successful login or refresh.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`   // always "Bearer"
	ExpiresIn    int    `json:"expires_in"`   // seconds until access token expires
}
