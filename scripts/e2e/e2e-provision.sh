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
PROJECT_DIR="$(realpath "$(dirname "${BASH_SOURCE[0]:-$0}")/../..")"
INSTALL_DIR="${1:-${PROJECT_DIR}/internal/e2e/testdata/immich-server}"
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


# Clean up any existing instance
if [ -f "${INSTALL_DIR}/docker-compose.yml" ]; then
    cd "${INSTALL_DIR}"
    echo -e "${YELLOW}🧹 Cleaning up existing instance...${NC}"
    docker compose down --volumes --remove-orphans >/dev/null 2>&1|| true
    docker system prune -f >/dev/null 2>&1 || true
    sudo rm -rf "${INSTALL_DIR}" >/dev/null 2>&1
fi

# Create installation directory
echo -e "${YELLOW}📁 Creating installation directory...${NC}"
mkdir -p "${INSTALL_DIR}"
cd "${INSTALL_DIR}"

# Download docker-compose.yml
echo -e "${YELLOW}📥 Downloading docker-compose.yml...${NC}"
if ! curl -fsSL "${COMPOSE_URL}" -o docker-compose.yml; then
    echo -e "${RED}❌ Failed to download docker-compose.yml${NC}"
    exit 1
fi

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

# Create pgAdmin docker-compose file
echo -e "${YELLOW}📝 Creating pgAdmin docker-compose file...${NC}"
cat > docker-compose-pgadmin.yml << 'EOF'
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

# Pull Docker images
echo -e "${YELLOW}🐳 Pulling Docker images...${NC}"
if ! docker compose -f docker-compose.yml -f docker-compose-pgadmin.yml pull -q; then
    echo -e "${RED}❌ Failed to pull Docker images${NC}"
    exit 1
fi

# Start Immich services
echo -e "${YELLOW}🚀 Starting Immich services...${NC}"
if ! docker compose -f docker-compose.yml -f docker-compose-pgadmin.yml up -d --build --renew-anon-volumes --force-recreate --remove-orphans; then
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
    echo -e "  ${BLUE}⏱${NC}  Still waiting... (${ELAPSED}s / ${TIMEOUT}s)"
    
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

# Determine the directory of the current script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

#  Create a few test users 
"${SCRIPT_DIR}/lib/e2e-provision-users.sh" "${IMMICH_URL}" "${INSTALL_DIR}/e2eusers.env" 

# Success!
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ Immich provisioned successfully!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

echo -e "  ${BLUE}Immich URL:${NC} ${IMMICH_URL}"
echo -e "  ${BLUE}Installation Directory:${NC} ${INSTALL_DIR}"
echo -e "  ${BLUE}Database access:${NC} http://localhost:8888 (admin@immich.app / admin)"
echo -e "  ${BLUE}  Check the page${NC} https://docs.immich.app/guides/database-gui"
echo ""
echo -e "${YELLOW}Created users:${NC}"
grep "E2E_.*_PASSWORD" "${INSTALL_DIR}/e2eusers.env" | while IFS='=' read -r key value; do
    email=$(echo "$key" | sed 's/E2E_//;s/_PASSWORD//')
    echo -e "  ${BLUE}•${NC} $email / $value"
done
echo ""
