-- Test data for local development

-- Insert test user
INSERT INTO users (id, email, created_at, updated_at)
VALUES ('00000000-0000-0000-0000-000000000001', 'test@example.com', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Insert test policy
INSERT INTO policies (id, user_id, token, version, base_url, default_action, rules, is_active, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000010',
    '00000000-0000-0000-0000-000000000001',
    'test-token-123',
    'v1.0.0',
    'https://api.example.com',
    'allow',
    '[
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
    ]'::jsonb,
    true,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO NOTHING;
