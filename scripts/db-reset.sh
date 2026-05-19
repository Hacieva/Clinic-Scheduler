#!/bin/sh
set -e

cd "$(dirname "$0")/.."

echo "Stopping all containers and removing volumes..."
docker compose down -v

echo "Starting postgres..."
docker compose up -d postgres

echo "Waiting for postgres to be healthy..."
sleep 5

echo "Running migrations..."
cd backend
DATABASE_URL="postgres://${POSTGRES_USER:-clinic}:${POSTGRES_PASSWORD:-clinic_pass}@localhost:5432/${POSTGRES_DB:-clinic_db}?sslmode=disable" \
make migrate-up

echo "Database reset complete."
