# Immich API Monitoring

This monitoring system tracks changes to the Immich OpenAPI specifications and alerts you when updates are detected. This is crucial for maintaining compatibility between your immich-go client and the Immich server API.

## Components

### 1. GitHub Actions Workflow (`.github/workflows/immich-api-monitor.yml`)

**Automated monitoring that:**
- Runs daily at 9:00 AM UTC
- Can be triggered manually via GitHub Actions UI
- Downloads the latest OpenAPI specs from `immich-app/immich`
- Compares with stored baseline
- Creates/updates GitHub issues when changes are detected
- Automatically updates the baseline after alerting

**Permissions required:**
- `contents: write` - To update baseline files
- `issues: write` - To create alert issues

### 2. Manual Monitoring Script (`scripts/check-immich-api.sh`)

**Interactive script for manual checking:**
```bash
./scripts/check-immich-api.sh
```

**Features:**
- Downloads current API specs
- Compares with local baseline
- Shows detailed diffs
- Interactive options to update baseline
- Opens browser to view specs

## Setup

### GitHub Actions Setup

1. **Enable the workflow:**
   - The workflow is ready to use once committed to your repository
   - No additional secrets required (uses `GITHUB_TOKEN`)

2. **First run:**
   - The workflow will create an initial baseline on first execution
   - No alerts will be generated on the initial run

3. **Subsequent runs:**
   - Will compare against the established baseline
   - Creates labeled issues (`immich-api-update`, `needs-review`) when changes detected

### Manual Script Setup

1. **Prerequisites:**
   ```bash
   # Required tools
   curl        # For downloading specs
   jq          # For JSON processing (optional but recommended)
   ```

2. **Run the script:**
   ```bash
   cd /path/to/your/immich-go/project
   ./scripts/check-immich-api.sh
   ```

## Monitoring Data

All monitoring data is stored in `.github/immich-api-monitor/`:

- `immich-openapi-specs-baseline.json` - Reference version for comparison
- `last-checked-commit.txt` - Last Immich commit hash checked

## Usage Scenarios

### During Development
```bash
# Check for API changes before starting work
./scripts/check-immich-api.sh

# If changes detected, review them and update your code accordingly
```

### Continuous Integration
The GitHub Actions workflow automatically:
1. Monitors for changes daily
2. Creates issues with change summaries
3. Updates baseline after alerting
4. Links to relevant Immich commits

### API Update Workflow
When changes are detected:

1. **Review the issue** created by the workflow
2. **Check the diff** to understand what changed
3. **Update your immich-go client** code as needed
4. **Test compatibility** with the new API version
5. **Close the issue** once updates are complete

## Issue Labels

Issues created by the monitor use these labels:
- `immich-api-update` - Identifies API monitoring issues
- `needs-review` - Requires developer attention

## Customization

### Change Monitoring Frequency
Edit the cron schedule in `.github/workflows/immich-api-monitor.yml`:
```yaml
schedule:
  # Current: Daily at 9:00 AM UTC
  - cron: '0 9 * * *'
  
  # Examples:
  # Every 6 hours: '0 */6 * * *'
  # Twice daily: '0 9,21 * * *'
  # Weekly: '0 9 * * 1'
```

### Modify Alert Behavior
The workflow can be customized to:
- Send Slack/Discord notifications
- Create pull requests instead of issues
- Run tests against new API versions
- Generate detailed compatibility reports

### Add Additional Monitoring
Extend the script to monitor:
- Immich release notes
- Database schema changes
- Breaking change announcements
- Version compatibility matrices

## Troubleshooting

### Workflow Fails to Download Specs
- Check if Immich repository structure changed
- Verify the OpenAPI file path is still correct
- GitHub API rate limits (shouldn't affect scheduled runs)

### Script Shows "No Changes" When Expected
- Clear the baseline: `rm .github/immich-api-monitor/immich-openapi-specs-baseline.json`
- Run script again to recreate baseline
- Manually compare with current specs online

### False Positives
- Sometimes formatting changes trigger alerts without functional changes
- Review the actual diff to determine if action is needed
- Consider updating comparison logic to ignore whitespace/formatting

## Integration with Development Workflow

### Pre-commit Checks
Add to your development routine:
```bash
# Check for API changes before major development
./scripts/check-immich-api.sh

# Run your tests against current baseline
go test ./...
```

### Release Preparation
Before releasing new versions:
1. Run API monitoring script
2. Update compatibility documentation
3. Test against latest Immich version
4. Update version requirements if needed

## Monitoring History

The workflow maintains a history of:
- When changes were detected
- Which Immich commits introduced changes
- Issue creation and resolution timeline
- Baseline update history (via Git commits)

This provides valuable insight into Immich API evolution and your project's compatibility maintenance.