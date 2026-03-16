-- name: GetActivePolicyForUser :one
SELECT * FROM policies
WHERE user_id = $1 AND is_active = TRUE
LIMIT 1;

-- name: CreatePolicy :one
INSERT INTO policies (user_id, upstream_credential_id, version, base_url, default_action, rules, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: DeactivateUserPolicies :exec
UPDATE policies SET is_active = FALSE WHERE user_id = $1;

-- name: ActivatePolicy :exec
UPDATE policies SET is_active = TRUE WHERE id = $1;

-- name: ListPolicyVersions :many
SELECT * FROM policies
WHERE user_id = $1
ORDER BY created_at DESC;
