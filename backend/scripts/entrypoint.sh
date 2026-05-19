#!/bin/sh
set -e

echo "Running database migrations..."
goose -dir /app/migrations postgres "$DATABASE_URL" up
echo "Migrations completed."

echo "Starting API server..."
exec /app/api
