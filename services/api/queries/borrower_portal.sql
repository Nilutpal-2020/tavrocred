-- name: CreateBorrowerAccessToken :one
INSERT INTO borrower_access_tokens (
    loan_id,
    peer_id,
    user_id,
    token_hash,
    purpose,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING id, loan_id, peer_id, user_id, token_hash, purpose, expires_at, revoked_at, used_at, created_at;

-- name: GetActiveBorrowerAccessTokenByHash :one
SELECT id, loan_id, peer_id, user_id, token_hash, purpose, expires_at, revoked_at, used_at, created_at
FROM borrower_access_tokens
WHERE token_hash = $1
  AND revoked_at IS NULL
  AND expires_at > NOW();

-- name: RevokeBorrowerAccessTokensByLoan :execrows
UPDATE borrower_access_tokens
SET revoked_at = NOW()
WHERE loan_id = $1 AND user_id = $2 AND revoked_at IS NULL;

-- name: MarkBorrowerAccessTokenUsed :one
UPDATE borrower_access_tokens
SET used_at = NOW()
WHERE id = $1
RETURNING id, loan_id, peer_id, user_id, token_hash, purpose, expires_at, revoked_at, used_at, created_at;

-- name: CreateLoanAcknowledgement :one
INSERT INTO loan_acknowledgements (
    loan_id,
    peer_id,
    version,
    status,
    dispute_reason,
    ip_address,
    user_agent,
    responded_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING id, loan_id, peer_id, version, status, dispute_reason, ip_address, user_agent, responded_at, created_at;

-- name: GetLatestAcknowledgementByLoan :one
SELECT id, loan_id, peer_id, version, status, dispute_reason, ip_address, user_agent, responded_at, created_at
FROM loan_acknowledgements
WHERE loan_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: ListAcknowledgementsByLoan :many
SELECT id, loan_id, peer_id, version, status, dispute_reason, ip_address, user_agent, responded_at, created_at
FROM loan_acknowledgements
WHERE loan_id = $1
ORDER BY created_at DESC;
