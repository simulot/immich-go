#!/bin/bash
# immich-cleanup.sh
# Complete cleanup of Immich e2e test instance (including root-owned volumes)
#
# Usage: immich-cleanup.sh [INSTALL_DIR]
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

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Immich Complete Cleanup${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if installation directory exists
if [ ! -d "${INSTALL_DIR}" ]; then
    echo -e "${YELLOW}⚠️  Installation directory not found: ${INSTALL_DIR}${NC}"
    echo -e "${GREEN}✅ Nothing to clean up${NC}"
    exit 0
fi

cd "${INSTALL_DIR}"

# Stop and remove containers
if [ -f "docker-compose.yml" ]; then
    echo -e "${YELLOW}🛑 Stopping Immich containers...${NC}"
    docker compose down --volumes --remove-orphans 2>/dev/null || {
        echo -e "${YELLOW}⚠️  docker compose down failed, continuing...${NC}"
    }
else
    echo -e "${YELLOW}⚠️  No docker-compose.yml found, skipping container cleanup${NC}"
fi

# Get volume names that might be created
VOLUMES=$(docker volume ls --format '{{.Name}}' | grep -E 'immich|e2e' || true)

if [ -n "${VOLUMES}" ]; then
    echo -e "${YELLOW}🗑️  Removing Docker volumes...${NC}"
    echo "${VOLUMES}" | while read -r volume; do
        echo -e "  ${BLUE}•${NC} Removing volume: ${volume}"
        docker volume rm "${volume}" 2>/dev/null || {
            echo -e "    ${YELLOW}⚠️  Could not remove ${volume}, may be in use${NC}"
        }
    done
fi

# Prune Docker system
echo -e "${YELLOW}🧹 Pruning Docker system...${NC}"
docker system prune -f 2>/dev/null || {
    echo -e "${YELLOW}⚠️  docker system prune failed, continuing...${NC}"
}

# Remove installation directory
# Check for files that might be root-owned
cd ..
INSTALL_DIR_BASENAME=$(basename "${INSTALL_DIR}")

if [ -d "${INSTALL_DIR_BASENAME}" ]; then
    echo -e "${YELLOW}📁 Removing installation directory...${NC}"
    
    # Try normal removal first
    if rm -rf "${INSTALL_DIR_BASENAME}" 2>/dev/null; then
        echo -e "${GREEN}✅ Directory removed successfully${NC}"
    else
        # If normal removal fails, might need sudo for root-owned files
        echo -e "${YELLOW}⚠️  Some files are root-owned, attempting with sudo...${NC}"
        
        if command -v sudo >/dev/null 2>&1; then
            if sudo rm -rf "${INSTALL_DIR_BASENAME}"; then
                echo -e "${GREEN}✅ Directory removed successfully (with sudo)${NC}"
            else
                echo -e "${RED}❌ Failed to remove directory even with sudo${NC}"
                echo -e "${YELLOW}You may need to manually remove: ${INSTALL_DIR_BASENAME}${NC}"
            fi
        else
            echo -e "${RED}❌ sudo not available, cannot remove root-owned files${NC}"
            echo -e "${YELLOW}You may need to manually remove: ${INSTALL_DIR_BASENAME}${NC}"
        fi
    fi
fi

# Success!
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Immich cleanup complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${YELLOW}Cleaned up:${NC}"
echo -e "  • ${BLUE}Stopped and removed containers${NC}"
echo -e "  • ${BLUE}Removed Docker volumes${NC}"
echo -e "  • ${BLUE}Pruned Docker system${NC}"
echo -e "  • ${BLUE}Removed installation directory${NC}"
echo ""
