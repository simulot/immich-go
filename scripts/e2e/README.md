# E2E Infrastructure Scripts

This directory contains bash scripts for managing Immich instances during e2e testing.

## Scripts

### immich-provision.sh
Provisions a fresh Immich instance for e2e testing.

**Usage:**
```bash
./scripts/e2e/immich-provision.sh [INSTALL_DIR] [PORT]
```

**Arguments:**
- `INSTALL_DIR`: Directory for Immich installation (default: `./internal/e2e/testdata/immich-server`)
- `PORT`: Port for Immich server (default: `2283`)

**What it does:**
1. Creates installation directory
2. Downloads latest docker-compose.yml and .env from Immich releases
3. Transforms docker-compose.yml to use internal Docker volumes
4. Pulls Docker images
5. Starts Immich services
6. Waits for API to become ready
7. Prints Immich URL

**Example:**
```bash
./scripts/e2e/immich-provision.sh ./internal/e2e/testdata/immich-server 3000
```

---

### immich-provision-users.sh
Provisions test users and API keys in an Immich instance.

**Usage:**
```bash
./scripts/e2e/immich-provision-users.sh [IMMICH_URL] [OUTPUT_FILE]
```

**Arguments:**
- `IMMICH_URL`: Immich server URL (default: `http://localhost:2283`)
- `OUTPUT_FILE`: Output file for credentials (default: `./internal/e2e/testdata/immich-server/e2eusers.yml`)

**What it does:**
1. Verifies Immich is accessible
2. Calls Go program to create users via API
3. Creates credentials file with user tokens and API keys

**Users created:**
- `admin@immich.app` - Admin with all permissions
- `user1@immich.app` - Regular user with minimal permissions
- `user2@immich.app` - Regular user with minimal permissions

**Example:**
```bash
./scripts/e2e/immich-provision-users.sh http://localhost:2283 ./internal/e2e/testdata/immich-server/e2eusers.yml
```

---

- `INSTALL_DIR`: Immich installation directory (default: `./internal/e2e/testdata/immich-server`)

**What it does:**
1. Stops Immich server container
2. Executes SQL to delete assets, albums, tags, etc.
3. Preserves users, API keys, and sessions
4. Restarts Immich server
5. Waits for API to become ready

**Example:**
./scripts/e2e/immich-reset.sh ./internal/e2e/testdata/immich-server
```

---

### immich-cleanup.sh
Complete cleanup of Immich e2e test instance (including root-owned volumes).

**Usage:**
```bash
./scripts/e2e/immich-cleanup.sh [INSTALL_DIR]
```

**Arguments:**
- `INSTALL_DIR`: Immich installation directory (default: `./internal/e2e/testdata/immich-server`)

**What it does:**
1. Stops and removes all containers
2. Removes Docker volumes (including root-owned)
3. Prunes Docker system
4. Removes installation directory (uses sudo if needed)

**Example:**
```bash
./scripts/e2e/immich-cleanup.sh ./internal/e2e/testdata/immich-server
```

---

## Complete Workflow

### Local Development
```bash
# 1. Provision Immich
./scripts/e2e/immich-provision.sh ./internal/e2e/testdata/immich-server 2283

# 2. Create users
./scripts/e2e/immich-provision-users.sh http://localhost:2283 ./internal/e2e/testdata/immich-server/e2eusers.yml

# 3. Run tests (explicitly setting env vars)
E2E_SERVER=http://localhost:2283 \
E2E_USERS=./internal/e2e/testdata/immich-server/e2eusers.yml \
go test -v -tags=e2e ./internal/e2e/client/...

# Or simply use defaults (same as above):
go test -v -tags=e2e ./internal/e2e/client/...

# 4. Reset between test runs
./scripts/e2e/immich-reset.sh ./internal/e2e/testdata/immich-server

# 5. Cleanup when done
./scripts/e2e/immich-cleanup.sh ./internal/e2e/testdata/immich-server
```

### GitHub Actions
See `.github/workflows/e2e-tests.yml` for CI usage.

---

## Requirements

- **Docker** and **Docker Compose** (v2)
- **curl** for downloading files
- **jq** (optional, for JSON processing)
- **Python 3** (optional, for docker-compose transformation)
- **Go** (for user provisioning)
- **sudo** (optional, for cleaning root-owned volumes)

---

## Troubleshooting

### Script fails with "Permission denied"
Make sure scripts are executable:
```bash
chmod +x scripts/e2e/*.sh
```

### Docker volumes can't be removed
The cleanup script will attempt to use sudo. If it fails:
```bash
sudo rm -rf ./internal/e2e/testdata/immich-server
```

### Immich API doesn't become ready
Check container logs:
```bash
docker compose -f ./internal/e2e/testdata/immich-server/docker-compose.yml logs immich-server
```

### Port already in use
Use a different port:
```bash
./scripts/e2e/immich-provision.sh ./internal/e2e/testdata/immich-server 3000
```

### Windows: bash not found
Use Git Bash or WSL. Ensure scripts have LF line endings:
```bash
dos2unix scripts/e2e/*.sh
```

---

## Notes

- Scripts use internal Docker volumes (not host paths) for portability
- Database resets preserve user accounts and API keys for speed
- Cleanup handles root-owned files automatically
- All scripts provide colored output and progress indication
- Scripts are idempotent where possible (safe to re-run)
