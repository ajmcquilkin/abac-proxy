#!/bin/bash
set -e

# Script to add a policy to the database from a JSON file
# Usage: ./scripts/add-policy.sh <policy-json> <user-id> <cred-name> <upstream-token> <downstream-token> <token-name> [token-type] [header-string]

if [ "$#" -lt 6 ] || [ "$#" -gt 8 ]; then
  echo "Usage: $0 <policy-json> <user-id> <cred-name> <upstream-token> <downstream-token> <token-name> [token-type] [header-string]"
  echo "Example (bearer): $0 policy.json user-id cred-name token client-token client"
  echo "Example (custom): $0 policy.browserbase.json user-id bb-api bb_live_xxx bb-token client custom x-bb-api-key"
  exit 1
fi

POLICY_FILE="$1"
USER_ID="$2"
CRED_NAME="$3"
UPSTREAM_TOKEN="$4"
DOWNSTREAM_TOKEN="$5"
TOKEN_NAME="$6"
TOKEN_TYPE="${7:-bearer}"
HEADER_STRING="${8:-}"

if [ ! -f "$POLICY_FILE" ]; then
  echo "✗ Policy file not found: $POLICY_FILE"
  exit 1
fi

# Load .env for DATABASE_URL
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

if [ -z "$DATABASE_URL" ]; then
  echo "✗ DATABASE_URL not set in .env"
  exit 1
fi

# Read policy JSON
POLICY_JSON=$(cat "$POLICY_FILE")
BASE_URL=$(echo "$POLICY_JSON" | jq -r '.baseUrl')
VERSION=$(echo "$POLICY_JSON" | jq -r '.version')
DEFAULT_ACTION=$(echo "$POLICY_JSON" | jq -r '.default_action')
RULES=$(echo "$POLICY_JSON" | jq -c '.policies')

# Verify user exists
USER_EMAIL=$(psql "$DATABASE_URL" -qtAX -c "SELECT email FROM users WHERE id = '$USER_ID'")

if [ -z "$USER_EMAIL" ]; then
  echo "✗ User not found with ID: $USER_ID"
  exit 1
fi

echo "→ Adding policy from $POLICY_FILE for $USER_EMAIL"

# Create upstream credential
if [ -n "$HEADER_STRING" ]; then
  CRED_ID=$(psql "$DATABASE_URL" -qtAX -c "
INSERT INTO upstream_credentials (user_id, name, token, api_endpoint, token_type, header_string)
VALUES ('$USER_ID', '$CRED_NAME', '$UPSTREAM_TOKEN', '$BASE_URL', '$TOKEN_TYPE', '$HEADER_STRING')
RETURNING id
")
else
  CRED_ID=$(psql "$DATABASE_URL" -qtAX -c "
INSERT INTO upstream_credentials (user_id, name, token, api_endpoint, token_type)
VALUES ('$USER_ID', '$CRED_NAME', '$UPSTREAM_TOKEN', '$BASE_URL', '$TOKEN_TYPE')
RETURNING id
")
fi

# Escape single quotes in JSON for SQL
RULES_ESCAPED=$(echo "$RULES" | sed "s/'/''/g")

# Create policy
POLICY_ID=$(psql "$DATABASE_URL" -qtAX -c "
INSERT INTO policies (user_id, upstream_credential_id, version, base_url, default_action, rules, is_active)
VALUES ('$USER_ID', '$CRED_ID', '$VERSION', '$BASE_URL', '$DEFAULT_ACTION', '$RULES_ESCAPED'::jsonb, true)
RETURNING id
")

# Create downstream token (plaintext for testing)
TOKEN_ID=$(psql "$DATABASE_URL" -qtAX -c "
INSERT INTO downstream_tokens (policy_id, token_hash, name)
VALUES ('$POLICY_ID', '$DOWNSTREAM_TOKEN', '$TOKEN_NAME')
RETURNING id
")

echo "✓ Policy added: $POLICY_ID"
echo "  Credential: $CRED_ID"
echo "  Token: $TOKEN_ID"
echo ""
echo "Test: curl -H 'Authorization: Bearer $DOWNSTREAM_TOKEN' -H 'Host: $(echo $BASE_URL | sed 's|https://||' | sed 's|http://||')' http://localhost:8080/path"
