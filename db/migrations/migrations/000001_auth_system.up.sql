-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE base_user (
    uuid UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(64) NOT NULL UNIQUE,
    user_type VARCHAR(8) NOT NULL CHECK (user_type IN ('sso', 'manual')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()  -- обновлять в go
);

CREATE TABLE manual_user (
    user_uuid UUID PRIMARY KEY REFERENCES base_user(uuid) ON DELETE CASCADE,
    password_hash BYTEA NOT NULL
);

CREATE TABLE sso_user (
    user_uuid UUID PRIMARY KEY REFERENCES base_user(uuid) ON DELETE CASCADE,
    provider VARCHAR(32) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    metadata JSONB,
    UNIQUE(provider, external_id)
);

-- +goose StatementEnd