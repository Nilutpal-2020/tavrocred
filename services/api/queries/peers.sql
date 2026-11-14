-- name: CreatePeer :one
INSERT INTO peers (
    user_id,
    name,
    email,
    phone,
    notes
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING id, user_id, name, email, phone, notes, created_at, updated_at;

-- name: GetPeerByIDAndUser :one
SELECT id, user_id, name, email, phone, notes, created_at, updated_at
FROM peers
WHERE id = $1 AND user_id = $2;

-- name: ListPeersByUser :many
SELECT id, user_id, name, email, phone, notes, created_at, updated_at
FROM peers
WHERE user_id = $1
  AND ($2 = '' OR LOWER(name) LIKE LOWER('%' || $2 || '%') OR LOWER(COALESCE(email, '')) LIKE LOWER('%' || $2 || '%'))
ORDER BY name ASC, created_at DESC
LIMIT $3 OFFSET $4;

-- name: CountPeersByUser :one
SELECT COUNT(*)
FROM peers
WHERE user_id = $1
  AND ($2 = '' OR LOWER(name) LIKE LOWER('%' || $2 || '%') OR LOWER(COALESCE(email, '')) LIKE LOWER('%' || $2 || '%'));

-- name: UpdatePeer :one
UPDATE peers
SET name = $3,
    email = $4,
    phone = $5,
    notes = $6,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, name, email, phone, notes, created_at, updated_at;

-- name: DeletePeer :execrows
DELETE FROM peers
WHERE id = $1 AND user_id = $2;

