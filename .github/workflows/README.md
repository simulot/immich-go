# GitHub Actions Workflows

This document provides an overview of the CI/CD workflows used in the `immich-go` project.

## Workflows Overview

| Workflow | Trigger | Description |
|---|---|---|
| [PR Checks](#pr-checks---fast-feedback) | Pull Request | Provides fast feedback (lint, unit tests, build) on code changes. |
| [Nightly E2E](#nightly-e2e-tests) | Daily Schedule | Runs end-to-end tests against the latest Immich server release. |
| [Run E2E Tests](#run-e2e-tests) | PR Comment | Runs end-to-end tests for a specific pull request. |
| [Release](#release) | Manual | Creates a new release using GoReleaser. |

---

## 1. PR Checks - Fast Feedback

**File:** `pr-checks.yml`

This workflow runs on every pull request to provide rapid feedback to contributors. It checks for code quality and runs basic tests.

### Triggers

- On `pull_request` to `main` or `develop` branches.



### Jobs

1.  **`🔎 Analyze Changes`**:
    -   Determines if the pull request contains changes to Go code, scripts, or workflow files.
    -   It ignores changes to documentation (e.g., `.md` files).
    -   The result (`has_code_changes`) is used to decide whether to run the next job.

2.  **`🚀 Fast Feedback`**:
    -   This job runs only if code changes are detected.
    -   **Lint Code**: Runs `golangci-lint` to check for style issues.
    -   **Run Unit Tests**: Executes `go test` with the `-race` detector.
    -   **Build Binary**: Compiles the application to ensure it builds correctly.

3.  **`📢 Notify for E2E`**:
    -   If the `fast-feedback` job succeeds, this job posts a comment on the pull request.
    -   The comment informs maintainers and contributors how to trigger the full End-to-End (E2E) tests.

---

## 2. Nightly E2E Tests

**File:** `nightly-e2e.yml`

This workflow validates the `main` branch against the very latest version of the Immich server to catch any breaking changes from upstream.

### Triggers

-   Daily at 1:00 AM UTC (`schedule`).
-   Manually via `workflow_dispatch`.

### Jobs

1.  **`🧪 Test against latest Immich`**:
    -   **Deploy latest Immich server**: Downloads and starts the official Immich server using the latest `docker-compose.yml` from their repository.
    -   **Wait for Immich API**: Pings the server until it's ready to accept requests.
    -   **Run E2E Tests**: Executes the Go E2E test suite (`go test -tags=e2e`).
    -   **Show Immich logs on failure**: If the tests fail, it dumps the Docker logs for the Immich server to help with debugging.
    -   **Cleanup**: Stops and removes the Docker containers and volumes, ensuring a clean environment for the next run.
