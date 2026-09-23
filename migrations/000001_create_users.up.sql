CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE IF NOT EXISTS users(
    id BIGSERIAL PRIMARY KEY,
    username CITEXT NOT NULL,
    password_hash TEXT NOT NULL,
    game_tag TEXT,
    profile_picture TEXT,
    created_at TIMESTAMPTZ(0) NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT users_username_unique UNIQUE (username)
);
