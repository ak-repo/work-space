-- Run once before starting the server:
-- psql <DATABASE_URL> -f migrations/001_init.sql

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS products (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    sku         TEXT        UNIQUE NOT NULL,
    total_stock INT         NOT NULL DEFAULT 0 CHECK (total_stock >= 0),
    reserved    INT         NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    version     INT         NOT NULL DEFAULT 0,
    reorder_at  INT         NOT NULL DEFAULT 10
);

CREATE TABLE IF NOT EXISTS reservations (
    id          UUID        PRIMARY KEY,
    product_id  UUID        NOT NULL REFERENCES products(id),
    order_id    UUID        NOT NULL,
    quantity    INT         NOT NULL CHECK (quantity > 0),
    expires_at  TIMESTAMPTZ NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'confirmed', 'released'))
);

-- Index used by the TTL watcher query (status=pending AND expires_at <= now)
CREATE INDEX IF NOT EXISTS idx_reservations_ttl
    ON reservations (status, expires_at)
    WHERE status = 'pending';

-- Seed data for quick testing
INSERT INTO products (name, sku, total_stock, reserved, version, reorder_at)
VALUES
    ('Widget A', 'SKU-001', 100, 0, 0, 10),
    ('Widget B', 'SKU-002', 50,  0, 0, 5)
ON CONFLICT (sku) DO NOTHING;
