-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
VALUES (
    $1,
    NOW(),
    NOW(),
    $2,
    $3
)
RETURNING *;

-- name: QueryAllChirps :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: QueryAllAuthorChirps :many
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: QueryChirp :one
SELECT * FROM chirps
WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1;

-- name: QueryRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: RevokeToken :one
UPDATE refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE token = $1
RETURNING *;


-- name: QueryUser :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET hashed_password = $2, email = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UserPro :one
UPDATE users
SET is_chirpy_red = TRUE
WHERE id = $1
RETURNING *;

-- name: DeleteUsers :exec
DELETE FROM users;