# Release Notes for v0.30.0

We're excited to announce the next major release of immich-go, bringing significant improvements to testing infrastructure, enhanced filtering capabilities, and better stability across the board. This release includes comprehensive end-to-end testing, new asset filtering options, and important bug fixes that improve the overall user experience.

## ✨ New Features

- **Enhanced Asset Filtering**: Added support for filtering assets by city, country, state, camera make, model, and tags in the from-immich command
- **Album and People Filtering**: New options to filter assets by specific albums, people, and exclude partner's photos
- **Search Suggestions**: New GetSearchSuggestions method for improved search functionality
- **Copy Asset Feature**: Introduced CopyAsset method for asset duplication (replacing deprecated ReplaceAsset)

## 🚀 Improvements

- **Unified Asset Tracking**: Implemented a new FileProcessor system for consistent asset tracking and event logging across all adapters
- **E2E Testing Framework**: Added comprehensive end-to-end tests for upload and archive commands with automated Immich server setup
- **Adapter Migration**: Migrated folder, Google Photos, and from-immich adapters to the new FileProcessor design for better performance and consistency
- **Error Handling**: Improved error handling in multipart uploads and server communications
- **CLI Consistency**: Standardized flag naming and added persistent flag support for better subcommand inheritance
- **Configuration**: Enhanced configuration file support with better serialization and validation
- **Documentation**: Updated command documentation with comprehensive examples and improved readability

## 🐛 Bug Fixes

- **Race Condition Fix**: Resolved race condition in uploadAsset function that could cause upload failures
- **E2E Test Stability**: Fixed various E2E test issues including server readiness checks, Windows compatibility, and user provisioning
- **API Compatibility**: Updated API calls to use newer endpoints and improved error handling
- **Date Range Validation**: Added validation to ensure date ranges are logically consistent
- **File Extension Handling**: Fixed MP4 extension handling and improved file type detection
- **CI/CD Improvements**: Enhanced workflow reliability and reduced false positives in automated checks

## 💥 Breaking Changes

- **Flag Renaming**: Renamed `--server-errors` flag to `--on-errors` for consistency
- **API Changes**: Updated replaceAsset method to use assets/copy endpoint instead of direct replacement
- **Configuration Structure**: Some internal configuration structures have changed for better serialization

## 🔧 Internal Changes

- **Code Refactoring**: Extensive refactoring to improve code maintainability and remove technical debt
- **Dependency Updates**: Updated Go dependencies and CI/CD tools to latest versions
- **Test Infrastructure**: Added E2E test tasks to pre-commit checks and improved test coverage
- **Documentation Generation**: Enhanced automated documentation generation and release notes process
- **CI/CD Optimization**: Streamlined workflows with conditional E2E testing and improved branch validation