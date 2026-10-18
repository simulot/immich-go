#!/bin/bash
# Reset the e2e Immich data between test runs. Users and API keys are kept.
#
# The database container is resolved from the e2e compose project. The script
# never acts on another container.

set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/common.sh"

usage() {
  cat <<'EOF'
Usage: e2e-reset.sh [options]

Options:
  --install-dir DIR   installation directory
                      (default: internal/e2e/testdata/immich-server)
  -h, --help          show this help
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --install-dir) [ $# -ge 2 ] || e2e_die "--install-dir needs a value"; E2E_INSTALL_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; e2e_die "unknown argument: $1" ;;
  esac
done
export E2E_INSTALL_DIR

e2e_require_docker
e2e_require_cmd curl

[ -f "${E2E_INSTALL_DIR}/docker-compose.yml" ] ||
  e2e_die "no e2e instance in ${E2E_INSTALL_DIR}; run scripts/e2e/e2e-provision.sh first"

container="$(e2e_compose ps --status running -q database)"
[ -n "${container}" ] ||
  e2e_die "the database service of project ${E2E_PROJECT} is not running; re-run scripts/e2e/e2e-provision.sh"

# Keep users, API keys and sessions. Based on Immich's own e2e reset:
# https://github.com/immich-app/immich/blob/853d19dc2dcfb871edc03860ecc94aadad3b478a/e2e/src/utils.ts#L147-L187
RESET_SQL='
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

e2e_info "stopping the Immich server"
e2e_compose stop immich-server

e2e_info "resetting the database"
docker exec -i "${container}" psql --dbname=immich --username=postgres -c "${RESET_SQL}" >/dev/null ||
  e2e_die "the database reset failed"

e2e_info "restarting the Immich server"
e2e_compose up -d immich-server

if ! e2e_wait_ready 120; then
  e2e_compose logs --tail=30 immich-server >&2 || true
  e2e_die "Immich did not answer on $(e2e_immich_url) after the reset"
fi

e2e_info "database reset, $(e2e_immich_url) is ready"
