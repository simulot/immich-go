# GitHub Actions Workflows

## Overview

This directory contains the CI/CD workflows for the immich-go project.

## Workflows

### 1. CI (`ci.yml`)
**Triggers:** Pull requests and pushes to `main`/`develop`

**Runs automatically on all PRs** - No secrets required.

**Jobs:**
- ✅ Branch name validation
- ✅ Code linting (golangci-lint)
- ✅ Unit tests with coverage
- ✅ Build verification

**Purpose:** Fast feedback for all contributors without exposing secrets.

---

### 2. E2E Tests (`ci-e2e.yml`)
**Triggers:** Pull requests, pushes, and manual dispatch

**Requires approval for external contributors** - Uses Tailscale secrets.

**Jobs:**
- 🔐 Approval check (automatic for trusted, manual for external)
- 🖥️ E2E server provisioning (Tailscale)
- 🐧 Linux client E2E tests
- 🪟 Windows client E2E tests
- 🧹 Cleanup

**Security:**
- Trusted contributors (collaborators): E2E runs automatically
- External contributors: Requires `e2e-approved` label from maintainer

**For maintainers:** See [`E2E_APPROVAL_GUIDE.md`](../E2E_APPROVAL_GUIDE.md)

---

## CI Pipeline Flow

```
┌─────────────────────────────────────────────────────────────┐
│                     PR Opened/Updated                        │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ├─────────────────────────────────────┐
                      │                                     │
              ┌───────▼────────┐                   ┌────────▼────────┐
              │   CI Workflow   │                   │  E2E Workflow   │
              │   (ci.yml)      │                   │  (ci-e2e.yml)   │
              └───────┬────────┘                   └────────┬────────┘
                      │                                     │
         ┌────────────┼────────────┐                       │
         │            │            │                       │
    ┌────▼───┐  ┌────▼───┐  ┌────▼───┐            ┌──────▼──────┐
    │Validate│  │  Lint  │  │  Test  │            │Check Approval│
    │ Branch │  │  Code  │  │  & Cov │            │   & Changes  │
    └────┬───┘  └────┬───┘  └────┬───┘            └──────┬──────┘
         │            │            │                       │
         └────────────┴────────────┘               ┌───────┴───────┐
                      │                            │               │
                 ┌────▼────┐                  ┌────▼────┐    ┌────▼────┐
                 │  Build  │                  │ Trusted │    │External │
                 │  Check  │                  │  User   │    │Contrib. │
                 └─────────┘                  └────┬────┘    └────┬────┘
                                                   │              │
                                              ┌────▼────┐    ┌────▼─────┐
                                              │Run E2E  │    │Has Label?│
                                              │Tests    │    └────┬─────┘
                                              └─────────┘         │
                                                              ┌───┴────┐
                                                              │Yes│ No │
                                                              └┬──┴──┬─┘
                                                         ┌─────▼─┐   │
                                                         │Run E2E│   │
                                                         └───────┘   │
                                                                 ┌───▼────┐
                                                                 │Skip + │
                                                                 │Comment │
                                                                 └────────┘
```

## For Contributors

### Running Tests Locally

```bash
# Run linting
make lint
# or
golangci-lint run ./...

# Run unit tests
go test ./...

# Run unit tests with coverage
go test -cover ./...

# Run E2E tests (requires local Immich server)
go test -tags=e2e ./internal/e2e/client/...
```

### Expected CI Behavior

**For external contributors:**
1. ✅ Fast feedback (lint, test, build) runs immediately
2. ⏸️ E2E tests are skipped (requires maintainer approval)
3. Maintainer will review and add `e2e-approved` label if safe
4. ✅ E2E tests run after approval

**For repository collaborators:**
1. ✅ All checks run automatically (including E2E)

## For Maintainers

### Approving E2E Tests for External PRs

1. **Review the PR code thoroughly**
2. **Check for security concerns**
3. **Add the `e2e-approved` label:**

```bash
gh pr edit PR_NUMBER --add-label "e2e-approved"
```

4. **E2E tests run automatically**

See the complete guide: [`E2E_APPROVAL_GUIDE.md`](../E2E_APPROVAL_GUIDE.md)

### Creating the Label

If `e2e-approved` doesn't exist:

```bash
gh label create "e2e-approved" \
  --description "E2E tests approved by maintainer" \
  --color "0E8A16"
```

## Secrets Required

### For CI Workflow
None - Runs on all PRs without secrets

### For E2E Workflow
- `TS_OAUTH_CLIENT_ID` - Tailscale OAuth client ID
- `TS_OAUTH_SECRET` - Tailscale OAuth secret

⚠️ **These secrets are only exposed to:**
- Trusted contributors (repository collaborators)
- External PRs with `e2e-approved` label (after manual review)

## Troubleshooting

### Lint Failures
```bash
# Run linting locally to reproduce
golangci-lint run ./...
```

### Test Failures
```bash
# Run tests locally with verbose output
go test -v ./...

# Run specific test
go test -v -run TestName ./path/to/package
```

### E2E Tests Not Running
- **External contributor?** Wait for maintainer to add `e2e-approved` label
- **Trusted contributor?** Check collaborator status in repo settings
- **Only docs changed?** E2E tests are automatically skipped

### Manual E2E Trigger
Go to **Actions** → **E2E Tests** → **Run workflow**

---

For questions or issues with CI/CD, please open a GitHub issue or discussion.
