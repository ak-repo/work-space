// Package jwtutil wraps github.com/golang-jwt/jwt with typed claims.
//
// Algorithm: HS256 by default.
// To switch to RS256 (recommended for production):
//   1. Load *rsa.PrivateKey in Config (e.g. from PEM file / secret manager).
//   2. Replace jwt.SigningMethodHS256 with jwt.SigningMethodRS256.
//   3. Pass the private key to Sign and the public key to Parse.
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrNoToken is returned when no Bearer token is present in the request.
var ErrNoToken = errors.New("jwtutil: no token present")

// TokenType distinguishes access from refresh JWTs so they cannot be
// accidentally swapped (defense-in-depth on top of TTL differences).
type TokenType string

const (
	TypeAccess  TokenType = "access"
	TypeRefresh TokenType = "refresh"
)

// Claims is the full set of fields embedded in every JWT.
type Claims struct {
	jwt.RegisteredClaims
	Email     string    `json:"email,omitempty"`
	Scopes    []string  `json:"scopes,omitempty"`
	TokenType TokenType `json:"typ"`
}

// Manager holds the signing secret and default TTLs.
type Manager struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func New(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secret:          []byte(secret),
		accessTokenTTL:  accessTTL,
		refreshTokenTTL: refreshTTL,
	}
}

// AccessToken generates a signed access JWT.
func (m *Manager) AccessToken(userID, email string, scopes []string) (string, error) {
	return m.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Email:     email,
		Scopes:    scopes,
		TokenType: TypeAccess,
	})
}

// RefreshToken generates a signed refresh JWT.
// The raw token string is also stored (hashed) in the token store for
// server-side revocation.
func (m *Manager) RefreshToken(userID string) (string, error) {
	return m.sign(Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		TokenType: TypeRefresh,
	})
}

// Parse validates the token signature and returns Claims.
// It does NOT check the token store for revocation — callers must do that.
func (m *Manager) Parse(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("jwtutil: unexpected signing method")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("jwtutil: invalid token claims")
	}
	return claims, nil
}

// AccessTokenTTL returns the configured access token lifetime.
func (m *Manager) AccessTokenTTL() time.Duration { return m.accessTokenTTL }

func (m *Manager) sign(c Claims) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}
