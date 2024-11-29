-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at)
values ($1,
        NOW(),
        NOW(),
        $2,
        $3
       )
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: GetUserFromRefreshToken :one
SELECT u.* FROM users u
JOIN refresh_tokens rt ON u.id = rt.user_id
WHERE rt.token = $1
AND revoked_at IS NULL
AND expires_at > NOW();

-- name: RevokeToken :exec
UPDATE refresh_tokens SET updated_at = NOW(), revoked_at = NOW() WHERE token = $1;
