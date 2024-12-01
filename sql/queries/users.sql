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

-- name: DeleteAll :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT * from users WHERE email = $1;

-- name: UpdateEmailAndPassword :one
UPDATE users SET email = $2, hashed_password = $3, updated_at = NOW()
where id =$1
RETURNING *;

-- name: UpgradeToRed :one
UPDATE users SET is_chirpy_red = true, updated_at = NOW()
WHERE id = $1
RETURNING *;
