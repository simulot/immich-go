# Release Process

This document describes the release process for immich-go, including both regular releases and pre-releases.

## Overview

immich-go follows a structured release process with two main branches:
- **`main`**: Contains stable, production-ready code for official releases
- **`develop`**: Contains the latest development code for upcoming releases

## Types of Releases

### 1. Official Releases
Official releases are created from the `main` branch and represent stable, production-ready versions.

**Process:**
1. Merge `develop` into `main` when ready for release
2. Create a version tag (e.g., `v1.0.0`)
3. GoReleaser automatically builds and publishes the release

### 2. Pre-releases
Pre-releases are created from the `develop` branch to allow testing of development versions.

**Types of pre-releases:**
- **Alpha**: Early development versions (`v1.0.0-alpha.1`)
- **Beta**: Feature-complete but may have bugs (`v1.0.0-beta.1`)
- **Release Candidate**: Near-final versions (`v1.0.0-rc.1`)

## Creating Pre-releases

### Method 1: GitHub Actions Workflow (Recommended)

The easiest way to create a pre-release is through the GitHub Actions workflow:

1. **Navigate to Actions**: Go to the [Actions tab](https://github.com/simulot/immich-go/actions) in the repository
2. **Select Workflow**: Choose "Create Pre-release" from the workflow list
3. **Run Workflow**: Click "Run workflow" and fill in:
   - **Branch**: Ensure `develop` is selected
   - **Version**: Enter the pre-release version (e.g., `v1.0.0-beta.1`)
   - **Draft**: Check if you want to create a draft release (optional)

4. **Automatic Process**: The workflow will:
   - ✅ Validate the version format
   - ✅ Check that the version tag doesn't already exist
   - ✅ Switch to the `develop` branch
   - ✅ Run tests (`go test ./...`)
   - ✅ Run linter (`golangci-lint run`)
   - ✅ Create and push the version tag
   - ✅ Build binaries for all platforms using GoReleaser
   - ✅ Create the GitHub release
   - ✅ Mark it as a pre-release
   - ✅ Generate changelog since the last release
   - ✅ Add appropriate release notes with warnings

### Method 2: Local Script

For maintainers who prefer a local approach:

```bash
# Navigate to the repository root
cd /path/to/immich-go

# Make sure you're on develop and up to date
git checkout develop
git pull origin develop

# Run the pre-release script
./scripts/create-prerelease.sh v1.0.0-beta.1

# Optional flags:
./scripts/create-prerelease.sh v1.0.0-beta.1 --draft      # Create as draft
./scripts/create-prerelease.sh v1.0.0-beta.1 --local-only # Run checks only
```

**Requirements:**
- [GitHub CLI](https://cli.github.com/) installed and authenticated
- `golangci-lint` installed (optional but recommended)
- Git repository on the `develop` branch

## Version Naming Conventions

### Official Releases
- `v1.0.0` - Major release
- `v1.1.0` - Minor release (new features)
- `v1.0.1` - Patch release (bug fixes)

### Pre-releases
- `v1.0.0-alpha.1` - Alpha version (early development)
- `v1.0.0-beta.1` - Beta version (feature complete, testing phase)
- `v1.0.0-rc.1` - Release candidate (near-final version)

## What Happens During a Pre-release

1. **Validation**: Version format and tag uniqueness are verified
2. **Testing**: All tests must pass
3. **Linting**: Code must pass linting checks
4. **Building**: GoReleaser builds binaries for:
   - Linux (amd64, arm64)
   - Windows (amd64, arm64)
   - macOS (amd64, arm64)
   - FreeBSD (amd64, arm64)
5. **Release Creation**: GitHub release is created with:
   - Pre-release flag set
   - Changelog since last release
   - Warning about pre-release status
   - Installation instructions
   - Link to feedback/issues

## Release Artifacts

Each release (pre-release or official) includes:
- Compiled binaries for multiple platforms
- Checksums file (`checksums.txt`)
- Source code archives
- Changelog and release notes

## Troubleshooting

### Common Issues

**Version already exists:**
```
Error: Tag 'v1.0.0-beta.1' already exists
```
Solution: Choose a different version number or delete the existing tag if appropriate.

**Tests failing:**
```
Tests failed
```
Solution: Fix the failing tests in the `develop` branch before creating the pre-release.

**Linter errors:**
```
Linter failed
```
Solution: Fix linting issues or run `golangci-lint run --fix` if auto-fixable.

**GitHub CLI not authenticated:**
```
GitHub CLI (gh) is not installed
```
Solution: Install and authenticate GitHub CLI:
```bash
# Install (varies by OS)
# Ubuntu/Debian:
sudo apt install gh

# macOS:
brew install gh

# Authenticate
gh auth login
```

### Manual Cleanup

If a pre-release creation fails partway through:

```bash
# Delete the tag locally and remotely
git tag -d v1.0.0-beta.1
git push origin --delete v1.0.0-beta.1

# Delete the GitHub release if it was created
gh release delete v1.0.0-beta.1
```

## Best Practices

1. **Test thoroughly**: Always run tests and linting before creating a pre-release
2. **Use descriptive versions**: Include meaningful pre-release identifiers
3. **Update changelog**: Ensure the automatic changelog generation captures important changes
4. **Communicate**: Announce pre-releases to testers and developers
5. **Gather feedback**: Use pre-releases to collect feedback before official releases
6. **Document breaking changes**: Clearly document any breaking changes in pre-releases

## Security Considerations

- Pre-releases are publicly available on GitHub
- Binaries are built in GitHub Actions environment
- All releases are signed with checksums for verification
- Never include sensitive information in release notes or changelogs