#!/usr/bin/env bash
set -e

echo "Starting Postgres..."
docker compose up -d

echo "Applying migrations..."
docker exec -i social-network-main-db-1 psql -U main_user -d main_db -f /migrations/001_init.sql

echo "Starting Go server..."
DATABASE_URL="postgres://main_user:main_pass@localhost:5433/main_db?sslmode=disable" \
go run ./cmd/api
