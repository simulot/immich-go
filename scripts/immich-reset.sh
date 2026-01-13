#!/bin/bash
# immich-reset.sh
# Reset Immich data (keeps users and API keys) between test runs.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"
PROJECT_DIR="$(realpath "${SCRIPT_DIR}/..")"
INSTALL_DIR="${1:-${PROJECT_DIR}/internal/e2e/testdata/immich-server}"
TIMEOUT=60

info() { echo -e "${BLUE}$1${NC}"; }
success() { echo -e "${GREEN}$1${NC}"; }
warn() { echo -e "${YELLOW}$1${NC}"; }
fail() { echo -e "${RED}$1${NC}"; exit 1; }

ensure_env() {
  [ -d "${INSTALL_DIR}" ] || fail "Installation directory not found: ${INSTALL_DIR}"
  [ -f "${INSTALL_DIR}/docker-compose.yml" ] || fail "docker-compose.yml not found in ${INSTALL_DIR}"
}

find_port() {
  local port
  port=$(grep -E "^\s*-\s*['\"]?[0-9]+:2283['\"]?" "${INSTALL_DIR}/docker-compose.yml" | sed -E "s/.*['\"]?([0-9]+):2283.*/\1/" || true)
  echo "${port:-2283}"
}

reset_db() {
  info "🛑 Stopping Immich server"
  (cd "${INSTALL_DIR}" && docker compose stop immich-server) || fail "Failed to stop Immich server"

  info "🗑️  Resetting database"
  local sql
  sql='
DELETE FROM "stack" CASCADE;
DELETE FROM "library" CASCADE;
DELETE FROM "shared_link" CASCADE;
DELETE FROM "person" CASCADE;
DELETE FROM "album" CASCADE;
DELETE FROM "asset" CASCADE;
DELETE FROM "asset_face" CASCADE;
DELETE FROM "activity" CASCADE;
DELETE FROM "tag" CASCADE;
'

  if ! docker exec -i immich_postgres psql --dbname=immich --username=postgres -c "${sql}" >/dev/null 2>&1; then
    warn "⚠️  Default postgres container not found, scanning for alternatives"
    local container
    container=$(docker ps --format '{{.Names}}' | grep -E 'postgres|database' | head -n 1)
    [ -n "${container}" ] || fail "Could not find PostgreSQL container"
    docker exec -i "${container}" psql --dbname=immich --username=postgres -c "${sql}" || fail "Failed to reset database"
  fi
}

restart_server() {
  info "🚀 Restarting Immich server"
  (cd "${INSTALL_DIR}" && docker compose up -d immich-server) || fail "Failed to restart Immich server"
}

wait_for_api() {
  local port
  port=$(find_port)
  local url="http://localhost:${port}"
  local elapsed=0
  local ready=false

  info "⏳ Waiting for Immich API at ${url}"
  while [ ${elapsed} -lt ${TIMEOUT} ]; do
    if curl -sf "${url}/api/server/ping" >/dev/null 2>&1; then
      ready=true
      break
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done

  if [ "${ready}" = false ]; then
    (cd "${INSTALL_DIR}" && docker compose logs --tail=30 immich-server) || true
    fail "Immich API did not become ready in ${TIMEOUT} seconds"
  fi
}

main() {
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  info "  Immich Reset"
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo -e "  ${BLUE}Install Dir:${NC} ${INSTALL_DIR}"
  echo ""

  ensure_env
  reset_db
  restart_server
  wait_for_api

  echo ""
  success "✅ Immich reset complete"
}

main "$@"
