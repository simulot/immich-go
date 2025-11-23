# Workflow Guide

## Immich-Go PR Testing Flow

1. **PR Fast Feedback** (`pr-fast-feedback.yml`)
   - Runs on every PR push.
   - Covers lint, unit tests, security scan, and multi-platform builds.
   - Posts a reminder that E2E tests need maintainer action for forked PRs.

2. **E2E Authorization** (`e2e-authorize.yml`)
   - Listens for:
     - `pull_request_review` (state `approved`).
     - Maintainer comment `/run-e2e`.
     - `workflow_run` from Fast Feedback (for trusted authors).
   - Validates PR metadata (head SHA, repo, branch, doc-only changes).
   - Dispatches `e2e-tests.yml` with the PR head repo/ref plus `trusted` flag.

3. **E2E Tests** (`e2e-tests.yml`)
   - Spins up the Immich server via Tailscale, then runs Linux and Windows clients.
   - Uses the `trusted` flag to route secrets through the right environment (`e2e-trusted` vs `e2e-infra`).

### Typical Maintainer Actions

- **Approve the PR**: automatically triggers E2E for any contributor once all fast-feedback checks pass.
- **Re-run without approving**: leave `/run-e2e` as a comment on the PR.
- **Need another run after new commits?** Submit a fresh approval or comment `/run-e2e` again.

This setup keeps secrets protected while making “approve → run all E2E” the default path for contributor PRs.
