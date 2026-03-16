-- Seed script for ABAC proxy sample data
-- Based on jsonplaceholder.typicode.com policy and allowlist

-- Clean up existing test data
DELETE FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000';
DELETE FROM users WHERE id = '00000000-0000-0000-0000-000000000000';

-- Create sample user
INSERT INTO users (id, email, created_at, updated_at)
VALUES (
  '00000000-0000-0000-0000-000000000000',
  'demo@example.com',
  NOW(),
  NOW()
);

-- Create policy for jsonplaceholder API
INSERT INTO policies (id, user_id, token, version, base_url, default_action, rules, is_active, created_at, updated_at)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  '00000000-0000-0000-0000-000000000000',
  'test-token',
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

-- Summary
SELECT
  'Seed completed!' as status,
  (SELECT COUNT(*) FROM users WHERE id = '00000000-0000-0000-0000-000000000000') as users_created,
  (SELECT COUNT(*) FROM policies WHERE user_id = '00000000-0000-0000-0000-000000000000') as policies_created;
