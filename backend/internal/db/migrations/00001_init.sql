-- +goose Up
CREATE TABLE IF NOT EXISTS system_accounts (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    password_updated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS system_accounts_singleton_idx ON system_accounts ((true));

CREATE TABLE IF NOT EXISTS ftp_account (
    id SMALLINT PRIMARY KEY CHECK (id = 1),
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    anonymous_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    password_updated_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS app_settings (
    key TEXT PRIMARY KEY,
    value_json JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS media_assets (
    id UUID PRIMARY KEY,
    original_filename TEXT NOT NULL,
    relative_dir TEXT NOT NULL,
    storage_path TEXT NOT NULL UNIQUE,
    content_hash TEXT NOT NULL,
    hash_algorithm TEXT NOT NULL,
    size BIGINT NOT NULL,
    mime_type TEXT NOT NULL,
    is_photo BOOLEAN NOT NULL,
    exif_taken_at TIMESTAMPTZ NULL,
    fallback_taken_at TIMESTAMPTZ NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS media_assets_photo_time_idx ON media_assets (is_photo, exif_taken_at DESC, fallback_taken_at DESC);
CREATE INDEX IF NOT EXISTS media_assets_hash_idx ON media_assets (content_hash);
CREATE INDEX IF NOT EXISTS media_assets_name_hash_idx ON media_assets (original_filename, content_hash);
CREATE INDEX IF NOT EXISTS media_assets_uploaded_idx ON media_assets (uploaded_at DESC);

CREATE TABLE IF NOT EXISTS transfer_events (
    id UUID PRIMARY KEY,
    asset_id UUID NULL REFERENCES media_assets(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('uploaded', 'overwritten', 'deleted', 'failed')),
    original_filename TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    remote_addr TEXT NOT NULL,
    bytes BIGINT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES system_accounts(id) ON DELETE CASCADE,
    session_token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS sessions_account_idx ON sessions (account_id);
CREATE INDEX IF NOT EXISTS sessions_expires_idx ON sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS transfer_events;
DROP TABLE IF EXISTS media_assets;
DROP TABLE IF EXISTS app_settings;
DROP TABLE IF EXISTS ftp_account;
DROP INDEX IF EXISTS system_accounts_singleton_idx;
DROP TABLE IF EXISTS system_accounts;
