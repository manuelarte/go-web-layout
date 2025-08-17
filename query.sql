-- name: GetUsers :many
SELECT * FROM users LIMIT ? OFFSET ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (
    id, username, password
) VALUES (
  ?, ?, ?
)
RETURNING *;
