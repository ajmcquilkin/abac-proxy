-- name: CreateUpstreamCredential :one
INSERT INTO upstream_credentials (
    user_id,
    name,
    token,
    api_endpoint,
    token_type,
    expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetUpstreamCredentialByID :one
SELECT * FROM upstream_credentials
WHERE id = $1;

-- name: ListUpstreamCredentialsByUserID :many
SELECT * FROM upstream_credentials
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: ListActiveUpstreamCredentials :many
SELECT * FROM upstream_credentials
WHERE user_id = $1
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC;

-- name: UpdateUpstreamCredential :one
UPDATE upstream_credentials
SET
    name = COALESCE(sqlc.narg('name'), name),
    token = COALESCE(sqlc.narg('token'), token),
    api_endpoint = COALESCE(sqlc.narg('api_endpoint'), api_endpoint),
    token_type = COALESCE(sqlc.narg('token_type'), token_type),
    expires_at = COALESCE(sqlc.narg('expires_at'), expires_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUpstreamCredential :exec
DELETE FROM upstream_credentials
WHERE id = $1;
