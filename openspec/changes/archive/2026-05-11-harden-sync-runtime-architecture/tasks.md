## 1. Race-safe source sync

- [x] 1.1 Add regression coverage for concurrent RSS sync with multiple enabled sources and sync concurrency greater than one.
- [x] 1.2 Run `CGO_ENABLED=1 go test -race ./internal/sync` and confirm the current shared RSS parser race is reproduced before the fix.
- [x] 1.3 Refactor default source routing to use adapter factories or per-call adapter state so production RSS fetches do not share mutable parser state across goroutines.
- [x] 1.4 Preserve existing adapter injection tests and update router tests for factory/default-router behavior.
- [x] 1.5 Run `go test ./internal/source ./internal/sync` and `CGO_ENABLED=1 go test -race ./internal/sync`.

## 2. Context propagation and bounded external work

- [x] 2.1 Add tests showing command/source/metrics operations receive and respect caller context cancellation where practical.
- [x] 2.2 Make CLI execution signal-aware with a cancellable root context.
- [x] 2.3 Replace detached `context.Background()` usage in TUI sync and item actions with propagated command/model context where practical.
- [x] 2.4 Use `exec.CommandContext` for browser/editor launches.
- [x] 2.5 Add bounded timeout behavior for optional Habr metrics enrichment while preserving non-fatal metrics failures.
- [x] 2.6 Run targeted tests for `internal/cli`, `internal/app`, `internal/tui`, and `internal/metrics`.

## 3. Sync consistency and best-effort rollback

- [x] 3.1 Add failure-injection tests for new item Markdown write followed by SQLite insert failure.
- [x] 3.2 Add failure-injection tests for changed item version/current Markdown writes followed by SQLite update failure.
- [x] 3.3 Add a minimal store transaction helper for multi-step item version and item metadata updates.
- [x] 3.4 Implement best-effort cleanup of newly written Markdown when new item DB insert fails.
- [x] 3.5 Implement best-effort restoration of previous current Markdown when changed item DB transaction fails.
- [x] 3.6 Verify metrics failures remain non-fatal and do not alter content hash or item version.
- [x] 3.7 Run `go test ./internal/sync ./internal/store ./internal/content`.

## 4. Versioned SQLite migrations

- [x] 4.1 Add store migration tests for fresh database creation, already-applied v1 startup, and pending migration ordering.
- [x] 4.2 Refactor `store.Migrate` into ordered versioned migration steps using the existing `schema_migrations` table.
- [x] 4.3 Ensure migration failure prevents store open from returning a usable DB handle.
- [x] 4.4 Run `go test ./internal/store`.

## 5. TUI error visibility

- [x] 5.1 Add TUI model tests for manual sync failure, periodic sync failure, reload failure, and selected item action failure.
- [x] 5.2 Change sync messages to carry `sync.Result` and/or errors instead of a generic success marker.
- [x] 5.3 Make model reload and item action helpers record concise visible error messages without corrupting prior usable state.
- [x] 5.4 Render error messages with existing error styling in status/message areas.
- [x] 5.5 Run `go test ./internal/tui`.

## 6. Final verification

- [x] 6.1 Run `gofmt -w` on changed Go files.
- [x] 6.2 Run `go test ./...`.
- [x] 6.3 Run `CGO_ENABLED=1 go test -race ./internal/source ./internal/sync`.
- [x] 6.4 Run `openspec validate --specs --strict`.
- [x] 6.5 Update README or developer notes only if user-visible behavior or recommended verification commands changed.
