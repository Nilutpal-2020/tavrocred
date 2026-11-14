-- name: CreateLoan :one
INSERT INTO loans (
    user_id,
    peer_id,
    principal_amount,
    currency,
    interest_type,
    annual_interest_rate,
    day_count_convention,
    start_date,
    due_date,
    status,
    version,
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
    'DRAFT',
    1,
    $10
)
RETURNING id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
          day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at;

-- name: GetLoanByIDAndUser :one
SELECT id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
       day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at
FROM loans
WHERE id = $1 AND user_id = $2;

-- name: GetLoanForUpdate :one
SELECT id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
       day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at
FROM loans
WHERE id = $1 AND user_id = $2
FOR UPDATE;

-- name: ListLoansByUser :many
SELECT l.id, l.user_id, l.peer_id, l.principal_amount, l.currency, l.interest_type, l.annual_interest_rate,
       l.day_count_convention, l.start_date, l.due_date, l.status, l.version, l.notes, l.created_at, l.updated_at,
       p.name AS peer_name, p.email AS peer_email
FROM loans l
JOIN peers p ON l.peer_id = p.id
WHERE l.user_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR l.peer_id = $2)
  AND ($3::varchar = '' OR l.status = $3)
ORDER BY l.created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountLoansByUser :one
SELECT COUNT(*)
FROM loans
WHERE user_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR peer_id = $2)
  AND ($3::varchar = '' OR status = $3);

-- name: UpdateDraftLoan :one
UPDATE loans
SET peer_id = $3,
    principal_amount = $4,
    currency = $5,
    interest_type = $6,
    annual_interest_rate = $7,
    day_count_convention = $8,
    start_date = $9,
    due_date = $10,
    notes = $11,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status = 'DRAFT'
RETURNING id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
          day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at;

-- name: UpdateLoanStatus :one
UPDATE loans
SET status = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
          day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at;

-- name: AmendLoan :one
UPDATE loans
SET principal_amount = $3,
    interest_type = $4,
    annual_interest_rate = $5,
    due_date = $6,
    notes = $7,
    version = version + 1,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2 AND status = 'ACTIVE'
RETURNING id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
          day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at;

-- name: DeleteDraftLoan :execrows
DELETE FROM loans
WHERE id = $1 AND user_id = $2 AND status = 'DRAFT';

-- name: GetLoanByIDAndPeer :one
SELECT id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
       day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at
FROM loans
WHERE id = $1 AND peer_id = $2;

-- name: UpdateLoanStatusByPeer :one
UPDATE loans
SET status = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND peer_id = $2
RETURNING id, user_id, peer_id, principal_amount, currency, interest_type, annual_interest_rate,
          day_count_convention, start_date, due_date, status, version, notes, created_at, updated_at;
