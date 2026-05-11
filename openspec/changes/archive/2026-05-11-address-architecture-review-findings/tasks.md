## 1. Regression Tests First

- [x] 1.1 Add a failing sync regression test for two concurrent sync operations of the same source/item proving the committed markdown file is not removed by the losing insert path
- [x] 1.2 Add a failing test proving duplicate same-source sync is serialized while different source IDs can still run concurrently
- [x] 1.3 Add failing reconciliation tests proving app construction alone does not mutate runtime source state and explicit reconciliation surfaces errors
- [x] 1.4 Add a failing transactional reconciliation test proving partial source lifecycle updates roll back on failure
- [x] 1.5 Add a failing cancellation test proving sync stops after fetch but before markdown/database success when the context is cancelled
- [x] 1.6 Add a failing RSS timeout test using a stalled HTTP server and the default/configured fetch timeout
- [x] 1.7 Add a failing store migration test proving future schema versions are rejected before any mutating command can write
- [x] 1.8 Add failing atomic write tests for config/content writes proving interrupted writes do not leave partial active files
- [x] 1.9 Add an import or API boundary test proving CLI/TUI no longer depend on concrete store rows or public app store/config fields

## 2. Typed Domain and App Boundary

- [x] 2.1 Introduce minimal typed internal values/DTOs for source type, source lifecycle, sync status, item state, and metrics snapshots at package boundaries
- [x] 2.2 Add explicit config-to-domain and store-to-domain mappers that return errors for unknown lifecycle/status/type values
- [x] 2.3 Make concrete `App` store/config fields private and add app-level methods for source listing, item listing/actions, status, storage accounting, sync, and markdown preview
- [x] 2.4 Update CLI commands to call app-level methods instead of importing or coordinating runtime store internals
- [x] 2.5 Update TUI model/update/render flows to consume app DTOs and app operations instead of reading concrete store/config state directly
- [x] 2.6 Keep existing CLI output, JSON contracts, TUI keybindings, and visual behavior unchanged while changing the boundary

## 3. Source Reconciliation Hardening

- [x] 3.1 Move source lifecycle reconciliation out of implicit `app.Open` side effects into an explicit `ReconcileSources(ctx)` application use case
- [x] 3.2 Implement transactional reconciliation in the store layer so active/disabled/removed source updates commit all-or-nothing
- [x] 3.3 Preserve the last valid loaded configuration when reload or reconciliation fails and return the error to the caller
- [x] 3.4 Wire CLI/TUI paths that require reconciled source state to invoke the explicit reconciliation use case
- [x] 3.5 Ensure plain text and JSON CLI modes report reconciliation failures deterministically with non-zero exit status

## 4. Sync Safety, Cancellation, and Timeouts

- [x] 4.1 Add a per-source in-process sync gate and a cross-process SQLite-backed lock or guarded transaction for duplicate source sync
- [x] 4.2 Preserve configured concurrency for different source identities while serializing duplicate source identities
- [x] 4.3 Split sync side effects behind narrow interfaces for source fetch, content writing, item repository, source status repository, metrics enrichment, and clock/time
- [x] 4.4 Extract pure item classification/render decisions so they can be tested without real SQLite, network, or filesystem dependencies
- [x] 4.5 Add context checks before rendering, writing markdown, creating versions, enriching metrics, persisting item metadata, and recording sync success
- [x] 4.6 Convert long-running store calls used by sync to context-aware `ExecContext`/`QueryContext` operations
- [x] 4.7 Add a backwards-compatible configurable fetch timeout with a safe default and apply it to RSS fetching under the caller context
- [x] 4.8 Ensure cancelled or timed-out sync attempts do not mark source sync success and clean up only artifacts owned by the current attempt

## 5. Storage, Filesystem, and Migration Durability

- [x] 5.1 Implement ownership-aware markdown cleanup that checks committed item ownership before removing a path after persistence failure
- [x] 5.2 Standardize an atomic writer that creates temp files in the destination directory, closes, renames, and best-effort fsyncs file and parent directory
- [x] 5.3 Migrate config file writes to the shared atomic writer without changing the declarative config format
- [x] 5.4 Migrate content/version artifact writes to the shared atomic writer and avoid cross-device rename failures for custom content/version directories
- [x] 5.5 Add runtime schema-version validation that rejects applied migration versions greater than the binary's known maximum before write paths proceed
- [x] 5.6 Ensure future-schema errors explain that the user should upgrade feedctl or restore a compatible backup

## 6. Validation and Cleanup

- [x] 6.1 Run targeted tests for sync consistency, source reconciliation, RSS timeout, store migrations, config/content atomic writes, CLI output, and TUI model behavior
- [x] 6.2 Run `gofmt -w` on all changed Go files
- [x] 6.3 Run `go test ./...` and fix any regressions
- [x] 6.4 Run `openspec validate address-architecture-review-findings --strict` and fix any proposal/spec/task validation issues
- [x] 6.5 Review the final diff for behavior-preserving CLI/TUI output and document any intentional stricter error reporting
