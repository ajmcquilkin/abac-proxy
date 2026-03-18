#!/bin/bash
set -e

# Usage: ./run.sh [--migrate] [--seed] [--db]
#   --migrate: Run database migrations
#   --seed: Reset and seed the database with test data
#   --db: Run in database mode (requires DATABASE_URL in .env)
#
# Default: runs in file mode using examples/ directory

load_env() {
  if [ -f .env ]; then
    set -a
    source .env
    set +a
  fi
}

require_db_url() {
  if [ -z "$DATABASE_URL" ]; then
    echo "DATABASE_URL not set in .env"
    exit 1
  fi
}

if [[ "$*" == *"--migrate"* ]]; then
  load_env
  require_db_url
  echo "Running migrations..."
  DBMATE_MIGRATIONS_DIR="./infra/migrations" dbmate up
  echo ""
fi

if [[ "$*" == *"--seed"* ]]; then
  load_env
  require_db_url
  echo "Seeding database..."
  psql "$DATABASE_URL" -f infra/seed.sql
  echo ""
fi

if [[ "$*" == *"--db"* ]]; then
  load_env
  require_db_url

  echo "Starting proxy (database mode)..."
  echo "  Port: 8080"
  echo "  Database: $DATABASE_URL"
  echo ""

  bazel run //cmd/proxy:proxy_dev -- \
    --port 8080 \
    --database-url "$DATABASE_URL"
else
  echo "Starting proxy (file mode)..."
  echo "  Port: 8080"
  echo "  Policy dir: examples/"
  echo ""
  echo "Test with:"
  echo "  curl -H 'Authorization: Bearer my-proxy-token' -H 'Host: jsonplaceholder.typicode.com' http://localhost:8080/users | jq"
  echo ""

  bazel run //cmd/proxy:proxy_dev -- \
    --port 8080 \
    --policy-group-dir "$(pwd)/examples" \
    --passthrough-unspecified
fi
