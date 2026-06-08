CREATE TABLE IF NOT EXISTS users (
    id         UUID PRIMARY KEY,
    login      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS data_entries (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(32) NOT NULL,
    metadata   TEXT NOT NULL DEFAULT '',
    payload    BYTEA NOT NULL,
    version    BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted    BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_data_entries_user_id ON data_entries(user_id);
CREATE INDEX IF NOT EXISTS idx_data_entries_user_updated ON data_entries(user_id, updated_at);
