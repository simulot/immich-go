#!/bin/bash
# immich-provision-users.sh
# Provisions test users and API keys in Immich instance
#
# Usage: immich-provision-users.sh [IMMICH_URL] [OUTPUT_FILE]
#   IMMICH_URL: Immich server URL (default: http://localhost:2283)
#   OUTPUT_FILE: Output file for credentials (default: ./e2e-immich/e2eusers.yml)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
IMMICH_URL="${1:-http://localhost:2283}"
OUTPUT_FILE="${2:-./internal/e2e/testdata/immich-server/e2eusers.yml}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Immich User Provisioning${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BLUE}Immich URL:${NC} ${IMMICH_URL}"
echo -e "  ${BLUE}Output File:${NC} ${OUTPUT_FILE}"
echo ""

# Check if Immich is accessible
echo -e "${YELLOW}🔍 Checking Immich availability...${NC}"
if ! curl -sf "${IMMICH_URL}/api/server/ping" > /dev/null 2>&1; then
    echo -e "${RED}❌ Cannot reach Immich at ${IMMICH_URL}${NC}"
    echo -e "${YELLOW}Make sure Immich is running and accessible${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Immich is accessible${NC}"
echo ""

# Check if Go program exists
CREATE_USER_TOOL="${PROJECT_ROOT}/internal/e2e/e2eUtils/cmd/createUser"
if [ ! -f "${CREATE_USER_TOOL}/createUser.go" ]; then
    echo -e "${RED}❌ User creation tool not found at ${CREATE_USER_TOOL}${NC}"
    exit 1
fi

# Convert OUTPUT_FILE to absolute path to avoid issues when changing directories
if [[ ! "${OUTPUT_FILE}" = /* ]]; then
    OUTPUT_FILE="$(pwd)/${OUTPUT_FILE}"
fi

# Run the Go program to create admin user
echo -e "${YELLOW}👥 Creating admin user...${NC}"
cd "${CREATE_USER_TOOL}"

if ! go run createUser.go create-admin > "${OUTPUT_FILE}"; then
    echo -e "${RED}❌ Failed to create admin user${NC}"
    exit 1
fi

# Display the created credentials
cat "${OUTPUT_FILE}"

# Verify output file was created
if [ ! -f "${OUTPUT_FILE}" ]; then
    echo -e "${RED}❌ Output file was not created: ${OUTPUT_FILE}${NC}"
    exit 1
fi

# Success!
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Admin user created successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BLUE}Credentials file:${NC} ${OUTPUT_FILE}"
echo ""
echo -e "${YELLOW}Created user:${NC}"
echo -e "  • ${BLUE}admin@immich.app${NC} (admin, all permissions)"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Run tests with credentials from: ${BLUE}${OUTPUT_FILE}${NC}"
echo ""
