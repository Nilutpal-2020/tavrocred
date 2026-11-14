-- name: CreateRepayment :one
INSERT INTO repayments (
    user_id,
    loan_id,
    amount,
    currency,
    payment_date,
    payment_method,
    external_reference,
    status,
    source,
    confirmed_at,
    confirmed_by,
    notes
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12
)
RETURNING id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
          status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
          notes, created_at, updated_at;

-- name: GetRepaymentByIDAndUser :one
SELECT id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
       status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
       notes, created_at, updated_at
FROM repayments
WHERE id = $1 AND user_id = $2;

-- name: GetRepaymentForUpdate :one
SELECT id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
       status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
       notes, created_at, updated_at
FROM repayments
WHERE id = $1 AND user_id = $2
FOR UPDATE;

-- name: ListRepaymentsByLoan :many
SELECT id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
       status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
       notes, created_at, updated_at
FROM repayments
WHERE user_id = $1 AND loan_id = $2
  AND ($3::varchar = '' OR status = $3)
ORDER BY payment_date DESC, created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountRepaymentsByLoan :one
SELECT COUNT(*)
FROM repayments
WHERE user_id = $1 AND loan_id = $2
  AND ($3::varchar = '' OR status = $3);

-- name: GetConfirmedRepaymentsTotalByLoan :one
SELECT COALESCE(SUM(amount), 0)::numeric AS confirmed_total
FROM repayments
WHERE loan_id = $1 AND user_id = $2 AND status = 'CONFIRMED';

-- name: ConfirmRepayment :one
UPDATE repayments
SET status = 'CONFIRMED',
    confirmed_at = CURRENT_TIMESTAMP,
    confirmed_by = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status = 'PENDING_VERIFICATION'
RETURNING id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
          status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
          notes, created_at, updated_at;

-- name: RejectRepayment :one
UPDATE repayments
SET status = 'REJECTED',
    rejection_reason = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status = 'PENDING_VERIFICATION'
RETURNING id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
          status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
          notes, created_at, updated_at;

-- name: ReverseRepayment :one
UPDATE repayments
SET status = 'REVERSED',
    reversed_at = CURRENT_TIMESTAMP,
    reversal_reason = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status = 'CONFIRMED'
RETURNING id, user_id, loan_id, amount, currency, payment_date, payment_method, external_reference,
          status, source, confirmed_at, confirmed_by, rejection_reason, reversed_at, reversal_reason,
          notes, created_at, updated_at;
