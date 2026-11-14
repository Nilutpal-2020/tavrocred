-- +goose Up
CREATE TABLE IF NOT EXISTS loans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    peer_id UUID NOT NULL REFERENCES peers(id) ON DELETE RESTRICT,
    principal_amount NUMERIC(15, 2) NOT NULL CHECK (principal_amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    interest_type VARCHAR(20) NOT NULL DEFAULT 'NONE' CHECK (interest_type IN ('NONE', 'SIMPLE')),
    annual_interest_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00 CHECK (annual_interest_rate >= 0),
    day_count_convention VARCHAR(20) NOT NULL DEFAULT 'ACTUAL_365',
    start_date DATE NOT NULL,
    due_date DATE,
    status VARCHAR(20) NOT NULL DEFAULT 'DRAFT' CHECK (status IN ('DRAFT', 'ACTIVE', 'COMPLETED', 'CANCELLED', 'DISPUTED')),
    version INT NOT NULL DEFAULT 1 CHECK (version >= 1),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_due_date_after_start CHECK (due_date IS NULL OR due_date >= start_date),
    CONSTRAINT check_interest_rate_none CHECK (interest_type != 'NONE' OR annual_interest_rate = 0.00)
);

CREATE INDEX IF NOT EXISTS idx_loans_user_id ON loans (user_id);
CREATE INDEX IF NOT EXISTS idx_loans_peer_id ON loans (peer_id);
CREATE INDEX IF NOT EXISTS idx_loans_status ON loans (status);
CREATE INDEX IF NOT EXISTS idx_loans_user_peer ON loans (user_id, peer_id);
CREATE INDEX IF NOT EXISTS idx_loans_user_created ON loans (user_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS loans;
