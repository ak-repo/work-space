// Package postgres provides a Postgres-backed implementation of all store
// interfaces. Use this in production with a pgx pool.
package postgres

import (
	"context"
	"errors"
	"time"

	"auth-server/internal/model"
	"auth-server/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	pgUniqueViolation = "23505"
)

// mapPgError normalizes pg errors to store sentinels.
func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return store.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgUniqueViolation {
			return store.ErrDuplicate
		}
	}
	return err
}

// ── Users ────────────────────────────────────────────────────────────────────

type userStore struct {
	pool *pgxpool.Pool
}

func NewUserStore(pool *pgxpool.Pool) store.UserStore {
	return &userStore{pool: pool}
}

func (s *userStore) Create(ctx context.Context, u *model.User) error {
	_, err := s.pool.Exec(ctx, `
        insert into users (
            id, email, name, picture, password_hash, email_verified,
            provider, provider_id, disabled, created_at, updated_at
        ) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `,
		u.ID, u.Email, u.Name, u.Picture, u.PasswordHash, u.EmailVerified,
		u.Provider, u.ProviderID, u.Disabled, u.CreatedAt, u.UpdatedAt,
	)
	return mapPgError(err)
}

func (s *userStore) GetByID(ctx context.Context, id string) (*model.User, error) {
	row := s.pool.QueryRow(ctx, `
        select id, email, name, picture, password_hash, email_verified,
               provider, provider_id, disabled, created_at, updated_at
        from users
        where id = $1
    `, id)
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.Picture, &u.PasswordHash, &u.EmailVerified,
		&u.Provider, &u.ProviderID, &u.Disabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &u, nil
}

func (s *userStore) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row := s.pool.QueryRow(ctx, `
        select id, email, name, picture, password_hash, email_verified,
               provider, provider_id, disabled, created_at, updated_at
        from users
        where email = $1
    `, email)
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.Picture, &u.PasswordHash, &u.EmailVerified,
		&u.Provider, &u.ProviderID, &u.Disabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &u, nil
}

func (s *userStore) GetByProvider(ctx context.Context, provider model.AuthProvider, providerID string) (*model.User, error) {
	row := s.pool.QueryRow(ctx, `
        select id, email, name, picture, password_hash, email_verified,
               provider, provider_id, disabled, created_at, updated_at
        from users
        where provider = $1 and provider_id = $2
    `, provider, providerID)
	var u model.User
	err := row.Scan(
		&u.ID, &u.Email, &u.Name, &u.Picture, &u.PasswordHash, &u.EmailVerified,
		&u.Provider, &u.ProviderID, &u.Disabled, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &u, nil
}

func (s *userStore) Update(ctx context.Context, u *model.User) error {
	tag, err := s.pool.Exec(ctx, `
        update users set
            email = $2,
            name = $3,
            picture = $4,
            password_hash = $5,
            email_verified = $6,
            provider = $7,
            provider_id = $8,
            disabled = $9,
            created_at = $10,
            updated_at = $11
        where id = $1
    `,
		u.ID, u.Email, u.Name, u.Picture, u.PasswordHash, u.EmailVerified,
		u.Provider, u.ProviderID, u.Disabled, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *userStore) Delete(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `delete from users where id = $1`, id)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

// ── Tokens ───────────────────────────────────────────────────────────────────

type tokenStore struct {
	pool *pgxpool.Pool
}

func NewTokenStore(pool *pgxpool.Pool) store.TokenStore {
	return &tokenStore{pool: pool}
}

func (s *tokenStore) SaveRefreshToken(ctx context.Context, t *model.RefreshToken) error {
	_, err := s.pool.Exec(ctx, `
        insert into refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
        values ($1,$2,$3,$4,$5,$6)
    `, t.ID, t.UserID, t.TokenHash, t.ExpiresAt, t.Revoked, t.CreatedAt)
	return mapPgError(err)
}

func (s *tokenStore) GetRefreshToken(ctx context.Context, hash string) (*model.RefreshToken, error) {
	row := s.pool.QueryRow(ctx, `
        select id, user_id, token_hash, expires_at, revoked, created_at
        from refresh_tokens
        where token_hash = $1
    `, hash)
	var t model.RefreshToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		return nil, mapPgError(err)
	}
	if t.Revoked {
		return nil, store.ErrTokenRevoked
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	return &t, nil
}

func (s *tokenStore) RevokeRefreshToken(ctx context.Context, hash string) error {
	tag, err := s.pool.Exec(ctx, `
        update refresh_tokens set revoked = true where token_hash = $1
    `, hash)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *tokenStore) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	_, err := s.pool.Exec(ctx, `
        update refresh_tokens set revoked = true where user_id = $1
    `, userID)
	return mapPgError(err)
}

func (s *tokenStore) SaveEmailToken(ctx context.Context, t *model.EmailToken) error {
	_, err := s.pool.Exec(ctx, `
        insert into email_tokens (id, user_id, token_hash, type, expires_at, used_at, created_at)
        values ($1,$2,$3,$4,$5,$6,$7)
    `, t.ID, t.UserID, t.TokenHash, t.Type, t.ExpiresAt, t.UsedAt, t.CreatedAt)
	return mapPgError(err)
}

func (s *tokenStore) GetEmailToken(ctx context.Context, hash string) (*model.EmailToken, error) {
	row := s.pool.QueryRow(ctx, `
        select id, user_id, token_hash, type, expires_at, used_at, created_at
        from email_tokens
        where token_hash = $1
    `, hash)
	var t model.EmailToken
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.Type, &t.ExpiresAt, &t.UsedAt, &t.CreatedAt)
	if err != nil {
		return nil, mapPgError(err)
	}
	if t.UsedAt != nil {
		return nil, store.ErrTokenUsed
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	return &t, nil
}

func (s *tokenStore) MarkEmailTokenUsed(ctx context.Context, hash string) error {
	tag, err := s.pool.Exec(ctx, `
        update email_tokens set used_at = now() where token_hash = $1
    `, hash)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *tokenStore) RevokeEmailTokens(ctx context.Context, userID string, tokenType model.EmailTokenType) error {
	_, err := s.pool.Exec(ctx, `
        update email_tokens set used_at = now()
        where user_id = $1 and type = $2 and used_at is null
    `, userID, tokenType)
	return mapPgError(err)
}

// ── OAuth ─────────────────────────────────────────────────────────────────────

type oauthStore struct {
	pool *pgxpool.Pool
}

func NewOAuthStore(pool *pgxpool.Pool) store.OAuthStore {
	return &oauthStore{pool: pool}
}

func (s *oauthStore) SaveClient(ctx context.Context, c *model.OAuthClient) error {
	_, err := s.pool.Exec(ctx, `
        insert into oauth_clients (id, secret, name, redirect_uris, scopes, grant_types, public, created_at)
        values ($1,$2,$3,$4,$5,$6,$7,$8)
    `, c.ID, c.Secret, c.Name, c.RedirectURIs, c.Scopes, c.GrantTypes, c.Public, c.CreatedAt)
	return mapPgError(err)
}

func (s *oauthStore) GetClient(ctx context.Context, id string) (*model.OAuthClient, error) {
	row := s.pool.QueryRow(ctx, `
        select id, secret, name, redirect_uris, scopes, grant_types, public, created_at
        from oauth_clients
        where id = $1
    `, id)
	var c model.OAuthClient
	err := row.Scan(&c.ID, &c.Secret, &c.Name, &c.RedirectURIs, &c.Scopes, &c.GrantTypes, &c.Public, &c.CreatedAt)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &c, nil
}

func (s *oauthStore) ListClients(ctx context.Context) ([]*model.OAuthClient, error) {
	rows, err := s.pool.Query(ctx, `
        select id, secret, name, redirect_uris, scopes, grant_types, public, created_at
        from oauth_clients
        order by created_at desc
    `)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()
	out := make([]*model.OAuthClient, 0)
	for rows.Next() {
		var c model.OAuthClient
		if err := rows.Scan(&c.ID, &c.Secret, &c.Name, &c.RedirectURIs, &c.Scopes, &c.GrantTypes, &c.Public, &c.CreatedAt); err != nil {
			return nil, mapPgError(err)
		}
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	return out, nil
}

func (s *oauthStore) DeleteClient(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `delete from oauth_clients where id = $1`, id)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *oauthStore) SaveCode(ctx context.Context, code *model.AuthorizationCode) error {
	_, err := s.pool.Exec(ctx, `
        insert into authorization_codes (
            code, client_id, user_id, redirect_uri, scopes,
            code_challenge, code_challenge_method, expires_at, used, created_at
        ) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
    `,
		code.Code, code.ClientID, code.UserID, code.RedirectURI, code.Scopes,
		code.CodeChallenge, code.CodeChallengeMethod, code.ExpiresAt, code.Used, code.CreatedAt,
	)
	return mapPgError(err)
}

func (s *oauthStore) GetCode(ctx context.Context, code string) (*model.AuthorizationCode, error) {
	row := s.pool.QueryRow(ctx, `
        select code, client_id, user_id, redirect_uri, scopes,
               code_challenge, code_challenge_method, expires_at, used, created_at
        from authorization_codes
        where code = $1
    `, code)
	var c model.AuthorizationCode
	err := row.Scan(
		&c.Code, &c.ClientID, &c.UserID, &c.RedirectURI, &c.Scopes,
		&c.CodeChallenge, &c.CodeChallengeMethod, &c.ExpiresAt, &c.Used, &c.CreatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	if c.Used {
		return nil, store.ErrTokenUsed
	}
	if time.Now().After(c.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	return &c, nil
}

func (s *oauthStore) MarkCodeUsed(ctx context.Context, code string) error {
	tag, err := s.pool.Exec(ctx, `
        update authorization_codes set used = true where code = $1
    `, code)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *oauthStore) SaveOAuthToken(ctx context.Context, t *model.OAuthToken) error {
	_, err := s.pool.Exec(ctx, `
        insert into oauth_tokens (
            id, client_id, user_id, token_hash, scopes,
            token_type, expires_at, revoked, created_at
        ) values ($1,$2,$3,$4,$5,$6,$7,$8,$9)
    `,
		t.ID, t.ClientID, t.UserID, t.TokenHash, t.Scopes,
		t.TokenType, t.ExpiresAt, t.Revoked, t.CreatedAt,
	)
	return mapPgError(err)
}

func (s *oauthStore) GetOAuthToken(ctx context.Context, hash string) (*model.OAuthToken, error) {
	row := s.pool.QueryRow(ctx, `
        select id, client_id, user_id, token_hash, scopes, token_type, expires_at, revoked, created_at
        from oauth_tokens
        where token_hash = $1
    `, hash)
	var t model.OAuthToken
	err := row.Scan(&t.ID, &t.ClientID, &t.UserID, &t.TokenHash, &t.Scopes, &t.TokenType, &t.ExpiresAt, &t.Revoked, &t.CreatedAt)
	if err != nil {
		return nil, mapPgError(err)
	}
	if t.Revoked {
		return nil, store.ErrTokenRevoked
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, store.ErrTokenExpired
	}
	return &t, nil
}

func (s *oauthStore) RevokeOAuthToken(ctx context.Context, hash string) error {
	tag, err := s.pool.Exec(ctx, `
        update oauth_tokens set revoked = true where token_hash = $1
    `, hash)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *oauthStore) RevokeClientTokens(ctx context.Context, clientID string) error {
	_, err := s.pool.Exec(ctx, `
        update oauth_tokens set revoked = true where client_id = $1
    `, clientID)
	return mapPgError(err)
}

// ── Constructor ───────────────────────────────────────────────────────────────

// New returns a store.Store wired with all Postgres implementations.
func New(pool *pgxpool.Pool) store.Store {
	return store.Store{
		Users:  NewUserStore(pool),
		Tokens: NewTokenStore(pool),
		OAuth:  NewOAuthStore(pool),
	}
}
