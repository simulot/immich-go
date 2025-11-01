#!/bin/bash
# immich-provision.sh
# Provisions a fresh Immich instance for e2e testing
#
# Usage: immich-provision.sh [INSTALL_DIR] [PORT]
#   INSTALL_DIR: Directory for Immich installation (default: ./e2e-immich)
#   PORT: Port for Immich server (default: 2283)

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="${1:-./e2e-immich}"
IMMICH_PORT="${2:-2283}"
COMPOSE_URL="https://github.com/immich-app/immich/releases/latest/download/docker-compose.yml"
ENV_URL="https://github.com/immich-app/immich/releases/latest/download/example.env"
TIMEOUT=180  # seconds to wait for API

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  Immich E2E Provisioning${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BLUE}Installation Directory:${NC} ${INSTALL_DIR}"
echo -e "  ${BLUE}Port:${NC} ${IMMICH_PORT}"
echo ""

# Create installation directory
echo -e "${YELLOW}📁 Creating installation directory...${NC}"
mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

# Clean up any existing instance
if [ -f "docker-compose.yml" ]; then
    echo -e "${YELLOW}🧹 Cleaning up existing instance...${NC}"
    docker compose down --volumes --remove-orphans 2>/dev/null || true
    docker system prune -f 2>/dev/null || true
fi

# Download docker-compose.yml
echo -e "${YELLOW}📥 Downloading docker-compose.yml...${NC}"
if ! curl -fsSL "${COMPOSE_URL}" -o docker-compose.yml.tmp; then
    echo -e "${RED}❌ Failed to download docker-compose.yml${NC}"
    exit 1
fi

# Transform docker-compose.yml to use internal volumes
echo -e "${YELLOW}🔧 Transforming docker-compose.yml...${NC}"
python3 - <<'PYTHON_SCRIPT' || sed -E 's|\$\{[^}]*LOCATION\}|local-volume|g; /local-volume:/a\  local-volume:' docker-compose.yml.tmp > docker-compose.yml
import sys
import re

try:
    with open('docker-compose.yml.tmp', 'r') as f:
        content = f.read()
    
    # Replace all *_LOCATION variables with internal volume
    pattern = r'\$\{[^}]*LOCATION\}'
    lines = content.split('\n')
    output_lines = []
    
    for line in lines:
        if '_LOCATION}' in line:
            # Add comment
            indent = len(line) - len(line.lstrip())
            output_lines.append(' ' * indent + '# immich-go e2e: using internal volume')
            line = re.sub(pattern, 'local-volume', line)
        output_lines.append(line)
    
    # Add volume definition if not present
    if 'volumes:' in content and 'local-volume:' not in content:
        output_lines.append('  local-volume:')
    
    with open('docker-compose.yml', 'w') as f:
        f.write('\n'.join(output_lines))
    
    print("docker-compose.yml transformed successfully", file=sys.stderr)
except Exception as e:
    print(f"Python transformation failed: {e}, falling back to sed", file=sys.stderr)
    sys.exit(1)
PYTHON_SCRIPT

# Verify transformation worked
if [ ! -f "docker-compose.yml" ] || [ ! -s "docker-compose.yml" ]; then
    echo -e "${RED}❌ docker-compose.yml transformation failed${NC}"
    exit 1
fi

rm -f docker-compose.yml.tmp

# Download .env file
echo -e "${YELLOW}📥 Downloading .env file...${NC}"
if ! curl -fsSL "${ENV_URL}" -o .env; then
    echo -e "${RED}❌ Failed to download .env file${NC}"
    exit 1
fi

# Override the port in .env if not default
if [ "${IMMICH_PORT}" != "2283" ]; then
    echo "# E2E Test Override" >> .env
    echo "IMMICH_PORT=${IMMICH_PORT}" >> .env
fi

# Pull Docker images
echo -e "${YELLOW}🐳 Pulling Docker images...${NC}"
if ! docker compose pull -q; then
    echo -e "${RED}❌ Failed to pull Docker images${NC}"
    exit 1
fi

# Start Immich services
echo -e "${YELLOW}🚀 Starting Immich services...${NC}"
if ! docker compose up -d --build --renew-anon-volumes --force-recreate --remove-orphans; then
    echo -e "${RED}❌ Failed to start Immich services${NC}"
    docker compose logs
    exit 1
fi

# Wait for API to be ready
echo -e "${YELLOW}⏳ Waiting for Immich API to be ready...${NC}"
IMMICH_URL="http://localhost:${IMMICH_PORT}"
ELAPSED=0
READY=false

while [ $ELAPSED -lt $TIMEOUT ]; do
    if curl -sf "${IMMICH_URL}/api/server/ping" > /dev/null 2>&1; then
        READY=true
        break
    fi
    
    # Show progress every 5 seconds
    if [ $((ELAPSED % 5)) -eq 0 ]; then
        echo -e "  ${BLUE}⏱${NC}  Still waiting... (${ELAPSED}s / ${TIMEOUT}s)"
    fi
    
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ "$READY" = false ]; then
    echo -e "${RED}❌ Immich API did not become ready within ${TIMEOUT} seconds${NC}"
    echo ""
    echo -e "${YELLOW}Container status:${NC}"
    docker compose ps
    echo ""
    echo -e "${YELLOW}Recent logs:${NC}"
    docker compose logs --tail=50 immich-server
    exit 1
fi

# Success!
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Immich provisioned successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${BLUE}Immich URL:${NC} ${IMMICH_URL}"
echo -e "  ${BLUE}Installation Directory:${NC} $(pwd)"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo -e "  1. Provision users: ${BLUE}./scripts/e2e/immich-provision-users.sh ${IMMICH_URL} ${INSTALL_DIR}/e2eusers.yml${NC}"
echo -e "  2. Run tests with: ${BLUE}E2E_SERVER=${IMMICH_URL} E2E_USERS=${INSTALL_DIR}/e2eusers.yml${NC}"
echo ""
