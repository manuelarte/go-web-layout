-- name: GetUsers :many
SELECT * FROM users LIMIT ? OFFSET ?;

-- name: CountUsers :one
SELECT COUNT(*) FROM users;
