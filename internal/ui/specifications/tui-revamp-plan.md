## Specification Plan

### 1. Vision And Scope
- Provide a unified UI core with a Bubble Tea terminal shell today, but keep the layering clean enough to add web or native shells later without redoing business integrations.
- Deliver a modular component library (panels, tables, forms, notifications) with consistent theming and keyboard navigation.
- Support dual operation modes: **dashboard** (real-time telemetry) and **assistant** (guided inputs for credentials, server URLs, filters, etc.).
- Keep the TUI optional so non-interactive scripts remain unaffected.

### 2. User Journeys
1. **Guided Upload** – user selects server profile, dry-run toggle, target folder, then launches upload while monitoring progress + errors.
2. **Investigate Failures** – user reviews latest batch, drills into failed assets, exports copyable diagnostics.
3. **Multi-command Workspace** – user opens stacks/archives tabs without restarting CLI, sharing cached auth + discovery state.
Each journey feeds into acceptance criteria for UX mockups and component checklists.

### 3. Architecture Blueprint
- Split the UI into a **platform-agnostic core** plus **platform-specific shells**:
   - Core: state containers, reducers, event schemas, and services that translate Immich domain events into UI-friendly summaries.
   - Terminal shell: Bubble Tea MVU implementation that consumes the core models.
   - Future shells (web/native): reuse the core packages while swapping only rendering + interaction layers.
- Define an event bus (channels) for business logic → UI updates, decoupled from command implementations.
- Central theme manager with abstractions for typography, spacing, color tokens that can be mapped to Bubble Tea styles, HTML/CSS, or native UI kits.

### 4. Interaction Design
- Navigation: global shortcuts (`F1` help, `Ctrl+P` command palette, `Tab` cycle focus).
- Forms: inline validation, masked secrets, history of last-used entries.
- Logs: collapsible drawer plus on-demand fullscreen panel; highlight actionable errors.
- Accessibility: avoid relying on color alone, ensure focus indicators, support minimal terminal width.

### 5. Observability Requirements
- Structured events (`ui.Event`) for instrumentation (e.g., time spent in forms, failure counts).
- Trace hooks to correlate UI actions with command pipeline metrics.
- Config flags to emit debug snapshots for test automation.

### 6. Documentation & Assets
- Produce lightweight wireframes per screen stored in `scratchpad/ui/`.
- Update `docs/commands/*` with screenshots and shortcut tables once stable.

### 7. Multi-Interface Strategy
- Keep `internal/ui/core` completely renderer-agnostic so it can back Bubble Tea, web (WASM/server-driven), or native (Fyne/Gio) shells.
- Define shell capability contracts (e.g., `PromptProvider`, `NotificationSink`, `AssetInspector`) so each platform can describe what it supports; the runner picks best available shell.
- Introduce shared theme tokens (color, spacing, typography names) that each shell maps to its own primitives (lipgloss styles, CSS variables, native UI constants).
- Bundle platform shells behind build tags and configuration flags (`--ui=terminal|web|native`) so adding a new interface doesn’t disturb existing binaries.

### 8. Platform-Specific Notes
- **Terminal (Bubble Tea)**
   - MVP leverages lipgloss for theming, bubbles for widgets, and glamour for markdown rendering of help panes.
   - Layout should tolerate terminals as small as 80×24. Implement adaptive panels that collapse to tabs on narrow screens.
   - Keyboard-first interactions with optional mouse support (Bubble Tea supports mouse events; enable once stable).
- **Web (Future)**
   - Consider headless Bubble Tea over WebAssembly as a fast path, or build a small HTTP server that streams state via SSE/WebSocket to a React/Svelte UI.
   - Reuse `internal/ui/core` to serialize state snapshots (JSON) and diff them client-side to minimize bandwidth.
   - Authentication handled by reusing existing CLI credentials; the web UI stays local-only (served on `localhost`) to avoid exposing extra attack surface.
- **Native (Future)**
   - Evaluate Gio or Fyne for cross-platform desktop apps sharing Go business logic.
   - Keep the shell thin: subscribe to the same event bus, render via native widgets, and expose OS notifications for long uploads.
   - Mobile target (Android/iOS) would piggyback on native shell, but Phase 0+1 only ensure the abstractions won’t block it later.

### 9. Platform Selection Flow
1. CLI parses `--ui` flag (`auto` by default).
2. Runner checks compiled-in shells (build tags) and chooses in priority order: requested shell → terminal → legacy TUI → headless logging.
3. If requested shell isn’t compiled, runner emits concise warning explaining how to rebuild with needed tag.
4. Selected shell receives shared config structs (theme, keybindings) plus event channel; on exit, runner reports aggregated stats back to CLI for telemetry consistency.

## Migration Path

### Maintaining The Existing TUI During The Revamp
- **Branch Discipline**: keep the legacy tcell fixes on short-lived `bugfix/*` or `feature/*` branches based off `develop`, merging them quickly so they don’t pile up. Bubble Tea work stays confined to `feature/tui-*` branches.
- **Solo Maintainer Workflow**: triage issues on a fixed cadence (e.g., weekly), tagging them “legacy TUI” vs “revamp” so it’s obvious whether they block the next release. Use checklists in `scratchpad/` to track what’s already patched.
- **Shared Contracts**: the event publisher interfaces introduced in Phase 0 power both UIs; legacy code subscribes just like the new UI so telemetry improvements automatically benefit the current experience without double work.
- **Feature Flags**: retain `--tui-legacy` and `--tui-experimental` toggles so you can smoke-test both paths before tagging a release. When patching the old UI, run regression tests with the legacy flag to ensure nothing regresses as new plumbing lands.
- **Release Criteria**: don’t cut a release unless headless mode, legacy TUI, and (once available) Bubble Tea dashboard mode all pass smoke tests. Capture any known gaps directly in `docs/releases/` so users know which path to pick.
- **Focused Freeze Windows**: when you need deep work on the new UI, schedule a short personal freeze window (e.g., weekend) where no legacy fixes land; document pending legacy work so you can resume quickly afterwards.

### Phase 0 – Foundations
- Introduce `internal/ui/` module with Bubble Tea dependencies guarded by build tags to avoid breaking current release.
- Write adapters that translate existing upload events into neutral `ui.Message` instances.

#### Phase 0 Detailed Plan

**Objectives**
- Establish scaffolding for Bubble Tea without touching existing tcell screens.
- Prove that command pipelines can publish neutral UI messages without coupling to presentation details.
- Keep binary size/regression risk minimal via build tags and feature flags.

**Deliverables**
- `internal/ui/README.md` describing architecture decisions and toggles.
- Baseline package tree with compilable stubs: `internal/ui/core/{state,messages,services}`, `internal/ui/runner`, `internal/ui/platform/{terminal,web,native}`, plus placeholder component interfaces.
- Feature flag plumbing (`--tui-experimental`) recognized by Cobra commands but no visible UI yet.
- Upload pipeline instrumentation that publishes simplified lifecycle events (start/stop/progress/error) to a mock channel consumed by the stub UI package.
- Empty-but-compiling scaffolds for `internal/ui/platform/web` and `internal/ui/platform/native` so future shells have defined extension points.

**Constraints & Guardrails**
- No Bubble Tea runtime started in Phase 0; tests must ensure the new module is effectively a no-op when flag disabled.
- Build tags `//go:build tui` gate Bubble Tea import paths to keep current release reproducible without new deps.
- Target the latest stable Go release for this work while keeping APIs straightforward for core contributors (avoid gratuitous generics).

**Work Breakdown**
1. **Dependency Sandbox** – vendor Bubble Tea and lipgloss versions into `go.mod` under `tools` build tag, document upgrade process.
2. **State Contracts** – define immutable structs for telemetry (`RunStats`, `JobSummary`, `LogEvent`) shared between UI and commands.
3. **Message Bus** – create `ui/messages` package with strongly typed channel payloads and helper publishers (wrapping contexts for cancellation).
4. **Flag Wiring** – extend root Cobra command to parse `--tui-experimental`, store in global config, and emit analytics toggle.
5. **No-op Runner** – implement `ui/app.Run(ctx, cfg, source)` that currently drains events for logging only; placeholder for Phase 1 dashboard.
6. **Docs & ADR** – capture tech choices + roll-out plan in scratchpad and future ADR entry.

**Module Layout & Platform Strategy**
- `internal/ui/core/state` – immutable models + reducers shared by every frontend; no rendering imports allowed.
- `internal/ui/core/messages` – strongly typed events & publishers, reused across shells.
- `internal/ui/core/services` – utilities such as formatting, theming tokens, rate calculation.
- `internal/ui/platform/terminal/...` – Bubble Tea app entry point, components, and layout logic behind the `tui` build tag.
- `internal/ui/platform/web/...` – placeholder directory with interface definitions + integration tests so future web shell can plug in without touching core.
- `internal/ui/platform/native/...` – same idea for a desktop/mobile shell (currently stubs returning errors).
- `internal/ui/testing` – helpers to simulate event streams for unit tests across all shells.
- `internal/ui/runner` – feature-flag aware launcher that selects the proper platform implementation at runtime.
- Build tags:
   - `//go:build ui_terminal` (alias `tui`) for Bubble Tea-specific files.
   - `//go:build ui_web` for future web shell (at the moment only stubs so tags compile even if not used).
   - `//go:build !ui_terminal` fallback implementations to keep binaries small when terminal UI is disabled.
- Dependency management:
   - Keep Bubble Tea + lipgloss imports inside `platform/terminal` with the `ui_terminal` build tag.
   - Web/native shells can add their own deps without leaking into others thanks to separate tags.
   - Document `go test -tags ui_terminal ./...` (and eventually `ui_web`) in `docs/development.md` so contributors run platform-specific suites as they evolve.

**Scaffolding Steps**
1. Create directory skeleton:
   - `internal/ui/core/{state,messages,services}`
   - `internal/ui/platform/{terminal,web,native}`
   - `internal/ui/runner`
   - `internal/ui/testing`
2. Add `doc.go` to each folder explaining purpose and build-tag expectations.
3. Provide initial Go files:
   - `core/state/state.go` with placeholder structs (`RunStats`, `JobSummary`).
   - `core/messages/publisher.go` defining the publisher interface + no-op impl.
   - `core/services/theme.go` exporting token enums.
   - `platform/terminal/app_stub.go` (tagged `ui_terminal`) returning `ErrNotImplemented` until Phase 1.
   - `platform/web/app_stub.go` + `platform/native/app_stub.go` (respective tags) with same signatures.
   - `runner/runner.go` that chooses implementation based on config flag and build availability.
   - `testing/fakes.go` generating fake publishers + event streams for unit tests.
4. Wire `go:generate` comments where helpful (e.g., to auto-create mock publishers once we rely on them).
5. Update `internal/ui/specifications` README to reflect structure and keep diagrams close to code.

**Placeholder Responsibilities (Phase 0)**
- `core/state/state.go`: define structs with exported fields, JSON tags, and helper constructors returning zeroed values (so telemetry publishers have stable targets).
- `core/messages/publisher.go`: expose `Publisher` interface plus `NoopPublisher` that satisfies it; include `NewBufferedPublisher(buffer int)` stub returning error until Phase 1.
- `core/services/theme.go`: export enums/constants only—no rendering logic—to document supported tokens ahead of Bubble Tea stylesheets.
- `platform/terminal/app_stub.go`: implement `func Run(ctx context.Context, cfg Config, src <-chan messages.Event) error` returning `ErrTerminalUIUnavailable` when built without `ui_terminal` tag; include compile-time assertion verifying interface compliance.
- `platform/web/app_stub.go` & `platform/native/app_stub.go`: same signature + error but tagged `ui_web` / `ui_native`, enabling future shells to slot in.
- `runner/runner.go`: inspect CLI/config to decide which platform to invoke; if the requested shell isn’t compiled in, fall back to legacy TUI (Phase 0) or headless logging, emitting a clear warning.
- `testing/fakes.go`: lightweight generators for `RunStats`, `JobSummary`, etc., plus a `MemPublisher` capturing emitted events for assertions.
- `internal/ui/specifications/tui-revamp-plan.md`: remains the single source of truth for architecture notes; keep changelog snippets here until the core UI README is finalized.


**Event Adapter Strategy**
- Introduce `internal/ui/messages.Publisher` interface with methods `AssetQueued`, `AssetUploaded`, `AssetFailed`, `LogLine`, `JobStatus`, etc.
- Provide an implementation backed by buffered channels when `--tui-experimental` is on, and a no-op implementation otherwise (guarded by feature flag, not build tag).
- Modify upload pipeline entry points to depend on the interface (via DI or context values) so other commands can reuse the publisher later.
- Normalize payloads: keep IDs (`asset.UID`), counts, timestamps, plus short status strings. Raw errors stay in business layer but we ship sanitized view-friendly data.
- For Phase 0, route published events into `ui/app/drain.go` which simply logs summaries, ensuring we exercise the plumbing end-to-end.
- Testing approach:
   - Unit tests for publisher verifying non-blocking semantics (use fake channel to assert drop counters when full).
   - Contract tests ensuring upload pipeline emits expected event sequences for basic scenarios (start, progress, success, failure).
   - Fuzz minimal event payloads to guarantee JSON-safe output if we later persist snapshots.

**Upload Command Plumbing (Phase 0)**
- Instantiate the publisher inside `app/upload/up.go` when building the `UpCmd` struct:
   - Honor `--ui` / `--tui-experimental` flags to decide between `messages.NoopPublisher` and the buffered implementation.
   - Store it on `UpCmd` (e.g., `uiPublisher messages.Publisher`).
- Publish lifecycle events at strategic points:
   - `UpCmd.upload`: once adapter discovery begins, send a `RunStats` snapshot marking queue estimation; on failure to init UI, emit `LogEvent` with warning.
   - `handleGroup`: before worker submission, emit `AssetQueued` for each asset that survives filters.
   - `handleAsset` success paths: call `AssetUploaded` with bytes written; on errors, call `AssetFailed` with reason and include retry metadata in the payload.
   - `processUploadedAsset`/`manageAssetAlbums`: when albums/tags applied, surface summary stats through `UpdateStats`.
   - `pauseJobs`/`resumeJobs`: publish `LogEvent` entries styled as “server jobs paused/resumed” so users see admin actions immediately.
- Feed aggregated stats back into the publisher from existing counters (e.g., `uc.app.FileProcessor().GenerateReport()`); Phase 0 can snapshot every time counters change rather than tracking diff events.

**Context & Cancellation Wiring**
- Pass `context.Context` from the CLI entrypoint down to publishers so they can stop emitting once the run is cancelled; embed the publisher in the context for lower-level helpers that can’t reach `UpCmd` easily.
- Ensure `UpCmd.finishing` calls `publisher.Close()` to flush buffered events before returning; `defer uc.uiPublisher.Close()` right after instantiation.

**Future-Proofing For Other Commands**
- Define a small helper `ui.ConfigurePublisher(cfg *config.AppConfig) messages.Publisher` inside `internal/ui/runner` (or new `internal/ui/setup`) so other commands simply call it with their CLI config.
- Keep the upload-specific instrumentation in `app/upload`, but allow `stack`/`archive` to reuse the same publisher creation logic without duplicating flag parsing.

**Testing The Plumbing**
- Unit-test `UpCmd.handleAsset` using `testing.NewMemPublisher()` to assert that success/failure paths emit the expected event types.
- Add integration-style test that runs a tiny upload (using fake adapter) with `--ui=off` but `--tui-experimental` enabled; verify that the publisher drains without panics and that counts match `FileProcessor` report.
- Use race detector runs (`go test -race ./app/upload -run TestUIEvents`) to ensure publisher calls remain thread-safe given the worker pool.

**Acceptance Criteria**
- `go test ./...` passes with `-tags ''` and `-tags tui` (once Phase 1 adds code).
- Running CLI with `--tui-experimental` produces debug log confirming the flag is recognized and events are being published, without rendering UI.
- No regression in current TUI or headless modes (verified with smoke upload run).

**Risks & Mitigations**
- *Risk*: Feature flag plumbing leaks into stable configs. → Keep flag opt-in and hidden from help output until Phase 1.
- *Risk*: Event publishing introduces latency. → Use buffered channels with non-blocking writes + drop counters, measured via metrics.
- *Risk*: Dependency conflicts. → Pin Bubble Tea in `go.mod`, run `go mod tidy` under both tag configurations.

### Phase 1 – Parallel Status Board
- Implement a read-only dashboard mirroring current telemetry (queue size, throughput, errors) while legacy TUI remains default.
- Provide opt-in flag (`--tui-experimental`) and gather feedback.

### Phase 2 – Interactive Upload Flow
- Wire command-line flags into form fields, allowing users to override interactively before run.
- Add confirmation dialogs, real-time progress cards, and actionable error popups.
- Start phasing out legacy tcell views for `upload` once feature parity is hit.

### Phase 3 – Multi-command Enablement
- Generalize screens for `stack` and `archive`, sharing authentication/session widgets.
- Ensure shared background workers publish events understood by each screen, removing residual coupling.

### Phase 4 – Default Switch & Cleanup
- Flip feature flag so Bubble Tea UI is default; keep legacy UI behind `--tui-legacy` during one release cycle.
- Remove unused tcell code after confirmation, updating docs and e2e coverage accordingly.

### Supporting Workstreams
- **Testing**: write smoke tests using `tea_test` harness; add snapshot verification of key screens.
- **Automation**: extend e2e pipeline to toggle both UIs until legacy removal.
- **Change Management**: log progress in `scratchpad/` and prep release notes per phase.
