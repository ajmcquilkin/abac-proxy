-- migrate:up
-- Add upstream_credential_id to policies (nullable initially)
ALTER TABLE policies ADD COLUMN upstream_credential_id UUID REFERENCES upstream_credentials(id);

-- Migrate existing policy tokens to upstream_credentials
INSERT INTO upstream_credentials (user_id, name, token)
SELECT
    p.user_id,
    'migrated-' || p.id::text,
    p.token
FROM policies p
WHERE p.token IS NOT NULL;

-- Update policies with their new credential references
UPDATE policies p
SET upstream_credential_id = uc.id
FROM upstream_credentials uc
WHERE uc.name = 'migrated-' || p.id::text
  AND uc.user_id = p.user_id;

-- Create downstream_tokens with plaintext tokens (testing phase)
-- TODO: Hash tokens for production
INSERT INTO downstream_tokens (policy_id, token_hash, name)
SELECT
    p.id,
    p.token,
    'migrated-token'
FROM policies p
WHERE p.token IS NOT NULL;

-- Make upstream_credential_id NOT NULL
ALTER TABLE policies ALTER COLUMN upstream_credential_id SET NOT NULL;

-- Drop the old token column
ALTER TABLE policies DROP COLUMN token;

-- migrate:down
-- WARNING: This rollback will cause data loss of downstream tokens
ALTER TABLE policies ADD COLUMN token TEXT;

-- Restore tokens from upstream_credentials
UPDATE policies p
SET token = uc.token
FROM upstream_credentials uc
WHERE uc.id = p.upstream_credential_id;

-- Make token NOT NULL
ALTER TABLE policies ALTER COLUMN token SET NOT NULL;

-- Drop credential reference
ALTER TABLE policies DROP COLUMN upstream_credential_id;

-- Drop new tables (CASCADE will remove downstream_tokens)
DROP TABLE downstream_tokens;
DROP TABLE upstream_credentials;
