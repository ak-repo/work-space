-- Core users
create table if not exists users (
    id text primary key,
    email text not null unique,
    name text not null,
    picture text not null default '',
    password_hash text not null default '',
    email_verified boolean not null default false,
    provider text not null,
    provider_id text not null,
    disabled boolean not null default false,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    unique (provider, provider_id)
);

create index if not exists idx_users_provider on users (provider, provider_id);

-- Refresh tokens
create table if not exists refresh_tokens (
    id text primary key,
    user_id text not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamptz not null,
    revoked boolean not null default false,
    created_at timestamptz not null default now()
);

create index if not exists idx_refresh_tokens_user on refresh_tokens (user_id);

-- Email tokens (verify/reset)
create table if not exists email_tokens (
    id text primary key,
    user_id text not null references users(id) on delete cascade,
    token_hash text not null unique,
    type text not null,
    expires_at timestamptz not null,
    used_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists idx_email_tokens_user_type on email_tokens (user_id, type);

-- OAuth clients
create table if not exists oauth_clients (
    id text primary key,
    secret text not null default '',
    name text not null,
    redirect_uris text[] not null default '{}',
    scopes text[] not null default '{}',
    grant_types text[] not null default '{}',
    public boolean not null default false,
    created_at timestamptz not null default now()
);

-- Authorization codes
create table if not exists authorization_codes (
    code text primary key,
    client_id text not null references oauth_clients(id) on delete cascade,
    user_id text not null references users(id) on delete cascade,
    redirect_uri text not null,
    scopes text[] not null default '{}',
    code_challenge text not null,
    code_challenge_method text not null,
    expires_at timestamptz not null,
    used boolean not null default false,
    created_at timestamptz not null default now()
);

create index if not exists idx_auth_codes_client on authorization_codes (client_id);
create index if not exists idx_auth_codes_user on authorization_codes (user_id);

-- OAuth tokens
create table if not exists oauth_tokens (
    id text primary key,
    client_id text not null references oauth_clients(id) on delete cascade,
    user_id text references users(id) on delete cascade,
    token_hash text not null unique,
    scopes text[] not null default '{}',
    token_type text not null,
    expires_at timestamptz not null,
    revoked boolean not null default false,
    created_at timestamptz not null default now()
);

create index if not exists idx_oauth_tokens_client on oauth_tokens (client_id);
create index if not exists idx_oauth_tokens_user on oauth_tokens (user_id);
