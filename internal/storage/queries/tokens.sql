-- name: GetTokenByJTI :one
SELECT * FROM tokens WHERE jti = $1 LIMIT 1;

-- name: CreateToken :one
INSERT INTO tokens (user_id, jti, token_hash, scopes, issued_at, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: RevokeToken :exec
UPDATE tokens SET revoked_at = NOW() WHERE jti = $1;

-- name: UpdateTokenLastUsed :exec
UPDATE tokens SET last_used_at = NOW() WHERE jti = $1;

-- name: ListActiveTokensForUser :many
SELECT * FROM tokens
WHERE user_id = $1
  AND revoked_at IS NULL
  AND expires_at > NOW()
ORDER BY issued_at DESC;
