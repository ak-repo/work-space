package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"auth-server/internal/model"
	"auth-server/internal/store"
	"auth-server/pkg/jwtutil"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

var ErrProviderNotConfigured = errors.New("social: provider not configured")

// SocialProvider holds the OAuth2 config and a userinfo fetcher for one provider.
type SocialProvider struct {
	cfg          *oauth2.Config
	fetchProfile func(ctx context.Context, token *oauth2.Token) (*socialProfile, error)
}

// socialProfile is the normalized user info returned by any provider.
type socialProfile struct {
	ID      string
	Email   string
	Name    string
	Picture string
}

// SocialService manages OAuth2 social login.
// Add new providers by extending the providers map.
type SocialService struct {
	providers map[model.AuthProvider]*SocialProvider
	store     store.Store
	jwt       *jwtutil.Manager
}

func NewSocialService(
	store store.Store,
	jwt *jwtutil.Manager,
	googleClientID, googleClientSecret,
	githubClientID, githubClientSecret,
	baseURL string,
) *SocialService {
	svc := &SocialService{
		providers: make(map[model.AuthProvider]*SocialProvider),
		store:     store,
		jwt:       jwt,
	}

	if googleClientID != "" {
		svc.providers[model.ProviderGoogle] = &SocialProvider{
			cfg: &oauth2.Config{
				ClientID:     googleClientID,
				ClientSecret: googleClientSecret,
				RedirectURL:  baseURL + "/auth/social/google/callback",
				Scopes:       []string{"openid", "profile", "email"},
				Endpoint:     google.Endpoint,
			},
			fetchProfile: fetchGoogleProfile,
		}
	}

	if githubClientID != "" {
		svc.providers[model.ProviderGitHub] = &SocialProvider{
			cfg: &oauth2.Config{
				ClientID:     githubClientID,
				ClientSecret: githubClientSecret,
				RedirectURL:  baseURL + "/auth/social/github/callback",
				Scopes:       []string{"read:user", "user:email"},
				Endpoint:     github.Endpoint,
			},
			fetchProfile: fetchGitHubProfile,
		}
	}

	return svc
}

// AuthURL returns the provider's OAuth2 consent URL.
// state should be a random, CSRF-protected value stored in a short-lived cookie.
func (s *SocialService) AuthURL(provider model.AuthProvider, state string) (string, error) {
	p, ok := s.providers[provider]
	if !ok {
		return "", ErrProviderNotConfigured
	}
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline), nil
}

// HandleCallback exchanges the authorization code for tokens, fetches the
// user profile, and upserts the user record. Returns a TokenPair on success.
func (s *SocialService) HandleCallback(
	ctx context.Context,
	provider model.AuthProvider,
	code string,
) (*model.User, *model.TokenPair, error) {
	p, ok := s.providers[provider]
	if !ok {
		return nil, nil, ErrProviderNotConfigured
	}

	token, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("social: exchange code: %w", err)
	}

	profile, err := p.fetchProfile(ctx, token)
	if err != nil {
		return nil, nil, fmt.Errorf("social: fetch profile: %w", err)
	}

	user, err := s.upsertUser(ctx, provider, profile)
	if err != nil {
		return nil, nil, err
	}

	// Reuse AuthService helper to issue JWT pair.
	authSvc := &AuthService{store: s.store, jwt: s.jwt}
	pair, err := authSvc.issueTokenPair(ctx, user)
	if err != nil {
		return nil, nil, err
	}

	return user, pair, nil
}

// upsertUser finds or creates a user record for the social profile.
// If the email already exists as a local account, it links the social provider.
func (s *SocialService) upsertUser(ctx context.Context, provider model.AuthProvider, p *socialProfile) (*model.User, error) {
	// 1. Try by provider + providerID.
	u, err := s.store.Users.GetByProvider(ctx, provider, p.ID)
	if err == nil {
		// Existing social user — refresh picture/name.
		u.Name = p.Name
		u.Picture = p.Picture
		u.UpdatedAt = time.Now()
		_ = s.store.Users.Update(ctx, u)
		return u, nil
	}
	if !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	// 2. Try by email — link the provider to an existing account.
	if p.Email != "" {
		u, err = s.store.Users.GetByEmail(ctx, strings.ToLower(p.Email))
		if err == nil {
			u.Provider = provider
			u.ProviderID = p.ID
			u.Picture = p.Picture
			u.EmailVerified = true // social providers verify the email
			u.UpdatedAt = time.Now()
			_ = s.store.Users.Update(ctx, u)
			return u, nil
		}
	}

	// 3. Create new user.
	u = &model.User{
		ID:            uuid.NewString(),
		Email:         strings.ToLower(p.Email),
		Name:          p.Name,
		Picture:       p.Picture,
		Provider:      provider,
		ProviderID:    p.ID,
		EmailVerified: true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.store.Users.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("social: create user: %w", err)
	}
	return u, nil
}

// ── Google ────────────────────────────────────────────────────────────────────

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func fetchGoogleProfile(ctx context.Context, token *oauth2.Token) (*socialProfile, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var g googleUserInfo
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, err
	}
	return &socialProfile{ID: g.Sub, Email: g.Email, Name: g.Name, Picture: g.Picture}, nil
}

// ── GitHub ────────────────────────────────────────────────────────────────────

type githubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func fetchGitHubProfile(ctx context.Context, token *oauth2.Token) (*socialProfile, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var g githubUserInfo
	if err := json.Unmarshal(body, &g); err != nil {
		return nil, err
	}

	name := g.Name
	if name == "" {
		name = g.Login
	}

	emailAddr := g.Email
	// GitHub may hide email; fall back to /user/emails.
	if emailAddr == "" {
		emailAddr = fetchGitHubPrimaryEmail(client)
	}

	return &socialProfile{
		ID:      fmt.Sprintf("%d", g.ID),
		Email:   emailAddr,
		Name:    name,
		Picture: g.AvatarURL,
	}, nil
}

func fetchGitHubPrimaryEmail(client *http.Client) string {
	resp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var emails []githubEmail
	if err := json.Unmarshal(body, &emails); err != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}
