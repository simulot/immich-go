# Examples and Use Cases

This guide provides practical examples for common Immich-Go scenarios.

## Quick Reference

| Scenario | Command | Documentation |
|----------|---------|---------------|
| [Upload local photos](#local-photo-upload) | `upload from-folder` | Basic photo upload |
| [Google Photos migration](#google-photos-migration) | `upload from-google-photos` | Takeout import |
| [iCloud import](#icloud-import) | `upload from-icloud` | iCloud takeout |
| [Server backup](#server-backup) | `archive from-immich` | Full server archive |
| [Server migration](#server-migration) | `upload from-immich` | Transfer between servers |
| [Photo organization](#photo-organization) | `stack` | Organize existing photos |
| [Selective sync](#selective-sync) | Various filters | Partial imports |

## Local Photo Upload

### Basic Upload
```bash
# Upload entire photo collection
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  /home/user/Pictures
```

### Organized Upload with Albums
```bash
# Create albums from folder structure
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --folder-as-album=FOLDER \
  --manage-raw-jpeg=StackCoverRaw \
  /home/user/Pictures/Organized
```

### Tagged Upload
```bash
# Add custom tags and session tracking
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --tag="Import/2024" \
  --tag="Source/LocalFiles" \
  --session-tag \
  /home/user/Pictures
```

### ZIP Archive Upload
```bash
# Upload from compressed archives
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  /path/to/photo-archive.zip
```

## Google Photos Migration

### Complete Takeout Import
```bash
# Import all parts of a Google Photos takeout
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --manage-raw-jpeg=StackCoverRaw \
  --manage-burst=Stack \
  /downloads/takeout-*.zip
```

### Selective Import
```bash
# Import only visible phtos and exclude partner photos
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --include-partner=false \
  --include-trashed=false \
  /downloads/takeout-*.zip
```

### Album-Specific Import
```bash
# Import from specific album only
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --from-album-name="Vacation 2023" \
  /downloads/takeout-*.zip
```

### Large Takeout (Best Practices)
```bash
# Optimized for large takeouts (100k+ photos)
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=4 \
  --client-timeout=60m \
  --pause-immich-jobs=true \
  --on-errors=continue \
  --session-tag \
  /downloads/takeout-*.zip
```

## iCloud Import

### Basic iCloud Import
```bash
# Import iCloud takeout
immich-go upload from-icloud \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --manage-heic-jpeg=StackCoverJPG \
  /path/to/icloud-export
```

### iCloud with Memories
```bash
# Include iCloud memories as albums
immich-go upload from-icloud \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --memories \
  --manage-heic-jpeg=StackCoverJPG \
  /path/to/icloud-export
```

## Server Backup

### Complete Server Archive
```bash
# Backup entire Immich server
immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --write-to-folder=/backup/immich-complete
```

### Incremental Backup
```bash
# Backup only recent photos (last 30 days)
immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --from-date-range=$(date -d '30 days ago' '+%Y-%m-%d'),$(date '+%Y-%m-%d') \
  --write-to-folder=/backup/immich-recent
```

### Album-Specific Backup
```bash
# Backup specific albums
immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --from-album="Family Photos" \
  --from-album="Travel" \
  --write-to-folder=/backup/immich-albums
```

### Yearly Archives
```bash
# Create separate archives by year
for year in 2020 2021 2022 2023 2024; do
  immich-go archive from-immich \
    --server=http://localhost:2283 \
    --api-key=your-api-key \
    --from-date-range=$year \
    --write-to-folder=/backup/immich-$year
done
```

## Server Migration

### Complete Migration
```bash
# Transfer all photos between Immich servers
immich-go upload from-immich \
  --from-server=http://old-server:2283 \
  --from-api-key=old-api-key \
  --server=http://new-server:2283 \
  --api-key=new-api-key \
  --concurrent-tasks=4
```

### Selective Migration
```bash
# Migrate specific date range
immich-go upload from-immich \
  --from-server=http://old-server:2283 \
  --from-api-key=old-api-key \
  --from-date-range=2023-01-01,2023-12-31 \
  --server=http://new-server:2283 \
  --api-key=new-api-key
```

### Album Migration
```bash
# Migrate specific albums
immich-go upload from-immich \
  --from-server=http://old-server:2283 \
  --from-api-key=old-api-key \
  --from-album="Family" \
  --from-album="Work" \
  --server=http://new-server:2283 \
  --api-key=new-api-key
```

## Photo Organization

### Organize Existing Library
```bash
# Stack burst photos and RAW+JPEG pairs
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --manage-burst=Stack \
  --manage-raw-jpeg=StackCoverRaw \
  --manage-heic-jpeg=StackCoverJPG
```

### Test Organization (Dry Run)
```bash
# Preview organization changes
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --manage-burst=Stack \
  --manage-raw-jpeg=StackCoverRaw \
  --dry-run
```

### Folder Reorganization
```bash
# Reorganize messy folders into date-based structure
immich-go archive from-folder \
  --write-to-folder=/organized-photos \
  --manage-raw-jpeg=StackCoverRaw \
  /messy/photo/folders
```

## Selective Sync

### Date Range Upload
```bash
# Upload photos from specific year
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --date-range=2023 \
  /home/user/Pictures
```

### File Type Filtering
```bash
# Upload only videos
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --include-type=VIDEO \
  /home/user/Movies

# Upload only specific image formats
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --include-extensions=.jpg,.png,.heic \
  /home/user/Pictures
```

### Exclude Unwanted Files
```bash
# Skip large video files and screenshots
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --exclude-extensions=.mov,.mp4 \
  --ban-file="*screenshot*" \
  --ban-file="*Screen Shot*" \
  /home/user/Pictures
```

## Performance Optimization

### High-Performance Upload
```bash
# Optimize for fast network and powerful server
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=16 \
  --client-timeout=30m \
  --pause-immich-jobs=true \
  /large/photo/collection
```

### Conservative Upload (Slow Network)
```bash
# Optimize for slow/unstable connection
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=1 \
  --client-timeout=120m \
  --on-errors=continue \
  /photos
```

### Background Processing
```bash
# Run upload in background with logging
nohup immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --log-file=/tmp/upload.log \
  --no-ui \
  /photos > /dev/null 2>&1 &
```

## Automation Scripts

### Bash Script for Regular Backups
```bash
#!/bin/bash
# backup-immich.sh

set -e

IMMICH_SERVER="http://localhost:2283"
API_KEY="your-api-key"
BACKUP_DIR="/backup/immich"
DATE=$(date '+%Y-%m-%d')

echo "Starting Immich backup: $DATE"

# Create backup directory
mkdir -p "$BACKUP_DIR/$DATE"

# Backup recent photos (last 7 days)
immich-go archive from-immich \
  --server="$IMMICH_SERVER" \
  --api-key="$API_KEY" \
  --from-date-range="$(date -d '7 days ago' '+%Y-%m-%d'),$(date '+%Y-%m-%d')" \
  --write-to-folder="$BACKUP_DIR/$DATE" \
  --log-file="$BACKUP_DIR/$DATE/backup.log"

echo "Backup completed: $BACKUP_DIR/$DATE"
```

### PowerShell Script for Windows
```powershell
# backup-immich.ps1

$ImmichServer = "http://localhost:2283"
$ApiKey = "your-api-key"
$BackupDir = "D:\Backup\Immich"
$Date = Get-Date -Format "yyyy-MM-dd"

Write-Host "Starting Immich backup: $Date"

# Create backup directory
New-Item -ItemType Directory -Path "$BackupDir\$Date" -Force

# Backup recent photos
& immich-go archive from-immich `
  --server="$ImmichServer" `
  --api-key="$ApiKey" `
  --from-date-range="$(Get-Date (Get-Date).AddDays(-7) -Format 'yyyy-MM-dd'),$(Get-Date -Format 'yyyy-MM-dd')" `
  --write-to-folder="$BackupDir\$Date" `
  --log-file="$BackupDir\$Date\backup.log"

Write-Host "Backup completed: $BackupDir\$Date"
```

### Cron Job for Automated Backups
```bash
# Add to crontab: crontab -e

# Daily backup at 2 AM
0 2 * * * /home/user/scripts/backup-immich.sh

# Weekly full backup on Sundays at 3 AM  
0 3 * * 0 immich-go archive from-immich --server=http://localhost:2283 --api-key=your-key --write-to-folder=/backup/weekly/$(date +\%Y-\%m-\%d)
```

## Troubleshooting Examples

### Debug Upload Issues
```bash
# Maximum debug information
immich-go --log-level=DEBUG --api-trace \
  upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --dry-run \
  /test-photos
```

### Test Server Connection
```bash
# Verify server connectivity
immich-go --log-level=DEBUG \
  archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --from-date-range=2024-01-01,2024-01-01 \
  --write-to-folder=/tmp/test \
  --dry-run
```

### Handle Large Files
```bash
# Upload large video files with extended timeout
immich-go upload from-folder \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --include-type=VIDEO \
  --client-timeout=180m \
  --concurrent-tasks=2 \
  /large-videos
```

## See Also

- [Command Reference](commands/) - Detailed option documentation
- [Configuration Guide](configuration.md) - All configuration options
- [Best Practices](best-practices.md) - Performance and reliability tips
- [Technical Details](technical.md) - File processing information
