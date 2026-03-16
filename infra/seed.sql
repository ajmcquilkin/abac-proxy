-- Seed script for ABAC proxy sample data
-- Based on jsonplaceholder.typicode.com policy and allowlist

-- Clean up existing test data
DELETE FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM tokens WHERE user_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000000';

-- Create sample user
INSERT INTO users (id, email, role, created_at, updated_at)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'demo@example.com',
  'developer',
  NOW(),
  NOW()
);

-- Create policy for jsonplaceholder API
INSERT INTO policies (id, user_id, version, content, is_active, created_at, updated_at)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '00000000-0000-0000-0000-000000000000',
  '1.0',
  '{
    "version": "1.0",
    "user": {
      "token": "test-token",
      "id": "00000000-0000-0000-0000-000000000000"
    },
    "baseUrl": "https://jsonplaceholder.typicode.com",
    "policies": [
      {
        "route": "/users",
        "method": "GET",
        "action": "allow",
        "response_filter": {
          "type": "include_fields",
          "fields": ["[].id", "[].name", "[].address.city", "[].address.geo.*"]
        }
      },
      {
        "route": "/users/*",
        "method": "GET",
        "action": "allow",
        "response_filter": {
          "type": "include_fields",
          "fields": ["id", "name", "address.city", "address.geo.*"]
        }
      },
      {
        "route": "/posts",
        "method": "GET",
        "action": "allow",
        "response_filter": {
          "type": "include_fields",
          "fields": ["[].id", "[].title", "[].userId"]
        }
      },
      {
        "route": "/posts/*",
        "method": "GET",
        "action": "allow"
      }
    ],
    "default_action": "deny"
  }'::jsonb,
  true,
  NOW(),
  NOW()
);

-- Create sample token for the user
INSERT INTO tokens (id, user_id, jti, token_hash, scopes, issued_at, expires_at, last_used_at)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  '00000000-0000-0000-0000-000000000000',
  'test-token-jti',
  encode(sha256('test-token'::bytea), 'hex'),
  ARRAY['read', 'write'],
  NOW(),
  NOW() + INTERVAL '1 year',
  NULL
);

-- Add rate limits for the token
INSERT INTO rate_limits (id, token_id, endpoint, limit_type, requests_limit, window_seconds)
VALUES
  (
    '33333333-3333-3333-3333-333333333333',
    '22222222-2222-2222-2222-222222222222',
    '/users',
    'per_minute',
    60,
    60
  ),
  (
    '44444444-4444-4444-4444-444444444444',
    '22222222-2222-2222-2222-222222222222',
    '/posts',
    'per_minute',
    100,
    60
  );

-- Summary
SELECT
  'Seed completed!' as status,
  (SELECT COUNT(*) FROM users WHERE id = '00000000-0000-0000-0000-000000000000') as users_created,
  (SELECT COUNT(*) FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000') as policies_created,
  (SELECT COUNT(*) FROM tokens WHERE user_id = '00000000-0000-0000-0000-000000000000') as tokens_created,
  (SELECT COUNT(*) FROM rate_limits WHERE token_id = '22222222-2222-2222-2222-222222222222') as rate_limits_created;
