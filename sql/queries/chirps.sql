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

-- name: GetChirps :many
SELECT * FROM chirps
ORDER BY
    CASE
        WHEN $1 = 'desc' THEN created_at
        ELSE NULL -- Use NULL for invalid cases
        END DESC,
    CASE
        WHEN $1 != 'desc' THEN created_at
        END ASC;

-- name: GetChirpsByAuthorId :many
SELECT * FROM chirps WHERE user_id = $1
ORDER BY
    CASE
        WHEN $2 = 'desc' THEN created_at
        ELSE NULL -- Use NULL for invalid cases
        END DESC,
    CASE
        WHEN $2 != 'desc' THEN created_at
        END ASC;

-- name: GetChirp :one
SELECT * FROM chirps WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps where id = $1;
