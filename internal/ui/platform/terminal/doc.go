// Package terminal hosts the Bubble Tea based TUI implementation.
//
// Upload Dashboard layout (phase 0 wireframe brought to life in upload_dashboard.go):
//   - upload_dashboard.go: Bubble Tea model, view, and update logic for the experimental upload dashboard.
//   - app_enabled.go: CLI entry point that boots the dashboard when the experimental UI is selected.
//   - components/: reusable lipgloss widgets (sparkline, status card, log list) to share across commands later.
//   - mock helpers (planned): feed sample data when no event stream is attached to keep `tea` tests deterministic.
//
// Additional commands can reuse the same patterns: define a stream of `messages.Event`, pass it to Run, and compose
// their specific layout while sharing the styling tokens.
package terminal
