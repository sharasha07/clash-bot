CREATE TABLE IF NOT EXISTS chats(
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users ON DELETE CASCADE,
    name TEXT NOT NULL,
    updated_at TIMESTAMPTZ(0) NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ(0) NOT NULL DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS chats_user_id_updated_at_idx ON chats (user_id, updated_at DESC);
