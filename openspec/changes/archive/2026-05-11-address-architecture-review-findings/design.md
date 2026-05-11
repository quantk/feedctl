## Context

The current architecture keeps the public CLI/TUI behavior working, but the review identified several consistency risks in the internals:

- sync can process the same source/content path concurrently and perform file cleanup after a losing database insert;
- app startup performs reconciliation as a hidden side effect and does not make all errors visible to callers;
- sync mixes fetch, render, filesystem, SQLite, metrics, and status recording in one runner, making cancellation and consistency hard to enforce;
- CLI/TUI code reads concrete storage/config state directly, so UI code becomes a second service layer;
- storage/source/config boundaries pass stringly typed configuration structures instead of stable domain values;
- RSS fetching lacks a default timeout, config/content writes have inconsistent durability rules, and runtime migrations do not fail fast on future schema versions.

This change hardens those paths while preserving the existing command surface and TUI UX.

## Goals / Non-Goals

**Goals:**

- Prevent concurrent sync of the same source/item from deleting or corrupting content artifacts.
- Make source reconciliation explicit, transactional, and error-visible.
- Apply context cancellation and configurable timeouts across fetch, render, filesystem, metrics, and store boundaries.
- Add storage safeguards for atomic writes, ownership-aware cleanup, and future schema rejection.
- Refactor package boundaries so CLI/TUI depend on app-level use cases and DTOs, not concrete storage/config internals.
- Introduce typed internal domain values for source type, lifecycle, sync status, item state, and metrics snapshots.
- Preserve existing user-facing command syntax, output contracts unless stricter error reporting is required, and TUI keybindings/layout.

**Non-Goals:**

- No new feed source type or external persistence backend.
- No rewrite of the Bubble Tea TUI or Cobra CLI command tree.
- No user-facing config format migration unless a timeout field is added with a backwards-compatible default.
- No broad schema redesign beyond the minimum migration-safety checks and repositories needed for this hardening.

## Decisions

### 1. Serialize same-source sync and make artifact cleanup ownership-aware

Use a per-source sync gate in the app/sync layer and a cross-process SQLite-backed lock or guarded transaction for the same source identity. The lock scope is source-level, not global, so different sources can still sync concurrently. When a write/insert fails after a markdown artifact is created, cleanup must only remove files that the current attempt owns and must not remove a path already referenced by a committed item row.

Alternatives considered:

- Global sync mutex: simpler but unnecessarily blocks unrelated sources.
- Rely only on database uniqueness: insufficient because filesystem cleanup happens outside the database transaction.
- Randomize content paths for every attempt: avoids collisions but breaks deterministic artifact paths and existing specs.

### 2. Move source reconciliation into an explicit app use case

`app.Open` should construct dependencies and load configuration, but reconciliation should run through an explicit `ReconcileSources(ctx)` use case invoked by commands that need runtime source state. The use case should run storage mutations in a transaction and return errors to CLI/TUI callers. Failed config reloads must not replace the last valid in-memory state.

Alternatives considered:

- Keep best-effort reconciliation during `Open`: preserves current call sites but hides mutations and failure modes.
- Reconcile only in write commands: read-only commands would continue to show stale runtime state after config changes.

### 3. Introduce narrow ports around sync side effects

Split sync orchestration from side effects with small interfaces, for example `SourceFetcher`, `ContentWriter`, `ItemRepository`, `SourceRepository`, `MetricsEnricher`, and `Clock`. Keep implementations in existing packages initially. Extract pure classification/render decisions into functions that can be unit-tested without SQLite or filesystem setup.

Alternatives considered:

- Full domain-driven rewrite: too large for this change.
- Continue testing only through real SQLite/filesystem integration tests: misses cancellation and failure interleavings.

### 4. Use context-aware storage and bounded source fetching

Store APIs used by long-running flows should accept `context.Context` and use `ExecContext`/`QueryContext`. Source fetching must run under the command/TUI context plus a configurable timeout. RSS should receive a default timeout matching Telegram's bounded behavior unless the user overrides it.

Alternatives considered:

- HTTP-client timeout only: helps network hangs but not database waits or cancellation after fetch.
- Context checks only at the start/end of sync: still allows cancelled commands to write artifacts and mark success.

### 5. Add typed internal domain DTOs at package boundaries

Create a small internal domain layer or package-local DTO set for source type, lifecycle, sync status, item state, and metrics snapshots. Config and store packages map to/from these values at their boundaries. CLI/TUI should consume app DTOs rather than store rows.

Alternatives considered:

- Keep string constants in every package: easy short-term but compiler cannot protect invalid state transitions.
- Move all structs into one global domain package immediately: may create a new god package; keep types focused and small.

### 6. Standardize durable writes and migration guards

Use one atomic writer for config and content artifacts: create temp files in the destination directory, write, fsync where supported, close, rename, and fsync the parent directory. For runtime databases, migration must reject a database containing schema versions greater than the binary's known maximum before any write path proceeds.

Alternatives considered:

- Leave config writes as direct `WriteFile`: faster but can produce partial files after crashes.
- Ignore future schema versions: convenient for downgrades but risks silent data corruption.

### 7. Hide storage/config behind app-level use cases

Make concrete app fields private and expose methods for source listing, status, item listing/actions, sync, storage accounting, and markdown preview. CLI/TUI should keep their behavior but call these use cases. This permits store/config schema changes without editing UI code.

Alternatives considered:

- Allow read-only access to store rows: perpetuates tight coupling.
- Build a separate service layer outside app: more ceremony without clear benefit for this codebase size.

## Risks / Trade-offs

- Race fixes can reduce parallelism for duplicate source sync → use source-scoped locking and keep different sources concurrent.
- Transactional reconciliation may make commands fail where they previously succeeded with stale state → return deterministic errors and add tests for JSON/plain outputs.
- Context-aware repository changes can touch many call sites → migrate incrementally through app use cases and keep compatibility wrappers only where needed.
- Atomic fsync behavior differs by platform/filesystem → make best-effort fsync tolerant of unsupported operations while preserving temp-in-target-dir rename.
- Typed DTO introduction can become a large refactor → limit to boundary types required by reviewed findings and avoid broad renames.
- Future schema rejection can block intentional downgrades → error message must explain that the user should upgrade feedctl or restore from backup.

## Migration Plan

1. Add failing regression tests around the highest-risk behavior first: concurrent same-source sync, failed reconciliation, cancellation after fetch, future schema version, and atomic writer failure.
2. Introduce ports/DTOs behind existing behavior and migrate one boundary at a time.
3. Add reconciliation and locking safeguards without changing command syntax.
4. Add timeout config defaulting so existing configs continue to load.
5. Run targeted tests after each step and finish with `go test ./...`.

Rollback strategy: the change is internal and backwards-compatible; if a migration is added for lock/schema metadata, ensure older binaries fail fast on future schema instead of writing.

## Open Questions

- Should the configurable fetch timeout live globally, per source, or both? Default implementation should support a global default and leave room for per-source override.
- Should cross-process sync locking use a dedicated SQLite table or transaction mode only? Prefer a dedicated lock table if tests show transaction mode cannot protect filesystem cleanup ownership clearly.
