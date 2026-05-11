## Why

A deep architecture review found consistency, cancellation, durability, and layering risks in the sync/storage/runtime path. These risks can delete persisted content during concurrent sync, hide failed source reconciliation, hang indefinitely on RSS fetches, and make future storage/config changes unsafe to evolve.

## What Changes

- Make source sync for the same source/content artifact race-safe across in-process and cross-process execution.
- Make source lifecycle reconciliation explicit, transactional, and error-visible instead of best-effort during app opening.
- Harden sync cancellation and network fetch boundaries with configurable timeouts and context-aware storage operations.
- Preserve content/config durability with atomic writes on the destination filesystem and safe cleanup ownership rules.
- Reject runtime databases with unknown future schema versions before any writes occur.
- Refactor app/CLI/TUI boundaries so UI layers depend on app-level use cases and DTOs rather than direct storage/config internals.
- Introduce typed internal domain values for source type, lifecycle, sync status, items, and metrics at package boundaries.

## Capabilities

### New Capabilities

_None._

### Modified Capabilities

- `rss-sync-markdown`: Sync must be race-safe for duplicate/concurrent source runs, cancellable between fetch/render/write/store stages, bounded by fetch timeouts, and must not remove artifacts owned by successful database rows.
- `config-source-management`: Source lifecycle reconciliation must be explicit, transactional, error-visible, and must preserve valid runtime state when reload/reconcile fails.
- `runtime-item-storage`: Runtime storage must use context-aware operations for long-running flows, reject future schema versions, and keep item/content artifacts consistent under write failures.
- `cli-runtime`: CLI commands must surface reconciliation/cancellation failures deterministically and avoid hidden runtime mutations in read-only command setup.
- `internal-code-organization`: App, CLI, TUI, sync, source, and store packages must use narrow service boundaries and typed domain values instead of concrete storage/config leakage.

## Impact

- Affected code: `internal/app`, `internal/sync`, `internal/store`, `internal/source`, `internal/content`, `internal/config`, `internal/cli`, and `internal/tui`.
- Affected behavior: concurrent sync safety, source reconciliation error reporting, RSS fetch timeout behavior, database migration safety, and atomic file write durability.
- Tests: add regression tests for concurrent same-source sync, failed reconciliation, cancellation after fetch, RSS timeout, future schema rejection, atomic write failure handling, and app-boundary behavior.
- No expected user-facing command syntax changes; error reporting may become stricter when reconciliation or storage migration fails.
