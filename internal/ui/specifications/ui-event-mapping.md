# Legacy UI Event Mapping

This document pairs every counter or signal consumed by the existing tcell UI (see `app/upload/ui.go`) with the event or state update that the new renderer-agnostic pipeline must emit. The goal is one-to-one parity before layering new UX features.

## Discovery Zone

| Legacy label                          | `fileevent.Code`                                | Publisher action (new pipeline)                                                                                           | Notes |
|---------------------------------------|-------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------|-------|
| Images                                | `DiscoveredImage`                               | `AssetTracker.DiscoverAsset` already increments pending; also emit `Publisher.AppendLog` (info) with count/size snapshot.   | Needed for queue + discovery size. |
| Videos                                | `DiscoveredVideo`                               | Same as above.                                                                                                            | |
| Duplicates (local)                    | `DiscardedLocalDuplicate`                       | `AssetTracker.DiscoverAndDiscard`; send `AppendLog` with reason and size delta.                                           | UI shows total discards + size. |
| Already on server                     | `DiscardedServerDuplicate`                      | Same flow as above.                                                                                                       | |
| Filtered (rules)                      | `DiscardedFiltered`                             | `DiscoverAndDiscard` + log event.                                                                                         | |
| Banned                                | `DiscardedBanned`                               | `DiscoverAndDiscard` + log event (warn).                                                                                  | |
| Missing sidecar                       | `ProcessedMissingMetadata`                      | `Logger.Record` already increments; mirror as `Publisher.AppendLog` (warn) tagged `missing_sidecar`.                      | Needed for discovery total row. |
| Total discovered (count/bytes)        | Derived sum of above + tracker pending size     | Update via `Publisher.UpdateStats` extension or dedicated `DiscoverySnapshot` payload (counts/sizes).                      | Consider extending `RunStats`. |

## Processing Zone

| Legacy label              | `fileevent.Code`                  | Publisher action                                                                                     | Notes |
|---------------------------|----------------------------------|-------------------------------------------------------------------------------------------------------|-------|
| Sidecars associated       | `ProcessedAssociatedMetadata`     | `AppendLog` (info) with asset path + sidecar name; update tracker metadata counters if exposed.       | |
| Added to albums           | `ProcessedAlbumAdded`             | `AppendLog` (info) per asset per album; optionally enrich `AssetUploaded` payload tags.               | |
| Stacked                   | `ProcessedStacked`                | `AppendLog` (info) referencing stack key; tracker `SetProcessed` already runs.                        | |
| Tagged                    | `ProcessedTagged`                 | `AppendLog` (info) listing tag value.                                                                 | |
| Metadata updated          | `ProcessedMetadataUpdated`        | Already recorded when upgrades happen; emit `AppendLog` so UI can surface per-asset metadata events. | |

## Status Zone / Gauges

| Legacy metric                 | Source                                                        | Publisher action                                                                                                      | Notes |
|------------------------------|---------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|-------|
| Pending count/bytes          | `AssetTracker.DiscoverAsset` minus processed/discarded/error | Continue using tracker; push snapshot through `Publisher.UpdateStats` extended with pending/discarded/error fields.    | Extend `state.RunStats`. |
| Processed count/bytes        | `RecordAssetProcessed` (success/upgrade)                      | `Publisher.AssetUploaded` already increments `Uploaded`; include `bytes` to derive processed size.                      | Need to also flag upgrades vs fresh uploads. |
| Discarded count/bytes        | `RecordAssetDiscarded`                                        | On discard call, send `Publisher.AssetFailed` with `reason=discarded:<code>` and subtract from pending in stats.        | |
| Error count/bytes            | `RecordAssetError`                                            | Already call `publishAssetFailed`; ensure `reason` maps to code for UI filtering.                                       | |
| Total count/bytes            | Sum of above                                                  | Provided by tracker snapshot (same extension as pending).                                                               | |
| Google Photo preparation bar | `ProcessedAssociatedMetadata` + `ProcessedMissingMetadata`    | Emit periodic `AppendLog` or new `PreparationProgress` event containing `processed`, `total`.                            | |
| Upload gauge                 | `ProcessedUploadSuccess` totals                               | Already represented by `AssetUploaded`; stats should expose `totalDiscovered` to compute percentage.                    | |

## Error Modal Trigger

| Legacy condition                                      | Source events                                        | Publisher/log requirement                                                             |
|-------------------------------------------------------|------------------------------------------------------|---------------------------------------------------------------------------------------|
| “Some errors have occurred…” completion modal message | `ErrorUploadFailed`, `ErrorServerError`, `ErrorFileAccess`, `ErrorIncomplete` totals | Track via `RunStats.Failed` and/or dedicated `HasErrors` flag to display final warning. |

## Server Jobs Sparkline

| Legacy behavior                         | Current source                                      | Publisher mapping                                                                                                      |
|-----------------------------------------|-----------------------------------------------------|------------------------------------------------------------------------------------------------------------------------|
| Poll `AdminImmich.GetJobs` every 250 ms | `UpCmd.runUI` goroutine in `app/upload/ui.go`       | Move polling into `runner` (so it’s independent of upload command) and emit `EventJobsUpdated` via `Publisher.UpdateJobs`. |
| Sparkline shows active/waiting counts   | Derived from `job.JobCounts.Active/Waiting` sums    | Collapse each poll into `[]state.JobSummary` (one per Immich job queue) before pushing to shells.                      |
| Sparkline per job kind (future)         | Not supported today                                 | Include `JobSummary.Kind` (queue name) so shells can render multiple sparklines or stacked bars per job type.          |
| Idle detection (last active timestamp)  | `lastTimeServerActive` updated when `jobCount > 0`  | Include `UpdatedAt` in `JobSummary`; shells can compute idle duration without CLI-specific globals.                    |
| TUI title line “Server’s jobs: …”       | Inline formatting using `jobCount` totals           | Shells render from the `JobSummary` slice; terminal shell can keep sparkline while others use tables/badges.           |

Implementation steps:

1. Provide a lightweight polling helper inside `runner` that calls `AdminImmich.GetJobs` only when at least one shell subscribes to job updates.
2. Translate each Immich job into `state.JobSummary{Name, Kind, Pending, Completed, Failed, UpdatedAt}` and call `Publisher.UpdateJobs(ctx, summaries)`.
3. For the terminal shell, plot active counts as a sparkline; for future shells, expose aggregate metrics plus idle detection using `UpdatedAt` deltas.

## Implementation Notes

1. **Extend `state.RunStats`** to include pending/discarded/error counts and bytes so `UpdateStats` alone can satisfy the discovery/status widgets without inventing new event types.
2. **Augment `publishAsset*` helpers** to accept the originating `fileevent.Code` so log entries can retain precise semantics (e.g., differentiate duplicate vs banned discards).
3. **Batch discovery updates**: emit `UpdateStats` on a timer or every N discoveries to avoid flooding the channel.
4. **Backfill metadata events**: for album/tag/stack events, emitting structured log entries is sufficient because the legacy UI only surfaces counts/text.
5. **Completion warning**: the CLI should continue printing the modal text, but also send a final `AppendLog` at `Level="error"` so new shells can display the same warning inline.

## Phase 0 Rollout Plan

1. **Telemetry primitives**
	- Extend `state.RunStats` (pending/processed/discarded/error counts and bytes, total discovered) plus a `HasErrors` flag.
	- Wire `AssetTracker` callbacks so every `RecordAsset*` mutation results in a stats snapshot pushed through `Publisher.UpdateStats`.
	- Add a small debouncer (e.g., `time.Ticker`) inside `initUIPipeline` so high-volume discovery bursts batch into ~10 updates/second max.

2. **Asset lifecycle hooks**
	- Thread the originating `fileevent.Code` into `publishAssetQueued/Uploaded/Failed` so `AppendLog` entries and `AssetFailed` reasons stay machine-readable.
	- Emit discard-specific logs (`reason="discarded:<code>"`) and ensure bytes sent/removed adjust the new stats fields.

3. **Metadata/processing events**
	- When `ProcessedAssociatedMetadata`, `ProcessedAlbumAdded`, etc., fire, push structured `AppendLog` payloads (tag, album, stack identifiers) so shells can render text lists without re-parsing CLI logs.
	- For Google Takeout prep, introduce a dedicated `PreparationProgress{processed,total}` payload (or reuse stats) so gauges can read from the stream, not the CLI internals.

4. **Job telemetry**
	- Implement a `runner/jobs` poller that fans out `EventJobsUpdated` events with per-kind sparklines data (active history buffer per queue).
	- Terminal shell keeps sparkline per queue; other shells can show badges or charts without re-implementing polling.

5. **Testing & validation**
	- Unit-test `channel_publisher` for bursty inputs and closing semantics, ensuring no goroutine leaks when UI disabled.
	- Add tests around new stats math (queued/discarded/error deltas) and job summary translation.
	- Create golden-event tests for `appendLog` payloads to lock formatting before UI shells consume them.

6. **Migration safety nets**
	- Provide a CLI flag (env) to dump the event stream to stdout for debugging until new shells land.
	- Keep legacy tcell UI opt-in until the new renderer reaches feature parity, but ensure both consume the same publisher interface for identical behavior.
