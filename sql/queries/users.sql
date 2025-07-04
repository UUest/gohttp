-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING id, created_at, updated_at, email, chirpy_red;

-- name: DeleteAllUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;

-- name: GetUserByRefreshToken :one
SELECT users.*
FROM users
JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $1,
    hashed_password = $2,
    updated_at = NOW()
WHERE id = $3
RETURNING id, created_at, updated_at, email, chirpy_red;

-- name: UpdateUserChirpyRed :one
UPDATE users
SET chirpy_red = $1,
    updated_at = NOW()
WHERE id = $2
RETURNING id, created_at, updated_at, email, chirpy_red;

-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = $1;
