#!/bin/bash
# Remove the e2e Immich instance and its volumes.
#
# Only the volumes of the e2e compose project are removed. Other containers,
# images and volumes on the machine are left untouched.

set -euo pipefail
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/common.sh"

usage() {
  cat <<'EOF'
Usage: e2e-cleanup.sh [options]

Options:
  --yes               do not ask for confirmation (for scripts)
  --install-dir DIR   installation directory
                      (default: internal/e2e/testdata/immich-server)
  -h, --help          show this help
EOF
}

yes=false
while [ $# -gt 0 ]; do
  case "$1" in
    --yes) yes=true; shift ;;
    --install-dir) [ $# -ge 2 ] || e2e_die "--install-dir needs a value"; E2E_INSTALL_DIR="$2"; shift 2 ;;
    -h|--help) usage; exit 0 ;;
    *) usage >&2; e2e_die "unknown argument: $1" ;;
  esac
done
export E2E_INSTALL_DIR

if [ ! -d "${E2E_INSTALL_DIR}" ]; then
  e2e_info "nothing to clean up: ${E2E_INSTALL_DIR} does not exist"
  exit 0
fi

if [ "${yes}" != true ]; then
  if [ -t 0 ]; then
    read -r -p "Remove the e2e instance in ${E2E_INSTALL_DIR} and its volumes? [y/N] " answer
    case "${answer}" in
      y|Y|yes|YES) ;;
      *) e2e_info "aborted"; exit 0 ;;
    esac
  else
    e2e_die "refusing to remove ${E2E_INSTALL_DIR} without --yes"
  fi
fi

e2e_require_docker

if [ -f "${E2E_INSTALL_DIR}/docker-compose.yml" ]; then
  e2e_info "stopping the instance (project ${E2E_PROJECT})"
  e2e_compose down --volumes --remove-orphans ||
    e2e_die "docker compose down failed; the directory was not removed"
fi

# Compose leaves behind named volumes that no service mounts any more (rootless
# mode replaces the model cache with a bind mount). Remove the volumes of this
# project only, selected by their compose label, never by name pattern.
leftovers="$(docker volume ls -q --filter "label=com.docker.compose.project=${E2E_PROJECT}")"
if [ -n "${leftovers}" ]; then
  e2e_info "removing the remaining volumes of project ${E2E_PROJECT}"
  for volume in ${leftovers}; do
    docker volume rm "${volume}" >/dev/null || true
  done
fi

e2e_info "removing ${E2E_INSTALL_DIR}"
if ! rm -rf "${E2E_INSTALL_DIR}" 2>/dev/null; then
  e2e_die "could not remove ${E2E_INSTALL_DIR}; the container created some files as root.
       Run: sudo rm -rf '${E2E_INSTALL_DIR}'"
fi

e2e_info "done"
