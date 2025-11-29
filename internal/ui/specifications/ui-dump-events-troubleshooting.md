# Troubleshooting: Experimental UI Event Dump

## When to Use `--ui-dump-events`

Enable the hidden flag `--ui-dump-events` to log all experimental UI events for debugging purposes. This is useful when:

- Diagnosing issues in the new TUI event pipeline
- Verifying event flow and payloads during development
- Comparing legacy and experimental UI behaviors

**Note:**
- The flag is hidden from standard help output
- Output is verbose and intended for developer troubleshooting, not regular use

Example usage:
```bash
immich-go upload from-folder --ui-dump-events --server=http://localhost:2283 --api-key=your-key /photos
```

See also: `docs/commands/upload.md` for user-facing flag documentation.
