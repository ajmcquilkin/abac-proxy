-- name: GetRateLimitForToken :one
SELECT * FROM rate_limits
WHERE token_id = $1 AND endpoint = $2 AND limit_type = $3
LIMIT 1;

-- name: CreateRateLimit :one
INSERT INTO rate_limits (token_id, endpoint, limit_type, requests_limit, window_seconds)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetTokenUsage :one
SELECT * FROM token_usage
WHERE token_id = $1 AND endpoint = $2 AND window_start = $3
LIMIT 1;

-- name: IncrementTokenUsage :one
INSERT INTO token_usage (token_id, endpoint, window_start, request_count, last_request_at)
VALUES ($1, $2, $3, 1, NOW())
ON CONFLICT (token_id, endpoint, window_start)
DO UPDATE SET
    request_count = token_usage.request_count + 1,
    last_request_at = NOW()
RETURNING *;
