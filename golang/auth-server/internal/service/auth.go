package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"auth-server/internal/email"
	"auth-server/internal/model"
	"auth-server/internal/store"
	"auth-server/pkg/jwtutil"
	"auth-server/pkg/password"

	"github.com/google/uuid"
)

// Auth errors returned to callers (wrap or compare with errors.Is).
var (
	ErrEmailTaken         = errors.New("auth: email already registered")
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
	ErrEmailNotVerified   = errors.New("auth: email not verified")
	ErrAccountDisabled    = errors.New("auth: account disabled")
	ErrTokenInvalid       = errors.New("auth: token invalid or expired")
	ErrPasswordTooShort   = errors.New("auth: password must be at least 8 characters")
)

// AuthService handles email/password registration, login, token refresh,
// password reset, and email verification.
type AuthService struct {
	store      store.Store
	jwt        *jwtutil.Manager
	email      email.Sender
	bcryptCost int
	baseURL    string
}

func NewAuthService(s store.Store, j *jwtutil.Manager, e email.Sender, bcryptCost int, baseURL string) *AuthService {
	return &AuthService{store: s, jwt: j, email: e, bcryptCost: bcryptCost, baseURL: baseURL}
}

// Register creates a new local account and sends a verification email.
func (a *AuthService) Register(ctx context.Context, name, emailAddr, plain string) (*model.User, error) {
	if len(plain) < 8 {
		return nil, ErrPasswordTooShort
	}
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	hash, err := password.Hash(plain, a.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("auth: hash password: %w", err)
	}

	u := &model.User{
		ID:           uuid.NewString(),
		Email:        emailAddr,
		Name:         name,
		PasswordHash: hash,
		Provider:     model.ProviderLocal,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := a.store.Users.Create(ctx, u); err != nil {
		if errors.Is(err, store.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("auth: create user: %w", err)
	}

	if err := a.sendVerificationEmail(ctx, u); err != nil {
		slog.Error("auth: send verification email", slog.String("err", err.Error()))
		// Non-fatal — user can request resend.
	}

	return u, nil
}

// Login validates credentials and returns a token pair.
func (a *AuthService) Login(ctx context.Context, emailAddr, plain string) (*model.TokenPair, error) {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))

	u, err := a.store.Users.GetByEmail(ctx, emailAddr)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if u.Disabled {
		return nil, ErrAccountDisabled
	}

	if err := password.Verify(u.PasswordHash, plain); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Uncomment if you want to enforce email verification before login.
	// if !u.EmailVerified { return nil, ErrEmailNotVerified }

	return a.issueTokenPair(ctx, u)
}

// Refresh rotates the refresh token and issues a new token pair.
// The old refresh token is revoked (token rotation prevents replay attacks).
func (a *AuthService) Refresh(ctx context.Context, rawRefresh string) (*model.TokenPair, error) {
	claims, err := a.jwt.Parse(rawRefresh)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	if claims.TokenType != jwtutil.TypeRefresh {
		return nil, ErrTokenInvalid
	}

	hash := hashToken(rawRefresh)
	rt, err := a.store.Tokens.GetRefreshToken(ctx, hash)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	// Revoke old token immediately (rotation).
	_ = a.store.Tokens.RevokeRefreshToken(ctx, hash)

	u, err := a.store.Users.GetByID(ctx, rt.UserID)
	if err != nil {
		return nil, ErrTokenInvalid
	}
	if u.Disabled {
		return nil, ErrAccountDisabled
	}

	return a.issueTokenPair(ctx, u)
}

// Logout revokes the given refresh token.
func (a *AuthService) Logout(ctx context.Context, rawRefresh string) error {
	hash := hashToken(rawRefresh)
	err := a.store.Tokens.RevokeRefreshToken(ctx, hash)
	if errors.Is(err, store.ErrNotFound) {
		return nil // idempotent
	}
	return err
}

// LogoutAll revokes every refresh token for a user (e.g. on password change).
func (a *AuthService) LogoutAll(ctx context.Context, userID string) error {
	return a.store.Tokens.RevokeAllUserRefreshTokens(ctx, userID)
}

// ForgotPassword generates a password-reset token and sends it by email.
func (a *AuthService) ForgotPassword(ctx context.Context, emailAddr string) error {
	emailAddr = strings.ToLower(strings.TrimSpace(emailAddr))
	u, err := a.store.Users.GetByEmail(ctx, emailAddr)
	if err != nil {
		// Don't reveal whether the email exists.
		return nil
	}

	raw, hash, err := generateToken()
	if err != nil {
		return err
	}

	// Invalidate previous reset tokens for this user.
	_ = a.store.Tokens.RevokeEmailTokens(ctx, u.ID, model.EmailTokenResetPassword)

	et := &model.EmailToken{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		TokenHash: hash,
		Type:      model.EmailTokenResetPassword,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	if err := a.store.Tokens.SaveEmailToken(ctx, et); err != nil {
		return err
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", a.baseURL, raw)
	msg := email.ResetPasswordMessage(u.Email, resetURL)
	return a.email.Send(ctx, msg)
}

// ResetPassword validates the reset token and sets the new password.
func (a *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	hash := hashToken(rawToken)
	et, err := a.store.Tokens.GetEmailToken(ctx, hash)
	if err != nil {
		return ErrTokenInvalid
	}
	if et.Type != model.EmailTokenResetPassword {
		return ErrTokenInvalid
	}

	u, err := a.store.Users.GetByID(ctx, et.UserID)
	if err != nil {
		return ErrTokenInvalid
	}

	newHash, err := password.Hash(newPassword, a.bcryptCost)
	if err != nil {
		return err
	}
	u.PasswordHash = newHash
	u.UpdatedAt = time.Now()

	if err := a.store.Users.Update(ctx, u); err != nil {
		return err
	}

	_ = a.store.Tokens.MarkEmailTokenUsed(ctx, hash)
	// Invalidate all sessions after password change.
	_ = a.store.Tokens.RevokeAllUserRefreshTokens(ctx, u.ID)
	return nil
}

// VerifyEmail marks the user's email as verified using a token link.
func (a *AuthService) VerifyEmail(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)
	et, err := a.store.Tokens.GetEmailToken(ctx, hash)
	if err != nil {
		return ErrTokenInvalid
	}
	if et.Type != model.EmailTokenVerify {
		return ErrTokenInvalid
	}

	u, err := a.store.Users.GetByID(ctx, et.UserID)
	if err != nil {
		return ErrTokenInvalid
	}

	u.EmailVerified = true
	u.UpdatedAt = time.Now()
	if err := a.store.Users.Update(ctx, u); err != nil {
		return err
	}

	return a.store.Tokens.MarkEmailTokenUsed(ctx, hash)
}

// ResendVerification sends a new verification email.
func (a *AuthService) ResendVerification(ctx context.Context, userID string) error {
	u, err := a.store.Users.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if u.EmailVerified {
		return nil
	}
	return a.sendVerificationEmail(ctx, u)
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (a *AuthService) issueTokenPair(ctx context.Context, u *model.User) (*model.TokenPair, error) {
	access, err := a.jwt.AccessToken(u.ID, u.Email, nil)
	if err != nil {
		return nil, fmt.Errorf("auth: sign access token: %w", err)
	}

	rawRefresh, err := a.jwt.RefreshToken(u.ID)
	if err != nil {
		return nil, fmt.Errorf("auth: sign refresh token: %w", err)
	}

	rt := &model.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: time.Now().Add(a.jwt.AccessTokenTTL() * 672), // rough 7d; manager exposes this directly
		CreatedAt: time.Now(),
	}
	if err := a.store.Tokens.SaveRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("auth: save refresh token: %w", err)
	}

	return &model.TokenPair{
		AccessToken:  access,
		RefreshToken: rawRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    int(a.jwt.AccessTokenTTL().Seconds()),
	}, nil
}

func (a *AuthService) sendVerificationEmail(ctx context.Context, u *model.User) error {
	raw, hash, err := generateToken()
	if err != nil {
		return err
	}

	et := &model.EmailToken{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		TokenHash: hash,
		Type:      model.EmailTokenVerify,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := a.store.Tokens.SaveEmailToken(ctx, et); err != nil {
		return err
	}

	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s", a.baseURL, raw)
	msg := email.VerifyEmailMessage(u.Email, verifyURL)
	return a.email.Send(ctx, msg)
}

// generateToken returns (rawToken, sha256Hash, error).
// Store the hash; send the raw token by email.
func generateToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	raw = hex.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return
}

// hashToken returns the SHA-256 hex hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
