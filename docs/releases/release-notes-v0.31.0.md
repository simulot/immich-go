# Release Notes - v0.31.0

We're excited to share this update, which brings improvements to the CI/CD pipeline, enhanced asset tracking in the UI, and several important bug fixes to ensure a smoother experience.

## ✨ New Features

- **Enhanced Asset Tracking**: Introduced discovery and processing zones in the terminal UI, providing better visibility into asset processing status with size tracking and detailed counters.

## 🚀 Improvements

- **Improved CI/CD Pipeline**: 
  - Replaced API monitoring with comprehensive nightly E2E tests for better quality assurance.
  - Implemented a two-stage CI workflow with fast feedback checks and secure E2E testing.
  - Added approval system for E2E tests from external contributors to enhance security.
- **Better Globbing Behavior**: Enhanced file globbing patterns with improved error handling and clearer documentation. The system now gracefully continues when encountering filesystem errors during glob traversal.
- **Cleaner UI Layout**: Right-aligned size fields in the terminal UI processing status zone for improved readability.

## 🐛 Bug Fixes

- **Fixed `--folder-as-album=NONE` Conflict**: Resolved an issue where `--folder-as-album=NONE` would conflict with the `--into-album` option.
- **Documentation Corrections**: 
  - Updated documentation to show the correct value `NONE` (uppercase) instead of `none` for the `--folder-as-album` flag in configuration and environment files.
  - Corrected references from `--concurrent-uploads` to `--concurrent-tasks` throughout the documentation.
  - Clarified CI/CD workflows and job details in README.

## 🔧 Internal Changes

- **Code Cleanup**: 
  - Removed legacy journal structure and unified FileProcessor usage across the codebase.
  - Simplified the reporting system for better maintainability.
  - Migrated legacy file event codes to the new standardized event system.
  - Cleaned up test cases with more descriptive names and streamlined assertions.
- **CI/CD Maintenance**: 
  - Removed deprecated workflow files and outdated documentation.
  - Simplified branching model in contribution guidelines.
  - Fixed workflow failures when checking for non-document file changes in pull requests.
  - Improved script robustness to handle empty diffs and prevent false failures.
  - Corrected jq expressions in PR check workflows.
