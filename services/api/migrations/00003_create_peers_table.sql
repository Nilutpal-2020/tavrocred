-- +goose Up
CREATE TABLE IF NOT EXISTS peers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_peers_user_id ON peers (user_id);
CREATE INDEX IF NOT EXISTS idx_peers_user_name ON peers (user_id, LOWER(name));

-- +goose Down
DROP TABLE IF EXISTS peers;

