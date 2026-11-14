-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS borrower_access_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id UUID NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    purpose VARCHAR(32) NOT NULL DEFAULT 'PORTAL_ACCESS',
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_borrower_tokens_hash ON borrower_access_tokens(token_hash);
CREATE INDEX idx_borrower_tokens_loan ON borrower_access_tokens(loan_id);
CREATE INDEX idx_borrower_tokens_user ON borrower_access_tokens(user_id);

CREATE TABLE IF NOT EXISTS loan_acknowledgements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loan_id UUID NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE CASCADE,
    version INT NOT NULL DEFAULT 1,
    status VARCHAR(32) NOT NULL, -- 'ACKNOWLEDGED', 'DISPUTED'
    dispute_reason TEXT,
    ip_address VARCHAR(45),
    user_agent TEXT,
    responded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_loan_acknowledgements_loan ON loan_acknowledgements(loan_id, version);
CREATE INDEX idx_loan_acknowledgements_peer ON loan_acknowledgements(peer_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS loan_acknowledgements;
DROP TABLE IF EXISTS borrower_access_tokens;
-- +goose StatementEnd
