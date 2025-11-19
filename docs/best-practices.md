# Best Practices

This guide provides recommendations for optimal performance, reliability, and organization when using Immich-Go.

## Google Photos Migration

### Takeout Creation

#### ✅ Recommended Settings
- **Format**: Choose ZIP format for easier processing
- **File Size**: Select maximum size (50GB) to minimize parts
- **Content**: Include all photos, videos, and metadata
- **Download**: Ensure all parts are downloaded completely

#### ⚠️ Common Pitfalls
- **Incomplete Downloads**: Verify all `takeout-001.zip`, `takeout-002.zip`, etc. files are present
- **Mixed Formats**: Don't mix ZIP and TGZ formats in the same import
- **Partial Takeouts**: Some Google takeouts may be incomplete - request a new one if many files are missing

### Import Strategy

#### Large Collections (100k+ Photos)
```bash
# Conservative approach for maximum reliability
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=4 \
  --client-timeout=60m \
  --pause-immich-jobs=true \
  --on-server-errors=continue \
  --session-tag \
  /path/to/takeout-*.zip
```

#### Medium Collections (10k-100k Photos)
```bash
# Balanced performance and reliability
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=8 \
  --manage-raw-jpeg=StackCoverRaw \
  --manage-burst=Stack \
  /path/to/takeout-*.zip
```

#### Small Collections (<10k Photos)
```bash
# Fast import with full processing
immich-go upload from-google-photos \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --concurrent-tasks=12 \
  --manage-raw-jpeg=StackCoverRaw \
  --manage-burst=Stack \
  --manage-heic-jpeg=StackCoverJPG \
  /path/to/takeout-*.zip
```

### Troubleshooting Import Issues

#### Many Files Not Imported
1. **Check Takeout Completeness**: Verify all parts are included
   ```bash
   ls -la takeout-*.zip
   ```

2. **Force Import Unmatched Files**:
   ```bash
   immich-go upload from-google-photos \
     --include-unmatched \
     --server=... --api-key=... /path/to/takeout-*.zip
   ```

3. **Request New Takeout**: If data seems incomplete, create a new takeout for missing periods

#### Resuming Interrupted Imports
- **Safe to Restart**: Immich-Go detects existing files and skips duplicates
- **Use Session Tags**: Enable `--session-tag` to track what was imported when
- **Check Logs**: Review log files to identify where the import stopped

## Performance Optimization

### Network Considerations

#### Gigabit LAN (Fast, Stable)
```bash
# High throughput configuration
--concurrent-tasks=16
--client-timeout=30m
--pause-immich-jobs=true
```

#### Internet Connection (Variable Speed)
```bash
# Adaptive configuration
--concurrent-tasks=4-8
--client-timeout=60m
--on-server-errors=continue
```

#### Slow/Unstable Network
```bash
# Conservative configuration
--concurrent-tasks=1-2
--client-timeout=120m
--on-server-errors=continue
```

### Server Considerations

#### Powerful Server (High CPU/RAM)
```bash
# Maximize server utilization
--concurrent-tasks=12-20
--pause-immich-jobs=false  # Let server handle both
--client-timeout=30m
```

#### Limited Server Resources
```bash
# Reduce server load
--concurrent-tasks=2-4
--pause-immich-jobs=true
--client-timeout=60m
```

#### NAS or Low-Power Server
```bash
# Minimal resource usage
--concurrent-tasks=1-2
--pause-immich-jobs=true
--client-timeout=180m
```

### Storage Considerations

#### Fast Storage (SSD)
- Use higher concurrency (8-16 uploads)
- Enable all file management features
- Process larger batches

#### Slow Storage (Traditional HDD)
- Reduce concurrency (2-4 uploads)
- Process smaller batches
- Consider staging on faster temporary storage

#### Network Storage
- Account for network latency in timeouts
- Test with small batches first
- Monitor network utilization

## Organization Strategies

### Album Management

#### Folder-Based Albums
```bash
# Create albums from folder structure
immich-go upload from-folder \
  --folder-as-album=FOLDER \
  --album-path-joiner=" - " \
  --server=... --api-key=... /organized/photos/
```

**Benefits**: 
- Maintains existing organization
- Easy to understand structure
- Works well with hierarchical folder systems

**Best For**: Already organized photo collections

#### Date-Based Organization
```bash
# Let Immich organize by date, use tags for categories
immich-go upload from-folder \
  --tag="Source/Import2024" \
  --tag="Camera/Canon5D" \
  --session-tag \
  --server=... --api-key=... /photos/
```

**Benefits**:
- Chronological organization
- Flexible tagging system
- Better for mixed sources

**Best For**: Large collections from multiple sources

#### Hybrid Approach
```bash
# Combine albums and tags strategically
immich-go upload from-folder \
  --folder-as-album=PATH \
  --folder-as-tags=true \
  --tag="Import/$(date +%Y-%m)" \
  --server=... --api-key=... /photos/
```

### File Management

#### RAW + JPEG Workflows

##### Keep Both Separately
```bash
--manage-raw-jpeg=NoStack
```
**When to Use**: Need both formats accessible, have ample storage

##### RAW-Centric Workflow  
```bash
--manage-raw-jpeg=StackCoverRaw
```
**When to Use**: Primarily edit RAW files, JPEG for quick preview

##### JPEG-Centric Workflow
```bash
--manage-raw-jpeg=StackCoverJPG
```
**When to Use**: Mainly use JPEG, keep RAW as backup

##### Storage-Conscious
```bash
--manage-raw-jpeg=KeepRaw  # Or KeepJPG based on preference
```
**When to Use**: Limited storage, need to choose one format

#### Burst Photo Management

##### Creative Photography
```bash
--manage-burst=Stack
```
**Benefits**: Keeps all shots while reducing clutter

##### Casual Photography
```bash
--manage-burst=StackKeepJPEG
```
**Benefits**: Saves storage, keeps most useful format

##### Professional Photography
```bash
--manage-burst=NoStack
```
**Benefits**: Full access to all shots for selection

### Tagging Strategies

#### Hierarchical Tagging
```bash
# Geographic hierarchy
--tag="Location/Europe/France/Paris"

# Event hierarchy  
--tag="Events/2024/Wedding/Ceremony"

# Equipment hierarchy
--tag="Equipment/Camera/Canon/5D-Mark-IV"
```

#### Multi-Dimensional Tagging
```bash
# Combine multiple tag dimensions
--tag="Location/Paris" \
--tag="Event/Wedding" \
--tag="People/Family" \
--tag="Year/2024"
```

#### Source Tracking
```bash
# Always tag sources for future reference
--tag="Source/GooglePhotos" \
--tag="Import/$(date +%Y-%m-%d)" \
--session-tag
```

## Backup and Recovery

### Backup Strategies

#### 3-2-1 Backup Rule
1. **3 Copies**: Original + 2 backups
2. **2 Different Media**: Local + cloud/external
3. **1 Offsite**: Cloud or remote location

#### Implementation Example
```bash
# Local backup (Copy 2)
immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --write-to-folder=/local-backup/immich

# Offsite backup (Copy 3) - sync local backup to cloud
rsync -av /local-backup/immich/ user@remote-server:/backups/immich/
```

#### Incremental Backups
```bash
#!/bin/bash
# Daily incremental backup script

YESTERDAY=$(date -d '1 day ago' '+%Y-%m-%d')
TODAY=$(date '+%Y-%m-%d')

immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --from-date-range="$YESTERDAY,$TODAY" \
  --write-to-folder="/backup/incremental/$TODAY"
```

#### Full Periodic Backups
```bash
# Monthly full backup
immich-go archive from-immich \
  --server=http://localhost:2283 \
  --api-key=your-api-key \
  --write-to-folder="/backup/full/$(date +%Y-%m)"
```

### Testing and Validation

#### Pre-Migration Testing
```bash
# Test with small subset first
immich-go upload from-folder \
  --dry-run \
  --log-level=DEBUG \
  --server=... --api-key=... /small-test-folder/
```

#### Backup Validation
```bash
# Test restore capability
immich-go upload from-folder \
  --server=http://test-server:2283 \
  --api-key=test-api-key \
  /backup/folder/2024/2024-01/
```

## Security and Privacy

### API Key Management

#### ✅ Best Practices
- **Separate Keys**: Use different API keys for different purposes
- **Minimal Permissions**: Only grant necessary permissions
- **Regular Rotation**: Rotate keys periodically
- **Secure Storage**: Don't hardcode keys in scripts

#### Example Key Strategy
```bash
# Different keys for different operations
UPLOAD_KEY="key-with-upload-permissions"
BACKUP_KEY="key-with-read-only-permissions"
ADMIN_KEY="key-with-admin-permissions"

# Upload operations
immich-go upload from-folder --api-key="$UPLOAD_KEY" ...

# Backup operations  
immich-go archive from-immich --api-key="$BACKUP_KEY" ...
```

#### Script Security
```bash
#!/bin/bash
# Secure script example

# Read API key from secure file
API_KEY=$(cat ~/.config/immich-go/api-key)
chmod 600 ~/.config/immich-go/api-key  # Restrict permissions

# Or use environment variable
export IMMICH_API_KEY="your-key"
immich-go upload from-folder --api-key="$IMMICH_API_KEY" ...
```

### Network Security

#### SSL/TLS Configuration
```bash
# Always use HTTPS in production
immich-go upload from-folder \
  --server=https://immich.yourdomain.com \
  --api-key=your-key \
  /photos/

# Only use --skip-verify-ssl for testing/development
immich-go upload from-folder \
  --server=https://immich-dev.local \
  --skip-verify-ssl \  # Only for self-signed certs in dev
  --api-key=your-key \
  /photos/
```

#### Network Isolation
- **VPN Access**: Access Immich server through VPN when possible
- **Firewall Rules**: Restrict Immich server access to specific networks
- **Reverse Proxy**: Use reverse proxy with proper SSL termination

## Monitoring and Troubleshooting

### Logging Strategy

#### Development/Testing
```bash
--log-level=DEBUG \
--log-file=/tmp/immich-go-debug.log \
--api-trace
```

#### Production Operations
```bash
--log-level=INFO \
--log-file=/var/log/immich-go/upload-$(date +%Y%m%d).log
```

#### Automated Operations
```bash
--log-level=WARN \
--log-file=/var/log/immich-go/automated.log \
--no-ui
```

### Health Monitoring

#### Upload Progress Tracking
```bash
# Monitor upload with session tags
immich-go upload from-folder \
  --session-tag \
  --tag="Batch/$(date +%Y%m%d-%H%M)" \
  --server=... --api-key=... /photos/
```

#### Storage Monitoring
```bash
# Check available space before large operations
df -h /immich-storage
df -h /temp-directory

# Monitor during operation
watch df -h /immich-storage
```

#### Server Performance
```bash
# Monitor server resources during upload
htop
iotop
nethogs
```

### Error Recovery

#### Handling Upload Failures
```bash
# Continue on errors, log issues
immich-go upload from-folder \
  --on-server-errors=continue \
  --log-level=INFO \
  --log-file=/var/log/errors.log \
  --server=... --api-key=... /photos/

# Review errors later
grep "ERROR" /var/log/errors.log
```

#### Network Interruption Recovery
```bash
# Restart with same parameters - Immich-Go handles duplicates
immich-go upload from-folder \
  --server=... --api-key=... /photos/
```

#### Partial Import Recovery
```bash
# Use session tags to identify what was processed
immich-go upload from-folder \
  --session-tag \
  --server=... --api-key=... /remaining-photos/
```

## Migration Planning

### Pre-Migration Checklist

#### ✅ Requirements
- [ ] Immich server properly configured and accessible
- [ ] API key created with all necessary permissions
- [ ] Sufficient storage space (estimate 1.5x source size)
- [ ] Network capacity planned (estimate transfer time)
- [ ] Backup of source data created

#### ✅ Testing
- [ ] Test with small subset of photos
- [ ] Verify upload quality and metadata preservation
- [ ] Test backup/restore procedures
- [ ] Validate performance with expected load

#### ✅ Preparation
- [ ] Clean up source data (remove unwanted files)
- [ ] Organize source structure if needed
- [ ] Plan migration schedule (off-peak hours)
- [ ] Prepare monitoring and logging

### Migration Execution

#### Phase 1: Test Migration
```bash
# Small test batch
immich-go upload from-folder \
  --dry-run \
  --server=... --api-key=... /test-photos/
```

#### Phase 2: Pilot Migration
```bash
# Subset of real data
immich-go upload from-folder \
  --session-tag \
  --tag="Migration/Pilot" \
  --server=... --api-key=... /pilot-batch/
```

#### Phase 3: Full Migration
```bash
# Complete migration with monitoring
immich-go upload from-google-photos \
  --session-tag \
  --tag="Migration/Full" \
  --concurrent-tasks=8 \
  --log-file=/var/log/migration.log \
  --server=... --api-key=... /takeout-*.zip
```

### Post-Migration

#### Validation Steps
1. **Count Verification**: Compare source and destination counts
2. **Spot Checks**: Verify random samples for quality
3. **Metadata Check**: Ensure dates, locations, albums preserved
4. **Organization Review**: Confirm albums and tags applied correctly

#### Cleanup
```bash
# Archive original source data after validation
tar -czf google-photos-backup.tar.gz takeout-*.zip
mv google-photos-backup.tar.gz /secure-archive/

# Clean up temporary files
rm -rf /temp/immich-go-*
```

## See Also

- [Examples](examples.md) - Practical implementation examples
- [Configuration](configuration.md) - Detailed option explanations
- [Commands](commands/) - Complete command reference
- [Technical Details](technical.md) - File processing information