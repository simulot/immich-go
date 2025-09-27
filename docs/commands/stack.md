# Stack Command

The `stack` command organizes related photos into stacks on your Immich server without uploading new content.

## Syntax

```bash
immich-go stack [options]
```

## Purpose

Stacking groups related photos together for better organization:
- **Burst photos** from rapid shooting
- **RAW + JPEG** pairs from cameras that shoot both formats
- **HEIC + JPEG** pairs from iPhone/iPad Live Photos
- **Epson FastFoto** scan groups (original, corrected, back side)

## Required Options

| Option          | Required | Description       |
| --------------- | :------: | ----------------- |
| `-s, --server`  |    Y     | Immich server URL |
| `-k, --api-key` |    Y     | Your API key      |

## Connection Options

| Option              | Default | Description                       |
| ------------------- | ------- | --------------------------------- |
| `--skip-verify-ssl` | `false` | Skip SSL certificate verification |
| `--client-timeout`  | `5m`    | Server call timeout               |
| `--api-trace`       | `false` | Enable API call tracing           |

## Behavior Options

| Option        | Default | Description                              |
| ------------- | ------- | ---------------------------------------- |
| `--dry-run`   | `false` | Simulate stacking without making changes |
| `--time-zone` | System  | Override timezone for date operations    |

## Stacking Rules

### Burst Photos

| Option           | Values                                              | Description                   |
| ---------------- | --------------------------------------------------- | ----------------------------- |
| `--manage-burst` | `NoStack`, `Stack`, `StackKeepRaw`, `StackKeepJPEG` | How to handle burst sequences |

**Detection Methods:**
- **Time-based**: Photos taken within 900ms of each other
- **Filename patterns**: Device-specific naming conventions

**Supported Devices:**
- **Huawei**: `IMG_20231014_183246_BURST001_COVER.jpg`, `IMG_20231014_183246_BURST002.jpg`
- **Google Pixel**: `PXL_20230330_184138390.MOTION-01.COVER.jpg`, `PXL_20230330_184138390.MOTION-02.ORIGINAL.jpg`
- **Samsung**: `20231207_101605_001.jpg`, `20231207_101605_002.jpg`
- **Sony Xperia**: `DSC_0001_BURST20230709220904977.JPG`, `DSC_0035_BURST20230709220904977_COVER.JPG`
- **Nexus**: `00001IMG_00001_BURST20171111030039.jpg`, `00015IMG_00015_BURST20171111030039_COVER.jpg`
- **Nothing**: `00001IMG_00001_BURST1723801037429_COVER.jpg`, `00002IMG_00002_BURST1723801037429.jpg`

### RAW + JPEG Management

| Option              | Values                                                            | Description           |
| ------------------- | ----------------------------------------------------------------- | --------------------- |
| `--manage-raw-jpeg` | `NoStack`, `KeepRaw`, `KeepJPG`, `StackCoverRaw`, `StackCoverJPG` | Handle RAW+JPEG pairs |

- **NoStack**: Leave files separate
- **KeepRaw**: Delete JPEG, keep RAW only
- **KeepJPG**: Delete RAW, keep JPEG only  
- **StackCoverRaw**: Stack with RAW as cover image
- **StackCoverJPG**: Stack with JPEG as cover image

### HEIC + JPEG Management

| Option               | Values                                                              | Description            |
| -------------------- | ------------------------------------------------------------------- | ---------------------- |
| `--manage-heic-jpeg` | `NoStack`, `KeepHeic`, `KeepJPG`, `StackCoverHeic`, `StackCoverJPG` | Handle HEIC+JPEG pairs |

Same logic as RAW+JPEG but for HEIC format files.

### Epson FastFoto

| Option                    | Default | Description                      |
| ------------------------- | ------- | -------------------------------- |
| `--manage-epson-fastfoto` | `false` | Stack Epson FastFoto scan groups |

**File Pattern:**
- `image-name.jpg` (Original scan)
- `image-name_a.jpg` (Corrected scan) 
- `image-name_b.jpg` (Back of photo)

When enabled, stacks all three with corrected scan as cover.

## Examples

### Basic Stacking
```bash
# Stack burst photos automatically
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-burst=Stack

# Preview stacking without changes
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-burst=Stack \
  --dry-run
```

### RAW + JPEG Management
```bash  
# Stack with RAW as cover
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-raw-jpeg=StackCoverRaw

# Keep only JPEG files
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-raw-jpeg=KeepJPG
```

### Comprehensive Organization
```bash
# Handle all photo types
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-burst=Stack \
  --manage-raw-jpeg=StackCoverRaw \
  --manage-heic-jpeg=StackCoverJPG \
  --manage-epson-fastfoto=true
```

### Safe Testing
```bash
# Test stacking rules without changes
immich-go stack \
  --server=http://localhost:2283 \
  --api-key=your-key \
  --manage-burst=Stack \
  --manage-raw-jpeg=StackCoverRaw \
  --dry-run
```

## Detection Logic

### Burst Detection Priority
1. **Filename patterns** (device-specific)
2. **Time-based detection** (900ms threshold)

### File Pairing Logic
1. **Exact name matching** (different extensions)
2. **Same directory location**
3. **Similar timestamps** (within reasonable range)

## Best Practices

### 1. Test First
Always use `--dry-run` to preview changes:
```bash
immich-go stack --dry-run --manage-burst=Stack --server=... --api-key=...
```

### 2. Incremental Approach
Handle one type at a time:
```bash
# First, handle bursts
immich-go stack --manage-burst=Stack --server=... --api-key=...

# Then, handle RAW+JPEG
immich-go stack --manage-raw-jpeg=StackCoverRaw --server=... --api-key=...
```

### 3. Backup Consideration
Consider using the [archive command](archive.md) before major organization:
```bash
immich-go archive from-immich --write-to-folder=/backup --server=... --api-key=...
```

### 4. Monitor Progress
Use logging to track operations:
```bash
immich-go stack \
  --log-level=DEBUG \
  --log-file=/tmp/stacking.log \
  --manage-burst=Stack \
  --server=... --api-key=...
```

## Troubleshooting

### Nothing Gets Stacked
- Check photo timestamps and filenames
- Verify photos are from supported devices
- Use `--api-trace` to see server communication
- Try `--dry-run` with `--log-level=DEBUG`

### Unexpected Stacking
- Review detection logic for your device type
- Check if time-based detection is too aggressive
- Use filename patterns for specific device types

### Performance Issues
- Increase `--client-timeout` for large libraries
- Process in smaller batches by date range (using archive/upload workflow)

## See Also

- [Upload Command](upload.md) - Stacking during upload
- [Technical Details](../technical.md) - Detection algorithms
- [Best Practices](../best-practices.md) - Organization strategies