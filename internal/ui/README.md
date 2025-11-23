# UI Module Overview

This directory hosts the next-generation UI infrastructure for Immich Go. The goal is to keep all user interfaces (terminal, future web/native shells) layered on top of a renderer-agnostic core so business logic remains untouched while new experiences are added.

## Directory Layout

- `specifications/` – living design documents, current source of truth for the revamp plan.
- `core/state` – immutable data models (`RunStats`, `JobSummary`, etc.) shared by every shell.
- `core/messages` – event contracts + publishers used by CLI commands to push updates into the UI bus.
- `core/services` – utility packages (theme tokens, format helpers) with zero rendering dependencies.
- `platform/terminal` – Bubble Tea shell hooks guarded by the `ui_terminal` build tag. At the moment it only exposes stubs.
- `platform/web` – placeholder for a future local web experience.
- `platform/native` – placeholder for potential desktop/mobile shells.
- `runner/` – feature-flag aware launcher that selects the appropriate shell (or falls back to draining events).
- `testing/` – fakes and helpers for unit tests.

## Build Tags & Flags

| Tag | Purpose | Default |
| --- | --- | --- |
| `ui_terminal` | Compiles the Bubble Tea implementation once available. | Disabled |
| `ui_web` | Reserved for the future web shell. | Disabled |
| `ui_native` | Reserved for the future native shell. | Disabled |

With no tags enabled, the runner simply drains UI events to keep publishers non-blocking. To experiment with the terminal shell once it lands, build with:

```
go build -tags ui_terminal ./cmd/immich-go
```

Runtime flags (to be wired during Phase 0/1):

- `--ui=auto|terminal|web|native|off`
- `--tui-experimental` (opt-in until the new UI surpasses the legacy TUI)

## Development Notes

1. Keep all renderer-agnostic types inside `core/*` packages. Platform packages must not import CLI commands directly.
2. Publishers should remain non-blocking; default to `messages.NoopPublisher` when the UI is disabled.
3. Extend `internal/ui/specifications/tui-revamp-plan.md` before undertaking structural changes.
4. Tests that depend on the UI bus should use `internal/ui/testing.MemPublisher` to avoid brittle snapshot assertions.
