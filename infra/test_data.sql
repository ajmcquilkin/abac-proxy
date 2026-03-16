-- Test data for local development

-- Insert test user
INSERT INTO users (id, email, role, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'test@example.com', 'admin', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert test policy
INSERT INTO policies (id, user_id, version, content, is_active, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'v1.0.0',
    '{
        "version": "1.0",
        "user": {
            "token": "test-token-123",
            "id": "00000000-0000-0000-0000-000000000001"
        },
        "baseUrl": "https://api.example.com",
        "policies": [
            {
                "route": "/api/users/*",
                "method": "GET",
                "action": "allow",
                "response_filter": {
                    "type": "exclude_fields",
                    "fields": ["ssn", "salary"]
                }
            },
            {
                "route": "/api/admin/*",
                "method": "*",
                "action": "deny"
            }
        ],
        "default_action": "allow"
    }'::jsonb,
    true,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;

-- Insert test token
INSERT INTO tokens (id, user_id, jti, token_hash, scopes, issued_at, expires_at)
VALUES (
    '00000000-0000-0000-0000-000000000020',
    '00000000-0000-0000-0000-000000000001',
    'test-jti-123',
    'test-token-hash-123',
    ARRAY['read', 'write'],
    NOW(),
    NOW() + INTERVAL '30 days'
)
ON CONFLICT (jti) DO NOTHING;

-- Insert rate limit for test token
INSERT INTO rate_limits (token_id, endpoint, limit_type, requests_limit, window_seconds)
VALUES (
    '00000000-0000-0000-0000-000000000020',
    '/api/*',
    'global',
    100,
    60
)
ON CONFLICT (token_id, endpoint, limit_type) DO NOTHING;
