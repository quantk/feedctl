## Context

`feedctl` is a local-first CLI/TUI application. The current package split is clear enough for the MVP, but the review found cross-cutting resilience problems around background sync:

```text
CLI/TUI → app → sync → source adapters → content files
                 ↓        ↓
              SQLite   metrics providers
```

A race-detector run currently fails during concurrent RSS sync because the default source router shares one `RSSAdapter`, which shares one `gofeed.Parser`, across sync goroutines. The same area also mixes network fetch, Markdown writes, version writes, SQLite updates, and optional metrics enrichment without a clear consistency boundary. In the TUI, several app errors are ignored, so failures can be shown as successful refresh/sync.

Constraints:

- Preserve local-first data model: declarative TOML config, SQLite runtime state, Markdown content artifacts.
- Preserve existing CLI commands, JSON envelope shape, source files, and Markdown frontmatter.
- Keep the implementation small and test-driven; avoid introducing a broad framework or dependency injection container.
- Prefer explicit constructors/factories and standard-library context/timeout primitives.

## Goals / Non-Goals

**Goals:**

- Make concurrent source sync race-safe under `go test -race`, without reducing configured source concurrency to `1`.
- Make long-running operations cancellable from CLI/TUI contexts where practical.
- Improve sync consistency for item creation and item update/versioning when file or DB operations fail.
- Keep optional metrics enrichment bounded and non-fatal.
- Surface TUI errors for sync, reload, and item actions while keeping the UI usable.
- Add a versioned SQLite migration structure suitable for future schema changes.

**Non-Goals:**

- No new source types.
- No large clean-architecture rewrite, DI framework, or package reshuffle as part of this change.
- No cross-filesystem atomicity guarantee between SQLite and Markdown files; the design provides best-effort rollback plus reconciliation visibility.
- No change to user-visible config keys, default paths, CLI command names, or Markdown schema.
- No background daemon or multi-process locking redesign.

## Decisions

### 1. Use adapter factories for default source routing

The default router should create a fresh adapter instance for each `Fetch`/`Test` call, or otherwise ensure per-call parser state for stateful adapters. RSS parsing is the primary driver: `gofeed.Parser` mutates its internal HTTP client lazily and is not safe to share concurrently.

Preferred shape:

```text
AdapterRouter
  rss      -> func() Adapter { return NewRSSAdapter() }
  telegram -> func() Adapter { return NewTelegramAdapter() }
```

The public `source.Adapter` interface can remain unchanged. Existing tests that inject simple fake adapters can continue to use an adapter-value constructor, but production `NewDefaultAdapter()` should use factories.

Alternatives considered:

- Add a mutex to `RSSAdapter.Fetch`: fixes the race but serializes RSS fetches and hides the fact that the dependency is stateful.
- Force sync concurrency to `1`: avoids the symptom but breaks the configured concurrency requirement.
- Create only a new parser inside `RSSAdapter.Fetch`: smaller, but a factory router is more generally safe for future stateful adapters.

### 2. Propagate cancellation from the command boundary

`cli.Execute` should create a signal-aware root context (`signal.NotifyContext`) and pass it into Cobra execution. App and TUI command helpers should avoid replacing caller context with `context.Background()` for sync, source testing, metrics enrichment, and external commands.

External browser/editor launches should use `exec.CommandContext`. Sync commands scheduled by TUI should receive the TUI command context, and sync results should return to the model as data rather than being collapsed to `true`.

HTTP timeout policy:

- Telegram adapter already has a default timeout client; preserve it.
- RSS fetch remains context-bound through gofeed.
- Habr metrics provider should use a bounded default client or per-fetch timeout because metrics are optional enrichment and must not hold sync indefinitely.

Alternatives considered:

- Add timeout values to config now: more flexible, but expands user-facing config for a hardening change.
- Rely only on library defaults: leaves known unbounded paths (`http.DefaultClient`) in place.

### 3. Treat sync item processing as a small unit of work with best-effort rollback

SQLite and the filesystem cannot be made fully atomic together, so item processing should use a clear order and rollback policy.

For new items:

```text
render markdown
safe-write current file
insert item row
if insert fails: best-effort remove current file
then optional metrics
```

For changed items:

```text
save previous current file as version file
safe-write new current file
in one DB transaction: insert version row + update item row
if DB transaction fails: best-effort restore previous current from version file
then optional metrics
```

Storage reconciliation remains the fallback for detecting leftover inconsistencies. The implementation should return structured source-level errors rather than silently ignoring failures.

Alternatives considered:

- Write DB before files: would make items visible before Markdown exists, violating existing requirements.
- Introduce a full content journal: stronger, but too large for this hardening pass.

### 4. Keep metrics enrichment optional and bounded

Metrics must remain runtime metadata. A metrics provider error or timeout should not fail item sync, should not rewrite Markdown, and should not increment item versions. The enricher should respect context cancellation and provider timeouts.

Alternatives considered:

- Move metrics to a separate command/queue: cleaner long-term, but outside this change.
- Treat metrics failures as source sync failures: contradicts the current optional enrichment model.

### 5. Make TUI model carry failure state explicitly

The TUI should stop discarding errors in model helpers. Reload and action methods should set a visible message with error styling. Sync commands should return a message that includes `sync.Result` and any execution error; `sync ok` should only be displayed when the result is actually OK.

Expected model-level pattern:

```text
operation succeeds  -> message = "... ok", error = ""
operation fails     -> message/error includes concise reason, UI remains usable
```

Alternatives considered:

- Log errors only: not useful for fullscreen TUI users.
- Bubble up errors to terminate TUI: too disruptive for isolated source/action failures.

### 6. Convert migrations from ad-hoc DDL to versioned steps

`store.Migrate()` should continue to be idempotent on startup, but internally it should know which schema versions have been applied. The existing `schema_migrations` table can remain the migration source of truth.

Initial migration version `1` should correspond to the current schema. Future migrations can be appended as ordered migration steps. Migration application should run inside a transaction where SQLite permits it.

Alternatives considered:

- Use an external migration dependency: unnecessary for an embedded local MVP.
- Use only `PRAGMA user_version`: simpler, but the existing table already exists and is tested.

## Risks / Trade-offs

- Race fixes may require changing source router construction APIs → keep compatibility constructors and adapt tests incrementally.
- Best-effort rollback can fail due to filesystem permissions or crashes → storage reconciliation must report missing/orphaned files clearly.
- More context cancellation can expose new error paths → add targeted tests and stable error messages.
- TUI error messages can become noisy → show concise latest error in status/message area, not modal spam.
- Versioned migrations can overcomplicate a small schema → keep migration runner minimal and table-driven.

## Migration Plan

1. Introduce versioned migration runner while preserving the current schema as migration `1`.
2. Verify fresh DB creation and existing v1 DB startup both succeed.
3. Apply sync/source/TUI hardening changes behind existing APIs.
4. Run `go test ./...` after each stage and `go test -race ./internal/sync ./internal/source` before completion.

Rollback strategy: all user-facing formats remain compatible. If an implementation issue is found before release, revert code changes; databases created with the same v1 schema remain readable by the previous version.

## Open Questions

- Should a future change add configurable network timeouts to `config.toml`, or are hard-coded conservative defaults enough for now?
- Should storage reconciliation eventually repair inconsistencies automatically, or only report them as it does today?
- Should metrics move to a separate explicit command/background queue after this hardening pass?
