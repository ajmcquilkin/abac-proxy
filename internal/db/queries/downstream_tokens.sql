-- name: CreateDownstreamToken :one
INSERT INTO downstream_tokens (
    policy_id,
    token_hash,
    name,
    expires_at
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: GetDownstreamTokenByHash :one
SELECT
    dt.*,
    sqlc.embed(p),
    sqlc.embed(uc)
FROM downstream_tokens dt
JOIN policies p ON p.id = dt.policy_id
JOIN upstream_credentials uc ON uc.id = p.upstream_credential_id
WHERE dt.token_hash = $1
  AND dt.revoked = FALSE
  AND (dt.expires_at IS NULL OR dt.expires_at > NOW())
  AND p.is_active = TRUE
  AND (uc.expires_at IS NULL OR uc.expires_at > NOW());

-- name: ListDownstreamTokensByPolicyID :many
SELECT * FROM downstream_tokens
WHERE policy_id = $1
ORDER BY created_at DESC;

-- name: UpdateDownstreamTokenLastUsed :exec
UPDATE downstream_tokens
SET last_used_at = NOW()
WHERE id = $1;

-- name: RevokeDownstreamToken :exec
UPDATE downstream_tokens
SET
    revoked = TRUE,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteDownstreamToken :exec
DELETE FROM downstream_tokens
WHERE id = $1;
