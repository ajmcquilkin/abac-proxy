-- Seed script for ABAC proxy sample data
-- Based on jsonplaceholder.typicode.com policy and allowlist

-- Clean up existing test data (cascade will clean downstream_tokens)
DELETE FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM upstream_credentials WHERE user_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000000';

-- Create sample user
INSERT INTO users (id, email, created_at, updated_at)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'demo@example.com',
  NOW(),
  NOW()
);

-- Create upstream credential (API token for jsonplaceholder)
INSERT INTO upstream_credentials (id, user_id, name, token, api_endpoint, created_at, updated_at)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  '00000000-0000-0000-0000-000000000000',
  'jsonplaceholder-api',
  'upstream-api-token',
  'https://jsonplaceholder.typicode.com',
  NOW(),
  NOW()
);

-- Create policy for jsonplaceholder API
INSERT INTO policies (id, user_id, upstream_credential_id, version, base_url, default_action, rules, is_active, created_at, updated_at)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '00000000-0000-0000-0000-000000000000',
  '22222222-2222-2222-2222-222222222222',
  '1.0',
  'https://jsonplaceholder.typicode.com',
  'deny',
  '[
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
  ]'::jsonb,
  true,
  NOW(),
  NOW()
);

-- Create downstream token (client authentication token)
-- Plaintext for testing (TODO: hash for production)
INSERT INTO downstream_tokens (id, policy_id, token_hash, name, created_at, updated_at)
VALUES (
  '33333333-3333-3333-3333-333333333333',
  '11111111-1111-1111-1111-111111111111',
  'test-token',
  'demo-client',
  NOW(),
  NOW()
);

-- Summary
SELECT
  'Seed completed!' as status,
  (SELECT COUNT(*) FROM users WHERE id = '00000000-0000-0000-0000-000000000000') as users_created,
  (SELECT COUNT(*) FROM upstream_credentials WHERE user_id = '00000000-0000-0000-0000-000000000000') as credentials_created,
  (SELECT COUNT(*) FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000') as policies_created,
  (SELECT COUNT(*) FROM downstream_tokens WHERE policy_id = '11111111-1111-1111-1111-111111111111') as tokens_created;
