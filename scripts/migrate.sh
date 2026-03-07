#!/usr/bin/env bash
set -euo pipefail

# MediLink migration helper script
# Usage: ./scripts/migrate.sh [up|down|version|force VERSION]

DIRECTION="${1:-up}"
DATABASE_URL="${DATABASE_URL:?DATABASE_URL environment variable is required}"

cd "$(dirname "$0")/../backend"

migrate -path migrations -database "$DATABASE_URL" "$@"
