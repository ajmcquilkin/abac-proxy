-- migrate:up
-- Drop unique constraint to allow multiple active policies per user
DROP INDEX IF EXISTS idx_policies_user_active;

-- migrate:down
-- Recreate unique constraint (requires only one active policy per user)
CREATE UNIQUE INDEX idx_policies_user_active ON policies(user_id) WHERE is_active = true;
