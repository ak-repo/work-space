package model

import "time"

// AuthProvider identifies how the user authenticated.
type AuthProvider string

const (
	ProviderLocal  AuthProvider = "local"
	ProviderGoogle AuthProvider = "google"
	ProviderGitHub AuthProvider = "github"
)

// User is the core identity record.
// PasswordHash is empty for social-only accounts.
type User struct {
	ID            string       `db:"id"             json:"id"`
	Email         string       `db:"email"          json:"email"`
	Name          string       `db:"name"           json:"name"`
	Picture       string       `db:"picture"        json:"picture,omitempty"`
	PasswordHash  string       `db:"password_hash"  json:"-"`
	EmailVerified bool         `db:"email_verified" json:"email_verified"`
	Provider      AuthProvider `db:"provider"       json:"provider"`
	ProviderID    string       `db:"provider_id"    json:"-"`
	Disabled      bool         `db:"disabled"       json:"-"`
	CreatedAt     time.Time    `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time    `db:"updated_at"     json:"updated_at"`
}
