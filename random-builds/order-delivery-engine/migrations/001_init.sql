CREATE TABLE IF NOT EXISTS drivers (
    id             TEXT PRIMARY KEY,
    name           TEXT NOT NULL,
    phone          TEXT NOT NULL UNIQUE,
    status         TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available','busy','unavailable')),
    current_lat    DOUBLE PRECISION NOT NULL DEFAULT 0,
    current_lng    DOUBLE PRECISION NOT NULL DEFAULT 0,
    capacity       INT NOT NULL DEFAULT 3 CHECK (capacity > 0),
    active_orders  INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS orders (
    id               TEXT PRIMARY KEY,
    customer_name    TEXT NOT NULL,
    customer_phone   TEXT,
    pickup_lat       DOUBLE PRECISION NOT NULL,
    pickup_lng       DOUBLE PRECISION NOT NULL,
    pickup_address   TEXT,
    delivery_lat     DOUBLE PRECISION NOT NULL,
    delivery_lng     DOUBLE PRECISION NOT NULL,
    delivery_address TEXT,
    status           TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','assigned','picked_up','delivered','cancelled')),
    driver_id        TEXT REFERENCES drivers(id),
    priority         INT NOT NULL DEFAULT 1 CHECK (priority BETWEEN 1 AND 3),
    notes            TEXT DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    assigned_at      TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_orders_status    ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_driver_id ON orders(driver_id);
CREATE INDEX IF NOT EXISTS idx_drivers_status   ON drivers(status);
