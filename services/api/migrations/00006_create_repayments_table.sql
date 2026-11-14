-- +goose Up
CREATE TABLE IF NOT EXISTS repayments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    loan_id UUID NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    amount NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'INR',
    payment_date DATE NOT NULL,
    payment_method VARCHAR(50) NOT NULL DEFAULT 'UPI',
    external_reference VARCHAR(100),
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_VERIFICATION' CHECK (status IN ('PENDING_VERIFICATION', 'CONFIRMED', 'REJECTED', 'REVERSED')),
    source VARCHAR(20) NOT NULL DEFAULT 'LENDER_RECORDED' CHECK (source IN ('LENDER_RECORDED', 'BORROWER_REPORTED')),
    confirmed_at TIMESTAMPTZ,
    confirmed_by UUID,
    rejection_reason TEXT,
    reversed_at TIMESTAMPTZ,
    reversal_reason TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_repayments_user_id ON repayments (user_id);
CREATE INDEX IF NOT EXISTS idx_repayments_loan_id ON repayments (loan_id);
CREATE INDEX IF NOT EXISTS idx_repayments_status ON repayments (status);
CREATE INDEX IF NOT EXISTS idx_repayments_loan_status ON repayments (loan_id, status);
CREATE INDEX IF NOT EXISTS idx_repayments_payment_date ON repayments (payment_date DESC);

-- +goose Down
DROP TABLE IF EXISTS repayments;
