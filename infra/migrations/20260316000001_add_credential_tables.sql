-- migrate:up
-- Create upstream_credentials table for API tokens
CREATE TABLE upstream_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    token TEXT NOT NULL,
    api_endpoint TEXT,
    token_type VARCHAR(50) DEFAULT 'bearer',
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX idx_upstream_credentials_user_id ON upstream_credentials(user_id);

-- Create downstream_tokens table for client authentication
CREATE TABLE downstream_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES policies(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(policy_id, name)
);

-- Critical index for token lookup (hash + active status)
CREATE INDEX idx_downstream_tokens_hash_active ON downstream_tokens(token_hash)
WHERE revoked = FALSE;

CREATE INDEX idx_downstream_tokens_policy_id ON downstream_tokens(policy_id);

-- migrate:down
DROP TABLE downstream_tokens;
DROP TABLE upstream_credentials;
