-- name: GetRulesForPolicy :many
SELECT * FROM policy_rules
WHERE policy_id = $1
ORDER BY priority DESC, id;

-- name: CreatePolicyRule :one
INSERT INTO policy_rules (policy_id, route, method, action, response_filter, priority)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
