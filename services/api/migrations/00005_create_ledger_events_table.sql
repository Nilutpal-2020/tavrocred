-- +goose Up
CREATE TABLE IF NOT EXISTS ledger_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    loan_id UUID REFERENCES loans(id) ON DELETE CASCADE,
    event_type VARCHAR(50) NOT NULL,
    actor_type VARCHAR(20) NOT NULL DEFAULT 'LENDER',
    actor_id UUID,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ledger_events_loan_id ON ledger_events (loan_id);
CREATE INDEX IF NOT EXISTS idx_ledger_events_user_id ON ledger_events (user_id);
CREATE INDEX IF NOT EXISTS idx_ledger_events_created ON ledger_events (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ledger_events_user_loan ON ledger_events (user_id, loan_id, created_at ASC);

-- +goose Down
DROP TABLE IF EXISTS ledger_events;
