#!/bin/bash
# immich-cleanup.sh
# Stop Immich stack, remove volumes, and delete the installation directory.

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

info() { echo -e "${BLUE}$1${NC}"; }
success() { echo -e "${GREEN}$1${NC}"; }
warn() { echo -e "${YELLOW}$1${NC}"; }

main() {
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  info "  Immich Cleanup"
  info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo -e "  ${BLUE}Install Dir:${NC} ${INSTALL_DIR}"
  echo ""

  if [ ! -d "${INSTALL_DIR}" ]; then
    warn "⚠️  Install directory not found; nothing to clean"
    return 0
  fi

  cd "${INSTALL_DIR}"

  if [ -f "docker-compose.yml" ]; then
    info "🛑 Stopping Immich containers"
    docker compose down --volumes --remove-orphans 2>/dev/null || warn "⚠️  docker compose down failed"
  else
    warn "⚠️  docker-compose.yml missing; skipping container shutdown"
  fi

  local volumes
  volumes=$(docker volume ls --format '{{.Name}}' | grep -E 'immich|e2e' || true)
  if [ -n "${volumes}" ]; then
    info "🗑️  Removing Docker volumes"
    while read -r volume; do
      [ -z "${volume}" ] && continue
      docker volume rm "${volume}" 2>/dev/null || warn "⚠️  Could not remove volume ${volume}"
    done <<< "${volumes}"
  fi

  info "🧹 Pruning Docker system"
  docker system prune -f 2>/dev/null || warn "⚠️  docker system prune failed"

  cd ..
  local dir_name
  dir_name="$(basename "${INSTALL_DIR}")"
  info "📁 Removing install directory"
  if ! rm -rf "${dir_name}" 2>/dev/null; then
    if command -v sudo >/dev/null 2>&1; then
      sudo rm -rf "${dir_name}" || warn "⚠️  Failed to remove directory with sudo"
    else
      warn "⚠️  sudo not available; manual removal may be required"
    fi
  fi

  echo ""
  success "✅ Immich cleanup complete"
}

main "$@"
