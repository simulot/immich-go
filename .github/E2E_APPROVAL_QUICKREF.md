# Quick Reference: E2E Approval for Maintainers

## Quick Commands

### Approve E2E Tests for External PR
```bash
gh pr edit PR_NUMBER --add-label "e2e-approved"
```

### Remove Approval
```bash
gh pr edit PR_NUMBER --remove-label "e2e-approved"
```

### Check if Label Exists
```bash
gh label list | grep e2e-approved
```

### Create Label (First Time)
```bash
gh label create "e2e-approved" \
  --description "E2E tests approved by maintainer" \
  --color "0E8A16"
```

### Check Contributor Status
```bash
gh api "/repos/simulot/immich-go/collaborators/USERNAME"
```

## Quick Decision Matrix

| Scenario | Fast Feedback (ci.yml) | E2E Tests (ci-e2e.yml) | Action Required |
|----------|----------------------|---------------------|-----------------|
| 🆕 External contributor PR | ✅ Runs automatically | ⏸️ Skipped | Review code → Add label |
| 👤 Repository collaborator PR | ✅ Runs automatically | ✅ Runs automatically | None |
| 📄 Documentation-only PR | ✅ Runs automatically | ⏭️ Skipped (no code changes) | None |
| 🔧 Push to main/develop | ✅ Runs automatically | ✅ Runs automatically | None |
| 🎯 Manual trigger | ✅ Runs | ✅ Runs (optional skip approval) | None |

## Approval Checklist

Before adding `e2e-approved` label:

- [ ] Review **all code changes** in the PR
- [ ] Check for attempts to access/log secrets
- [ ] Verify no malicious code (crypto mining, data exfiltration, etc.)
- [ ] Check contributor's GitHub profile and history
- [ ] Ensure PR doesn't modify workflow files suspiciously
- [ ] Verify changes are relevant to stated purpose

**When in doubt, DON'T approve!** Ask for clarification or other maintainer's opinion.

## What Gets Tested?

### Fast Feedback (Always, No Secrets)
- Branch naming validation
- golangci-lint
- Unit tests with race detection
- Code coverage
- Build verification

### E2E Tests (Approval Required, Uses Secrets)
- Immich server deployment (Docker)
- Linux client E2E tests
- Windows client E2E tests
- Full integration testing
- Tailscale network communication

## Expected PR Comments

### External Contributor PR (No Label)
```markdown
⚠️ E2E Tests Approval Required

This PR is from an external contributor and requires manual review 
before E2E tests can run.

For maintainers: 
1. Review the changes in this PR carefully
2. If safe to run E2E tests, add the `e2e-approved` label to this PR
3. E2E tests will automatically trigger once the label is added

Note: Fast feedback checks (linting, unit tests, build) run 
automatically and don't require approval.
```

## Workflow Status Indicators

### Waiting for Approval
```
✅ CI - Passed
⏸️ E2E Tests - Waiting for approval
```

### Approved and Running
```
✅ CI - Passed
🔄 E2E Tests - Running
```

### All Complete
```
✅ CI - Passed
✅ E2E Tests - Passed
```

## Troubleshooting Quick Tips

**E2E not running after adding label?**
→ Check label is exactly `e2e-approved` (case-sensitive)

**Collaborator not auto-approved?**
→ Verify in Settings → Collaborators with write access

**Bot comment not appearing?**
→ Check if changes are code-related (not doc-only)

**Need to re-run E2E?**
→ Remove and re-add label, or use manual trigger

## Security Red Flags 🚩

Watch out for:
- Code that accesses `${{ secrets.* }}`
- Unusual network requests
- Base64 encoded payloads
- Workflow file modifications
- Code obfuscation
- Suspicious imports or dependencies
- Crypto mining code
- Data exfiltration attempts

## Contact

For questions:
- See: `.github/E2E_APPROVAL_GUIDE.md` (detailed guide)
- See: `.github/workflows/README.md` (workflow overview)
- Open: GitHub issue or discussion

---

**Remember:** Only approve PRs you've thoroughly reviewed and trust!
