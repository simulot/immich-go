#!/bin/bash
# immich-provision.sh
# Provision a fresh Immich instance for local/e2e testing, create an admin user,
# and update VS Code launch configuration with the admin API key.

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
IMMICH_PORT="${2:-2283}"
IMMICH_URL="http://localhost:${IMMICH_PORT}"
USERS_FILE="${INSTALL_DIR}/e2eusers.env"
LAUNCH_FILE="${PROJECT_DIR}/.vscode/launch.json"
COMPOSE_URL="https://github.com/immich-app/immich/releases/latest/download/docker-compose.yml"
ENV_URL="https://github.com/immich-app/immich/releases/latest/download/example.env"
TIMEOUT=180  # seconds to wait for API

info() { echo -e "${BLUE}$1${NC}"; }
success() { echo -e "${GREEN}$1${NC}"; }
warn() { echo -e "${YELLOW}$1${NC}"; }
fail() { echo -e "${RED}$1${NC}"; exit 1; }

ensure_clean_dir() {
  if [ -d "${INSTALL_DIR}" ]; then
    warn "🧹 Cleaning existing Immich install at ${INSTALL_DIR}"
    if [ -f "${INSTALL_DIR}/docker-compose.yml" ]; then
      (cd "${INSTALL_DIR}" && docker compose down --volumes --remove-orphans >/dev/null 2>&1) || true
    fi
    docker system prune -f >/dev/null 2>&1 || true
    if ! rm -rf "${INSTALL_DIR}" 2>/dev/null; then
      if command -v sudo >/dev/null 2>&1; then
        sudo rm -rf "${INSTALL_DIR}" || true
      fi
    fi
  fi
  mkdir -p "${INSTALL_DIR}"
}

download_immich_artifacts() {
  info "📥 Downloading docker-compose.yml"
  curl -fsSL "${COMPOSE_URL}" -o "${INSTALL_DIR}/docker-compose.yml" || fail "Failed to download docker-compose.yml"

  info "📥 Downloading .env"
  curl -fsSL "${ENV_URL}" -o "${INSTALL_DIR}/.env" || fail "Failed to download .env"

  if [ "${IMMICH_PORT}" != "2283" ]; then
    {
      echo "# E2E Test Override"
      echo "IMMICH_PORT=${IMMICH_PORT}"
    } >> "${INSTALL_DIR}/.env"
  fi

  info "📝 Writing pgAdmin compose override"
  cat > "${INSTALL_DIR}/docker-compose-pgadmin.yml" << 'EOF'
name: immich

services:
  pgadmin:
    image: dpage/pgadmin4
    container_name: pgadmin4_container
    restart: always
    ports:
      - "8888:80"
    environment:
      PGADMIN_DEFAULT_EMAIL: admin@immich.app
      PGADMIN_DEFAULT_PASSWORD: admin
    volumes:
      - pgadmin-data:/var/lib/pgadmin

volumes:
  pgadmin-data:
EOF
}

start_stack() {
  info "🐳 Pulling Docker images"
  (cd "${INSTALL_DIR}" && docker compose -f docker-compose.yml -f docker-compose-pgadmin.yml pull -q) || fail "Failed to pull Docker images"

  info "🚀 Starting Immich stack"
  if ! (cd "${INSTALL_DIR}" && docker compose -f docker-compose.yml -f docker-compose-pgadmin.yml up -d --build --renew-anon-volumes --force-recreate --remove-orphans); then
    (cd "${INSTALL_DIR}" && docker compose logs --tail=50) || true
    fail "Failed to start Immich services"
  fi
}

wait_for_api() {
  info "⏳ Waiting for Immich API at ${IMMICH_URL}"
  local elapsed=0
  local ready=false
  while [ ${elapsed} -lt ${TIMEOUT} ]; do
    if curl -sf "${IMMICH_URL}/api/server/ping" >/dev/null 2>&1; then
      ready=true
      break
    fi
    printf "  Still waiting... (%ss / %ss)\n" "${elapsed}" "${TIMEOUT}"
    sleep 2
    elapsed=$((elapsed + 2))
  done

  if [ "${ready}" = false ]; then
    (cd "${INSTALL_DIR}" && docker compose ps) || true
    (cd "${INSTALL_DIR}" && docker compose logs --tail=50 immich-server) || true
    fail "Immich API did not become ready in ${TIMEOUT} seconds"
  fi
}

create_admin_user() {
  local tool_dir="${PROJECT_DIR}/internal/e2e/e2eUtils/cmd/createUser"
  if [ ! -f "${tool_dir}/createUser.go" ]; then
    fail "User creation tool not found at ${tool_dir}"
  fi

  mkdir -p "$(dirname "${USERS_FILE}")"
  info "👥 Creating admin user and API key"
  (cd "${tool_dir}" && go run createUser.go create-admin > "${USERS_FILE}") || fail "Failed to create admin user"
}

update_launch_config() {
  if [ ! -f "${USERS_FILE}" ]; then
    warn "⚠️  User credentials not found at ${USERS_FILE}; skipping launch.json update"
    return
  fi

  local api_key
  api_key=$(grep "E2E_admin@immich.app_APIKEY" "${USERS_FILE}" | cut -d'=' -f2 || true)
  if [ -z "${api_key}" ]; then
    warn "⚠️  Admin API key not found in ${USERS_FILE}; skipping launch.json update"
    return
  fi

  if [ ! -f "${LAUNCH_FILE}" ]; then
    warn "ℹ️  launch.json not found at ${LAUNCH_FILE}; skipping update"
    return
  fi

  info "🔑 Updating VS Code launch configuration"
  if ! sed -i -E "s/--api-key=([A-Za-z0-9]+)/--api-key=${api_key}/" "${LAUNCH_FILE}"; then
    warn "⚠️  Failed to update launch.json"
  fi
}

print_summary() {
  echo ""
  success "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  success "✅ Immich provisioned successfully"
  success "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo -e "  ${BLUE}Immich URL:${NC} ${IMMICH_URL}"
  echo -e "  ${BLUE}Install Dir:${NC} ${INSTALL_DIR}"
  echo -e "  ${BLUE}Credentials:${NC} ${USERS_FILE}"
  echo -e "  ${BLUE}pgAdmin:${NC} http://localhost:8888 (admin@immich.app / admin)"
}

main() {
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  info "  Immich Provision"
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo -e "  ${BLUE}Install Dir:${NC} ${INSTALL_DIR}"
  echo -e "  ${BLUE}Port:${NC} ${IMMICH_PORT}"
  echo ""

  ensure_clean_dir
  download_immich_artifacts
  start_stack
  wait_for_api
  create_admin_user
  update_launch_config
  print_summary
}

main "$@"
