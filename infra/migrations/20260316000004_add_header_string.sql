-- migrate:up
-- Add header_string column for custom authentication headers
ALTER TABLE upstream_credentials ADD COLUMN header_string TEXT;

-- Update token_type to be more explicit
UPDATE upstream_credentials SET token_type = 'bearer' WHERE token_type IS NULL OR token_type = 'bearer';

-- migrate:down
ALTER TABLE upstream_credentials DROP COLUMN header_string;
