-- name: CreateLedgerEvent :one
INSERT INTO ledger_events (
    user_id,
    loan_id,
    event_type,
    actor_type,
    actor_id,
    payload
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING id, user_id, loan_id, event_type, actor_type, actor_id, payload, created_at;

-- name: ListLedgerEventsByLoan :many
SELECT id, user_id, loan_id, event_type, actor_type, actor_id, payload, created_at
FROM ledger_events
WHERE user_id = $1 AND loan_id = $2
ORDER BY created_at ASC;
