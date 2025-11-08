# CI/CD Quick Reference Guide

## Active Workflows

### 1. Fast Feedback CI ⚡ (Primary)
**File:** `.github/workflows/fast-feedback.yml`

**When it runs:**
- ✅ Every pull request (with Go code changes)
- ✅ Every push to main/develop (with Go code changes)
- ✅ Manual trigger
- ⏭️ Skips docs-only changes

**What it does:**
```
validate (lint) ──┬→ test-linux ───┐
                  ├→ test-windows ─┤→ all-checks-passed ✅
                  └→ build-check ──┘
```

**Time:** ~3-5 minutes  
**Cost:** ~$0.04

---

### 2. E2E Tests 🧪 (Conditional)
**File:** `.github/workflows/e2e-tests.yml`

**When it runs:**
- ✅ Manual trigger (always)
- ✅ After Fast Feedback **succeeds** AND relevant files changed
- ❌ Never if Fast Feedback fails
- ⏭️ Skips if only docs changed
- ⏭️ Skips if no relevant code paths changed

**What it does:**
```
Fast Feedback CI succeeds ──→ should-run (checks files) ──→ run tests
                                      │
                                      └──→ skip (if no relevant changes)

should-run ──┬→ e2e_server (stays running)
             └→ e2e-linux ──→ e2e-windows ──→ clean-up
                                                   │
                    └──────────────────────────────┘
                    (creates done marker for server)
```

**Relevant paths for E2E:**
- `app/`, `adapters/`, `immich/`, `internal/`
- `main.go`, `go.mod`, `go.sum`
- `internal/e2e/testdata/immich-server/`, `.github/workflows/e2e-tests.yml`

**Time:** ~12-15 minutes  
**Cost:** ~$0.10

---

## Running CI Checks Locally

### Lint
```bash
golangci-lint run ./...
```

### Unit Tests
```bash
# Linux (with race detection)
go test -race -v -count=1 -coverprofile=coverage.out ./...

# Windows (without race detection)
go test -v -count=1 ./...
```

### Build
```bash
go build -o immich-go main.go
```

### All Checks
```bash
# Run the full suite
golangci-lint run ./... && \
go test -race -v -count=1 ./... && \
go build -o immich-go main.go && \
echo "✅ All checks passed!"
```

---

## Manual Workflow Triggers

### Trigger Fast Feedback CI
```bash
# Via GitHub UI:
# 1. Go to Actions tab
# 2. Select "Fast Feedback CI"
# 3. Click "Run workflow"
# 4. Select branch
# 5. Click "Run workflow"
```

### Trigger E2E Tests
```bash
# Via GitHub UI:
# 1. Go to Actions tab
# 2. Select "E2E Tests"
# 3. Click "Run workflow"
# 4. Select branch
# 5. Click "Run workflow"
```

---

## Understanding CI Results

### Fast Feedback CI

**✅ All jobs passed:**
- Your code is ready for review
- No action needed

**❌ validate failed:**
- Linting issues detected
- Run `golangci-lint run ./...` locally
- Fix issues and push

**❌ test-linux or test-windows failed:**
- Unit tests failed
- Run `go test -v ./...` locally
- Fix failing tests and push

**❌ build-check failed:**
- Code doesn't compile
- Run `go build main.go` locally
- Fix compilation errors and push

### E2E Tests

**✅ All jobs passed:**
- Integration tests passed
- Your changes work with real Immich server

**⏭️ Skipped:**
- Path filters excluded your changes
- Or Fast Feedback CI failed
- If you need E2E: trigger manually

**❌ e2e_server failed:**
- Immich server setup failed
- Usually infrastructure issue
- Check workflow logs
- May need to re-run

**❌ e2e-linux or e2e-windows failed:**
- Integration tests failed
- Check test output in logs
- May need to update E2E tests
- Could indicate real bug in changes

---

## Path Filtering Rules

### Fast Feedback CI runs when:
```yaml
Changed files match:
  - **.go          # Any Go file
  - go.mod         # Dependencies
  - go.sum         # Dependency checksums
  - main.go        # Entry point
  - .github/workflows/fast-feedback.yml  # Workflow itself
```

### E2E Tests run when:
```yaml
Changed files match:
  - app/**         # Application code
  - adapters/**    # Adapter implementations
  - immich/**      # Immich client
  - internal/**    # Internal packages
  - main.go        # Entry point
  - go.mod         # Dependencies
  - go.sum         # Dependency checksums
  - internal/e2e/testdata/immich-server/**  # E2E infrastructure
  - .github/workflows/e2e-tests.yml  # Workflow itself
```

### Both skip when:
```yaml
Only changed files are:
  - **.md          # Markdown docs
  - docs/**        # Documentation
  - scratchpad/**  # Scratch notes
  - LICENSE        # License file
```

---

## Common Scenarios

### Scenario: "I updated documentation only"
**Expected:** Both workflows skip  
**Time:** ~0 seconds  
**Action:** None needed

### Scenario: "I fixed a bug in app/upload/"
**Expected:** Fast Feedback runs, then E2E auto-runs  
**Time:** ~17 minutes  
**Action:** Wait for both to complete

### Scenario: "I added a test in internal/utils/"
**Expected:** Fast Feedback runs, E2E auto-runs  
**Time:** ~17 minutes  
**Action:** Wait for both to complete

### Scenario: "I updated CONTRIBUTING.md"
**Expected:** Both workflows skip  
**Time:** ~0 seconds  
**Action:** None needed

### Scenario: "Fast Feedback passed, but I want E2E anyway"
**Expected:** E2E might have been skipped by path filter  
**Time:** ~12 minutes  
**Action:** Manually trigger E2E workflow

### Scenario: "Fast Feedback failed on lint"
**Expected:** E2E won't run  
**Time:** Fix and push takes ~3 minutes  
**Action:** 
1. Run `golangci-lint run ./...` locally
2. Fix issues
3. Push changes
4. Fast Feedback re-runs automatically

---

## Troubleshooting

### "My PR has been open for 5 minutes with no CI"
**Check:**
- Did you only change documentation files?
- Path filters may have skipped CI
- This is normal and expected

**Fix:**
- If you need CI to run, make a small code change
- Or manually trigger the workflow

### "Fast Feedback passed but E2E didn't run"
**Check:**
- Did you change files outside app/immich/internal?
- Path filters may have skipped E2E
- This is normal and expected

**Fix:**
- Manually trigger E2E if needed
- Or wait for maintainer review

### "E2E tests are taking forever"
**Normal:**
- E2E tests take 12-15 minutes
- Includes server setup, two test suites, cleanup

**Not normal:**
- If it's been > 20 minutes, check logs
- May need to cancel and re-run

### "All tests pass locally but fail in CI"
**Common causes:**
1. Race conditions (CI runs with -race flag)
2. Platform differences (Windows vs Linux)
3. Environment differences (paths, dependencies)
4. Timing issues (network, filesystem)

**Fix:**
- Run with `-race` flag locally: `go test -race ./...`
- Check CI logs for specific failure
- May need to adjust test for CI environment

---

## Deprecation Notice

### Old CI Workflow (Deprecated)
**File:** `.github/workflows/ci.yml.deprecated`  
**Status:** ❌ Disabled  
**Replaced by:** `fast-feedback.yml`

**DO NOT USE** - The old CI workflow has been deprecated because:
- ❌ Duplicated work with E2E tests (50% waste)
- ❌ No path filtering (ran on every change)
- ❌ Slower feedback (10-15 min vs 3-5 min)
- ❌ Higher cost per PR

If you see references to `ci.yml`, update them to `fast-feedback.yml`.

---

## Getting Help

### Workflow failing?
1. Check the workflow logs in GitHub Actions
2. Run checks locally to reproduce
3. Check this guide for common issues
4. Ask in team chat or create an issue

### Need to change workflow behavior?
1. Edit the workflow YAML file
2. Test in a feature branch
3. Submit PR with changes
4. Document what changed and why

### Questions about CI setup?
- Read documentation in `/scratchpad/`
- Check CONTRIBUTING.md
- Ask maintainers

---

## Quick Commands Cheat Sheet

```bash
# Check what will trigger CI
git diff --name-only origin/develop...HEAD

# Run all local checks
make ci-local  # (if Makefile exists)
# or
golangci-lint run ./... && go test -race -v ./...

# Fix common lint issues
golangci-lint run --fix ./...
goimports -w .

# Check test coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Build for all platforms (like CI)
GOOS=linux GOARCH=amd64 go build
GOOS=windows GOARCH=amd64 go build  
GOOS=darwin GOARCH=amd64 go build
```

---

**Last Updated:** November 2, 2025  
**Workflow Version:** 2.0 (Fast Feedback + Conditional E2E)
