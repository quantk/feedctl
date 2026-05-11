## Why

The current MVP already has a clear local-first architecture, but the architecture review found resilience gaps that will become user-visible as source count and background sync usage grow. The most urgent issue is a confirmed data race during concurrent RSS sync (`go test -race ./...` fails), followed by weak error visibility and partial failure handling around sync, storage, and TUI operations.

## What Changes

- Make concurrent source sync safe under Go's race detector, especially for RSS adapter/parser usage.
- Add explicit cancellation and timeout behavior for CLI/TUI-triggered sync, source testing, metrics enrichment, and external open/editor commands where practical.
- Strengthen sync consistency so Markdown files, versions, metrics, and SQLite runtime state do not silently diverge after partial failures.
- Improve TUI error reporting: failed syncs, reload failures, and item action failures must be visible instead of being silently ignored or shown as success.
- Introduce a versioned migration path for future SQLite schema changes while preserving existing idempotent startup behavior.
- Keep public CLI and data formats backwards-compatible; no breaking command or config changes are intended.

## Capabilities

### New Capabilities

- None.

### Modified Capabilities

- `rss-sync-markdown`: concurrent sync safety, bounded/cancellable source fetches, and stronger item write/versioning consistency.
- `runtime-item-storage`: versioned runtime database migrations and clearer storage reconciliation behavior for missing/orphaned artifacts.
- `tui-inbox`: visible TUI errors for sync/reload/item actions and accurate sync status messages.
- `cli-runtime`: signal-aware command execution and cancellation propagation for long-running commands.
- `item-metrics`: bounded optional metrics enrichment that cannot hang sync and remains non-fatal.

## Impact

- Affected packages: `internal/source`, `internal/sync`, `internal/store`, `internal/app`, `internal/tui`, `internal/cli`, `internal/metrics`.
- Test impact: add regression coverage for race-safe concurrent RSS sync, cancellation/timeouts, partial sync failures, TUI error state, and migration behavior.
- Operational impact: more predictable sync behavior under concurrency, better user feedback in TUI, and safer future database evolution.
- Compatibility impact: no planned breaking changes to config files, CLI command names, JSON envelope shape, Markdown frontmatter, or existing SQLite data.
