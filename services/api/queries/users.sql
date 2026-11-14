-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash,
    full_name
) VALUES (
    LOWER($1),
    $2,
    $3
)
RETURNING id, email, password_hash, full_name, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, full_name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, full_name, created_at, updated_at
FROM users
WHERE LOWER(email) = LOWER($1);

