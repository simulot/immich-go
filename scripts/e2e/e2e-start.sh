#!/bin/bash
# e2e-start.sh
# Spins up an Immich e2e test environment using existing provision and cleanup scripts.

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Spinning up Immich E2E Environment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Determine the directory of the current script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Call the provision script
echo -e "${YELLOW}🚀 Provisioning Immich instance...${NC}"
"${SCRIPT_DIR}/e2e-provision.sh" "$@"

echo ""
echo -e "${GREEN}✅ Immich E2E environment is ready!${NC}"
echo ""
echo -e "${BLUE}To clean up the environment, run:${NC}"
echo -e "  ${YELLOW}${SCRIPT_DIR}/e2e-cleanup.sh${NC}"
echo ""
