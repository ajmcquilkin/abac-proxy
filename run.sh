#!/bin/bash
set -e

# Load .env
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

if [ -z "$DATABASE_URL" ]; then
  echo "✗ DATABASE_URL not set in .env"
  exit 1
fi

echo "→ Seeding database..."
psql "$DATABASE_URL" -f infra/seed.sql

echo ""
echo "→ Starting proxy..."
echo "  Port: 8080"
echo "  Policy store: database"
echo "  Database: $DATABASE_URL"
echo ""
echo "Test with:"
echo "  curl -H 'Authorization: Bearer test-token' -H 'Host: jsonplaceholder.typicode.com' http://localhost:8080/users | jq"
echo ""

bazel run //cmd/proxy:proxy_dev -- \
  --port 8080 \
  --database-url "$DATABASE_URL" \
  --policy-store-type db \
  --allowlist "$(pwd)/allowlist.jsonplaceholder.json"
