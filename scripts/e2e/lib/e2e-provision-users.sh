#!/bin/bash
# Create the e2e admin user and its API key in a running Immich instance.
#
# The credentials file is written with mode 600 and is never printed.

set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/common.sh"

usage() {
  cat <<'EOF'
Usage: e2e-provision-users.sh [IMMICH_URL] [OUTPUT_FILE]

Arguments:
  IMMICH_URL    running Immich API (default: http://localhost:2283)
  OUTPUT_FILE   credentials file (default: <install dir>/e2eusers.env)
EOF
}

case "${1:-}" in
  -h|--help) usage; exit 0 ;;
esac

IMMICH_URL="${1:-$(e2e_immich_url)}"
OUTPUT_FILE="${2:-${E2E_INSTALL_DIR}/e2eusers.env}"

e2e_require_cmd go
e2e_require_cmd curl

curl -sf "${IMMICH_URL}/api/server/ping" >/dev/null 2>&1 ||
  e2e_die "cannot reach Immich at ${IMMICH_URL}"

tool="${E2E_PROJECT_ROOT}/internal/e2e/e2eUtils/cmd/createUser"
[ -f "${tool}/createUser.go" ] ||
  e2e_die "user creation tool not found at ${tool}"

mkdir -p "$(dirname "${OUTPUT_FILE}")"
umask 077

# The Go tool reads the instance URL from e2e_url, like the tests do.
tmp="$(mktemp "${OUTPUT_FILE}.XXXXXX")"
if ! (
  cd "${tool}"
  e2e_url="${IMMICH_URL}" go run createUser.go create-admin
) >"${tmp}"; then
  cat "${tmp}" >&2
  rm -f "${tmp}"
  e2e_die "the user creation tool failed"
fi

if [ ! -s "${tmp}" ]; then
  rm -f "${tmp}"
  e2e_die "the user creation tool wrote no credentials"
fi

chmod 600 "${tmp}"
mv "${tmp}" "${OUTPUT_FILE}"

echo "credentials written to ${OUTPUT_FILE} (git-ignored, contents not shown)"
echo "users:"
sed -n 's/^E2E_\(.*\)_PASSWORD=.*/  - \1/p' "${OUTPUT_FILE}"
