// Package memory provides a thread-safe in-memory implementation of all store
// interfaces. Use this for local development and testing. Replace with a
// Postgres + Redis implementation for production.
package memory

import (
	"context"
	"sync"
	"time"

	"auth-server/internal/model"
	"auth-server/internal/store"
)

// ── Users ────────────────────────────────────────────────────────────────────

type userStore struct {
	mu      sync.RWMutex
	byID    map[string]*model.User
	byEmail map[string]*model.User
}

func NewUserStore() store.UserStore {
	return &userStore{byID: make(map[string]*model.User), byEmail: make(map[string]*model.User)}
}

func (s *userStore) Create(_ context.Context, u *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byEmail[u.Email]; ok {
		return store.ErrDuplicate
	}
	cp := *u
	s.byID[u.ID] = &cp
	s.byEmail[u.Email] = &cp
	return nil
}

func (s *userStore) GetByID(_ context.Context, id string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byID[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *userStore) GetByEmail(_ context.Context, email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.byEmail[email]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (s *userStore) GetByProvider(_ context.Context, provider model.AuthProvider, providerID string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.byID {
		if u.Provider == provider && u.ProviderID == providerID {
			cp := *u
			return &cp, nil
		}
	}
	return nil, store.ErrNotFound
}

func (s *userStore) Update(_ context.Context, u *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, ok := s.byID[u.ID]
	if !ok {
		return store.ErrNotFound
	}
	if old.Email != u.Email {
		delete(s.byEmail, old.Email)
		s.byEmail[u.Email] = u
	}
	cp := *u
	s.byID[u.ID] = &cp
	s.byEmail[u.Email] = &cp
	return nil
}

func (s *userStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.byID[id]
	if !ok {
		return store.ErrNotFound
	}
	delete(s.byID, id)
	delete(s.byEmail, u.Email)
	return nil
}

// ── Tokens ───────────────────────────────────────────────────────────────────

type tokenStore struct {
	mu            sync.RWMutex
	refreshTokens map[string]*model.RefreshToken // key = tokenHash
	emailTokens   map[string]*model.EmailToken   // key = tokenHash
}

func NewTokenStore() store.TokenStore {
	return &tokenStore{
		refreshTokens: make(map[string]*model.RefreshToken),
		emailTokens:   make(map[string]*model.EmailToken),
	}
}

func (s *tokenStore) SaveRefreshToken(_ context.Context, t *model.RefreshToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.refreshTokens[t.TokenHash] = &cp
	return nil
}

func (s *tokenStore) GetRefreshToken(_ context.Context, hash string) (*model.RefreshToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.refreshTokens[hash]
	if !ok {
		return nil, store.ErrNotFound
	}
	if t.Revoked {
		return nil, store.ErrTokenRevoked
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	cp := *t
	return &cp, nil
}

func (s *tokenStore) RevokeRefreshToken(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.refreshTokens[hash]
	if !ok {
		return store.ErrNotFound
	}
	t.Revoked = true
	return nil
}

func (s *tokenStore) RevokeAllUserRefreshTokens(_ context.Context, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.refreshTokens {
		if t.UserID == userID {
			t.Revoked = true
		}
	}
	return nil
}

func (s *tokenStore) SaveEmailToken(_ context.Context, t *model.EmailToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.emailTokens[t.TokenHash] = &cp
	return nil
}

func (s *tokenStore) GetEmailToken(_ context.Context, hash string) (*model.EmailToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.emailTokens[hash]
	if !ok {
		return nil, store.ErrNotFound
	}
	if t.UsedAt != nil {
		return nil, store.ErrTokenUsed
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	cp := *t
	return &cp, nil
}

func (s *tokenStore) MarkEmailTokenUsed(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.emailTokens[hash]
	if !ok {
		return store.ErrNotFound
	}
	now := time.Now()
	t.UsedAt = &now
	return nil
}

func (s *tokenStore) RevokeEmailTokens(_ context.Context, userID string, tokenType model.EmailTokenType) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for _, t := range s.emailTokens {
		if t.UserID == userID && t.Type == tokenType && t.UsedAt == nil {
			t.UsedAt = &now
		}
	}
	return nil
}

// ── OAuth ─────────────────────────────────────────────────────────────────────

type oauthStore struct {
	mu      sync.RWMutex
	clients map[string]*model.OAuthClient
	codes   map[string]*model.AuthorizationCode
	tokens  map[string]*model.OAuthToken
}

func NewOAuthStore() store.OAuthStore {
	return &oauthStore{
		clients: make(map[string]*model.OAuthClient),
		codes:   make(map[string]*model.AuthorizationCode),
		tokens:  make(map[string]*model.OAuthToken),
	}
}

func (s *oauthStore) SaveClient(_ context.Context, c *model.OAuthClient) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *c
	s.clients[c.ID] = &cp
	return nil
}

func (s *oauthStore) GetClient(_ context.Context, id string) (*model.OAuthClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clients[id]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (s *oauthStore) ListClients(_ context.Context) ([]*model.OAuthClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*model.OAuthClient, 0, len(s.clients))
	for _, c := range s.clients {
		cp := *c
		out = append(out, &cp)
	}
	return out, nil
}

func (s *oauthStore) DeleteClient(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clients[id]; !ok {
		return store.ErrNotFound
	}
	delete(s.clients, id)
	return nil
}

func (s *oauthStore) SaveCode(_ context.Context, code *model.AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *code
	s.codes[code.Code] = &cp
	return nil
}

func (s *oauthStore) GetCode(_ context.Context, code string) (*model.AuthorizationCode, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.codes[code]
	if !ok {
		return nil, store.ErrNotFound
	}
	if c.Used {
		return nil, store.ErrTokenUsed
	}
	if time.Now().After(c.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	cp := *c
	return &cp, nil
}

func (s *oauthStore) MarkCodeUsed(_ context.Context, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.codes[code]
	if !ok {
		return store.ErrNotFound
	}
	c.Used = true
	return nil
}

func (s *oauthStore) SaveOAuthToken(_ context.Context, t *model.OAuthToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *t
	s.tokens[t.TokenHash] = &cp
	return nil
}

func (s *oauthStore) GetOAuthToken(_ context.Context, hash string) (*model.OAuthToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tokens[hash]
	if !ok {
		return nil, store.ErrNotFound
	}
	if t.Revoked {
		return nil, store.ErrTokenRevoked
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	cp := *t
	return &cp, nil
}

func (s *oauthStore) RevokeOAuthToken(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tokens[hash]
	if !ok {
		return store.ErrNotFound
	}
	t.Revoked = true
	return nil
}

func (s *oauthStore) RevokeClientTokens(_ context.Context, clientID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.tokens {
		if t.ClientID == clientID {
			t.Revoked = true
		}
	}
	return nil
}

// ── Constructor ───────────────────────────────────────────────────────────────

// New returns a store.Store wired with all in-memory implementations.
func New() store.Store {
	return store.Store{
		Users:  NewUserStore(),
		Tokens: NewTokenStore(),
		OAuth:  NewOAuthStore(),
	}
}
