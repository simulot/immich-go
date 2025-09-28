# Pre-release Workflow Setup

This setup provides an on-demand pre-release system for immich-go based on the develop branch.

## What's Been Created

### 1. GitHub Actions Workflow
**File**: `.github/workflows/pre-release.yml`

**Features**:
- ✅ Manual trigger with version input
- ✅ Validation of version format and uniqueness  
- ✅ Automated testing and linting
- ✅ Multi-platform builds (Linux, Windows, macOS, FreeBSD)
- ✅ Automatic pre-release creation with changelog
- ✅ Proper pre-release marking and warnings

**Usage**: Go to Actions → "Create Pre-release" → Run workflow

### 2. Local Pre-release Script
**File**: `scripts/create-prerelease.sh`

**Features**:
- ✅ Interactive pre-release creation
- ✅ Local validation and testing
- ✅ Automatic changelog generation
- ✅ GitHub CLI integration
- ✅ Draft release support
- ✅ Local-only testing mode

**Usage**: `./scripts/create-prerelease.sh v1.0.0-beta.1`

### 3. VS Code Tasks
**File**: `.vscode/tasks.json` (updated)

**New Tasks**:
- `Release: Test Pre-release Script` - Run local checks only
- `Release: Create Pre-release (Local)` - Create actual pre-release locally

### 4. Documentation Updates
**Files**:
- `CONTRIBUTING.md` - Added pre-release section
- `docs/misc/release-process.md` - Complete release guide
- `docs/misc/README.md` - Updated index

## Quick Start

### Method 1: GitHub Actions (Recommended)
1. Go to [Actions](../../actions) in GitHub
2. Select "Create Pre-release"
3. Click "Run workflow"
4. Enter version (e.g., `v1.0.0-beta.1`)
5. Wait for completion

### Method 2: Local Script
```bash
# Test locally first
./scripts/create-prerelease.sh v1.0.0-beta.1 --local-only

# Create actual pre-release
./scripts/create-prerelease.sh v1.0.0-beta.1
```

### Method 3: VS Code Tasks
1. `Ctrl+Shift+P` → "Tasks: Run Task"
2. Select "Release: Test Pre-release Script"
3. Enter version when prompted

## Version Format

Use semantic versioning with pre-release identifiers:
- Alpha: `v1.0.0-alpha.1`
- Beta: `v1.0.0-beta.1` 
- RC: `v1.0.0-rc.1`

## Requirements

For local script usage:
- GitHub CLI (`gh`) installed and authenticated
- `golangci-lint` installed (optional)
- Repository on `develop` branch

## What Happens During Pre-release

1. **Validation**: Version format and tag uniqueness
2. **Testing**: `go test ./...`
3. **Linting**: `golangci-lint run`
4. **Building**: Multi-platform binaries via GoReleaser
5. **Release**: GitHub release with changelog and warnings

## Features

- 🛡️ **Safe**: Multiple validation steps prevent errors
- 🚀 **Fast**: Parallel builds for all platforms
- 📝 **Automatic**: Changelog generation and release notes
- 🔄 **Flexible**: GitHub Actions or local script
- 📋 **Documented**: Complete process documentation
- 🧪 **Testable**: Local-only mode for validation

## Troubleshooting

See `docs/misc/release-process.md` for detailed troubleshooting guide.

## Security

- All builds run in GitHub Actions secure environment
- Checksums generated for all artifacts
- Pre-releases clearly marked with warnings
- No sensitive information in release notes