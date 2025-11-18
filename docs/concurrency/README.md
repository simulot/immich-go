# Concurrency and Performance

This directory contains documentation about Immich-Go's concurrency features and performance characteristics.

## Files

### [Multi-threading Analysis](multi-threading.md)
Detailed analysis of Immich-Go's multi-threading capabilities, including:
- Performance benchmarks with different concurrency levels
- Network bandwidth vs CPU utilization analysis
- Recommendations for optimal `--concurrent-tasks` settings
- Test methodology and results

### [Concurrency Visualization](concurrency.html)
Interactive HTML visualization showing:
- Upload performance across different concurrency levels
- Resource utilization patterns
- Network bottleneck analysis

### [Performance Chart](Concurrency.png)
Visual representation of performance testing results showing the relationship between concurrent uploads and overall throughput.

## Key Findings

From our testing, we've determined that:

1. **Network Bound**: Upload performance is primarily constrained by network bandwidth rather than CPU usage
2. **Optimal Range**: For most users, 4-8 concurrent uploads provide the best balance of speed and stability  
3. **Diminishing Returns**: Beyond 12-16 concurrent uploads, performance gains are minimal and reliability may decrease
4. **CPU Scaling**: Using CPU core count as the default for `--concurrent-tasks` provides a good starting point

## Related Documentation

- [Configuration Options](../configuration.md#performance-configuration) - How to configure concurrency settings
- [Best Practices](../best-practices.md#performance-optimization) - Performance optimization recommendations
- [Upload Command](../commands/upload.md#upload-behavior-options) - Command-line options for concurrency control

## Testing Environment

The analysis was conducted using:
- Various network conditions (Gigabit LAN, cable internet, WiFi)
- Different server configurations (powerful desktop, NAS, cloud instances)
- Multiple file sizes and types (photos, videos, RAW files)
- Concurrent upload ranges from 1 to 20 workers