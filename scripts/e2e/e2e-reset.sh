#!/bin/bash
# immich-reset.sh
# Resets Immich database between e2e tests (preserves users and API keys)
#
# Usage: immich-reset.sh [INSTALL_DIR]
#   INSTALL_DIR: Immich installation directory (default: ./e2e-immich)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="${1:-./internal/e2e/testdata/immich-server}"
TIMEOUT=60  # seconds to wait for API after restart

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Immich Database Reset${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if installation directory exists
if [ ! -d "${INSTALL_DIR}" ]; then
    echo -e "${RED}❌ Installation directory not found: ${INSTALL_DIR}${NC}"
    exit 1
fi

# Check if docker-compose.yml exists
if [ ! -f "${INSTALL_DIR}/docker-compose.yml" ]; then
    echo -e "${RED}❌ docker-compose.yml not found in ${INSTALL_DIR}${NC}"
    exit 1
fi

cd "${INSTALL_DIR}"

# Get the port from docker-compose or default to 2283
IMMICH_PORT=$(grep -E "^\s*-\s*['\"]?[0-9]+:2283['\"]?" docker-compose.yml | sed -E "s/.*['\"]?([0-9]+):2283.*/\1/" || echo "2283")
IMMICH_URL="http://localhost:${IMMICH_PORT}"

echo -e "${YELLOW}🛑 Stopping Immich server...${NC}"
if ! docker compose stop immich-server; then
    echo -e "${RED}❌ Failed to stop Immich server${NC}"
    exit 1
fi

echo -e "${YELLOW}🗑️  Resetting database...${NC}"

# SQL to reset the database (preserves users, API keys, sessions)
# Based on Immich utils.ts https://github.com/immich-app/immich/blob/853d19dc2dcfb871edc03860ecc94aadad3b478a/e2e/src/utils.ts#L147-L187
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

# Execute SQL in PostgreSQL container
if ! docker exec -i immich_postgres psql --dbname=immich --username=postgres -c "${RESET_SQL}" > /dev/null 2>&1; then
    echo -e "${RED}❌ Failed to reset database${NC}"
    echo -e "${YELLOW}Attempting alternative container name...${NC}"
    
    # Try alternative container name patterns
    CONTAINER_NAME=$(docker ps --format '{{.Names}}' | grep -E 'postgres|database' | head -n 1)
    if [ -z "${CONTAINER_NAME}" ]; then
        echo -e "${RED}❌ Could not find PostgreSQL container${NC}"
        exit 1
    fi
    
    echo -e "${YELLOW}Found container: ${CONTAINER_NAME}${NC}"
    if ! docker exec -i "${CONTAINER_NAME}" psql --dbname=immich --username=postgres -c "${RESET_SQL}"; then
        echo -e "${RED}❌ Failed to reset database${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}✅ Database reset complete${NC}"

echo -e "${YELLOW}🚀 Restarting Immich server...${NC}"
if ! docker compose up -d immich-server; then
    echo -e "${RED}❌ Failed to restart Immich server${NC}"
    exit 1
fi

# Wait for API to be ready
echo -e "${YELLOW}⏳ Waiting for Immich API...${NC}"
ELAPSED=0
READY=false

while [ $ELAPSED -lt $TIMEOUT ]; do
    if curl -sf "${IMMICH_URL}/api/server/ping" > /dev/null 2>&1; then
        READY=true
        break
    fi
    
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ "$READY" = false ]; then
    echo -e "${RED}❌ Immich API did not become ready within ${TIMEOUT} seconds${NC}"
    echo ""
    echo -e "${YELLOW}Recent logs:${NC}"
    docker compose logs --tail=30 immich-server
    exit 1
fi

# Success!
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Immich reset successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BLUE}Immich URL:${NC} ${IMMICH_URL}"
echo -e "  ${BLUE}Status:${NC} Ready for testing"
echo ""
