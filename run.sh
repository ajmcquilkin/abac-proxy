#!/bin/bash
set -e

# Usage: ./run.sh [--migrate] [--seed]
#   --migrate: Run database migrations
#   --seed: Reset and seed the database with test data

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

# Only run migrations if --migrate flag is provided
if [[ "$*" == *"--migrate"* ]]; then
  echo "→ Running migrations..."
  DBMATE_MIGRATIONS_DIR="./infra/migrations" dbmate up
  echo ""
fi

# Only seed if --seed flag is provided
if [[ "$*" == *"--seed"* ]]; then
  echo "→ Seeding database..."
  psql "$DATABASE_URL" -f infra/seed.sql
  echo ""
fi

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
