## 1. Project Skeleton and Dependencies

- [x] 1.1 Initialize the Go module and create the `cmd/feedctl` entry point.
- [x] 1.2 Select and add MVP dependencies for CLI commands, TOML parsing, SQLite, RSS parsing, Markdown/HTML conversion helpers, and Bubble Tea TUI.
- [x] 1.3 Create the internal package layout for `app`, `config`, `store`, `source`, `sync`, `content`, and `tui`.
- [x] 1.4 Add a basic build path that produces a single executable named `feedctl`.
- [x] 1.5 Add initial test fixtures and helpers for isolated config, data roots, RSS feeds, and temporary home directories.

## 2. CLI Foundation

- [x] 2.1 Implement the root command so `feedctl` opens the TUI and `feedctl tui` is the explicit equivalent.
- [x] 2.2 Register MVP command groups for `sync`, `add`, `sources`, `config`, `items`, `storage`, and `status`.
- [x] 2.3 Implement common `--json`, `--dry-run`, and `--yes` flag handling where applicable.
- [x] 2.4 Implement common structured output helpers for success responses, error responses, and meaningful exit codes.
- [x] 2.5 Implement `feedctl config path` with plain text and JSON output.
- [x] 2.6 Implement an initial `feedctl status` command that reads runtime summaries once storage services exist.
- [x] 2.7 Ensure unsupported MVP-excluded source types return structured unsupported-source-type errors.

## 3. Config and Source Management

- [x] 3.1 Implement config structs, default path resolution, `~` expansion, and duration-string parsing.
- [x] 3.2 Implement loading of `~/.config/feedctl/config.toml` with defaults when the file is absent.
- [x] 3.3 Implement loading of one source TOML file per source from the configured `sources.d` directory.
- [x] 3.4 Implement config validation for source id safety, duplicate source ids, required source fields, supported source types, and forbidden runtime fields.
- [x] 3.5 Implement canonical TOML writing for source files without runtime-only fields.
- [x] 3.6 Implement `feedctl config validate` with plain text and JSON error output.
- [x] 3.7 Implement `feedctl config format --yes` for canonical formatting of config and source files.
- [x] 3.8 Implement `feedctl add rss URL` with URL validation, feed metadata fetching, generated source ids, explicit id/name/tags flags, dry-run, JSON output, and conflict protection.
- [x] 3.9 Implement `feedctl sources list`, `feedctl sources show`, and `feedctl sources test`.
- [x] 3.10 Implement `feedctl sources enable`, `feedctl sources disable`, and `feedctl sources remove` with safe config-file mutation, dry-run where practical, and preservation of runtime data.

## 4. SQLite Runtime Store

- [x] 4.1 Implement runtime directory creation for data root, database parent, content directory, versions directory, tmp directory, and logs directory where needed.
- [x] 4.2 Implement SQLite open/close handling and idempotent schema migrations.
- [x] 4.3 Create initial tables for runtime sources, items, item versions, sync status/cursors, storage stats, and schema migration tracking.
- [x] 4.4 Implement source runtime repositories for upsert, lookup, listing, sync status, and lifecycle fields.
- [x] 4.5 Implement reconciliation from declarative source files to runtime lifecycle states: active, disabled, removed, and reappeared.
- [x] 4.6 Implement item repositories for create, update, lookup by id, lookup by source identity, list filters, and Markdown path lookup.
- [x] 4.7 Implement item state methods for read/unread, starred/unstarred, and archived-from-inbox behavior.
- [x] 4.8 Implement version metadata repositories for item version creation and lookup.
- [x] 4.9 Implement storage accounting repositories for current Markdown size, version size, database size, totals, and update timestamps.

## 5. Content and Markdown Storage

- [x] 5.1 Define normalized item/content models shared by source adapters, sync, content writing, and storage.
- [x] 5.2 Implement Markdown rendering with YAML-style frontmatter and readable body content.
- [x] 5.3 Implement stable content hash calculation that excludes volatile fields.
- [x] 5.4 Implement slug generation and path template rendering for content paths.
- [x] 5.5 Implement deterministic path collision resolution and store final paths relative to configured roots where practical.
- [x] 5.6 Implement safe writes through temporary files and atomic rename for current Markdown files.
- [x] 5.7 Implement version-file creation for changed items under the configured versions directory.
- [x] 5.8 Implement content/storage helper tests for frontmatter, hash stability, paths, collisions, safe writes, and version files.

## 6. RSS Sync

- [x] 6.1 Implement the source adapter interface and RSS adapter using the selected feed parser.
- [x] 6.2 Implement RSS feed metadata fetch/test behavior for source add and source test commands.
- [x] 6.3 Implement RSS item normalization for title, URLs, published time, summary/content, author, source-specific id, and tags.
- [x] 6.4 Implement the deterministic item identity chain and record the chosen identity kind.
- [x] 6.5 Implement single-source sync for new items, unchanged items, and changed items with Markdown writes, versioning, item rows, and sync status updates.
- [x] 6.6 Implement all-source sync orchestration with configured concurrency and source-level failure isolation.
- [x] 6.7 Implement `feedctl sync`, `feedctl sync --source`, and `feedctl sync --json` outputs.
- [x] 6.8 Add RSS sync tests using local fixture feeds for success, parse failure, duplicate identity, unchanged hash, changed content, and source failure isolation.

## 7. Item, Storage, and Status CLI

- [x] 7.1 Implement `feedctl items list` with default visibility, `--unread`, `--removed-sources`, and JSON output.
- [x] 7.2 Implement `feedctl items open ITEM_ID` using the configured browser command and structured item-not-found errors.
- [x] 7.3 Implement `feedctl items markdown ITEM_ID` returning the absolute current Markdown path.
- [x] 7.4 Implement app services used by the TUI for marking items read/unread, starring/unstarring, and archiving from inbox.
- [x] 7.5 Implement `feedctl storage` with item count, current Markdown size, versions size, database size, total size, and JSON output.
- [x] 7.6 Implement `feedctl storage reconcile` to scan content, versions, and database files and update storage accounting.
- [x] 7.7 Complete `feedctl status` using source counts, removed-source counts, unread counts, storage usage, and latest sync status.

## 8. TUI Foundation

- [x] 8.1 Implement the Bubble Tea program entry point and root TUI model.
- [x] 8.2 Load effective config, initialize runtime store, reconcile sources, and load initial status for TUI startup.
- [x] 8.3 Implement the main layout with section/filter area, item list, preview or reader pane, status bar, and help modal.
- [x] 8.4 Implement primary sections for Inbox, Unread, Starred, Sources, Removed Sources, and All Items.
- [x] 8.5 Implement item list loading for each section using shared app services and runtime visibility rules.
- [x] 8.6 Implement help modal display and close behavior for `?`, `Esc`, and `q`.

## 9. TUI Navigation, Search, and Actions

- [x] 9.1 Implement vim-like movement keys `j`, `k`, `h`, `l`, `g`, `G`, `Ctrl+d`, `Ctrl+u`, `Ctrl+f`, and `Ctrl+b` with arrow-key fallbacks.
- [x] 9.2 Implement section switching with number keys, `Tab`, and `Shift+Tab`.
- [x] 9.3 Implement item opening with `Enter` and `l`, plus back/quit behavior with `Esc` and `q`.
- [x] 9.4 Implement search mode with `/`, next result `n`, and previous result `N`.
- [x] 9.5 Implement filter menu/control with `f`, clear filters with `F`, and removed-source visibility toggle with `A`.
- [x] 9.6 Implement item actions for toggle read with `Space`, mark unread with `u`, star/unstar with `s`, and archive from inbox with `a`.
- [x] 9.7 Implement opening original URLs with `o` and local Markdown files in the configured editor with `e`.
- [x] 9.8 Update item lists and status bar state after item actions without requiring a restart.

## 10. TUI Sync and Status Bar

- [x] 10.1 Implement startup sync when `sync_on_startup` is enabled without blocking basic navigation longer than necessary.
- [x] 10.2 Implement periodic sync scheduling using default and source-specific intervals.
- [x] 10.3 Implement manual refresh/sync keys `r` and `R`.
- [x] 10.4 Send background sync results into the Bubble Tea update loop as messages rather than mutating view state directly.
- [x] 10.5 Display source sync failures in the TUI without crashing or blocking other views.
- [x] 10.6 Implement a compact status bar showing unread count, source count, removed source count, storage usage, sync status, and last/current sync indicator.

## 11. Integration and Acceptance Tests

- [x] 11.1 Add CLI tests for config path, config validation, source add dry-run/apply, source list/show/test, enable/disable/remove, and JSON errors.
- [x] 11.2 Add sync integration tests proving RSS items are saved as Markdown and indexed in SQLite.
- [x] 11.3 Add tests proving unchanged items do not rewrite Markdown and changed items create versions.
- [x] 11.4 Add tests proving active, disabled, removed, and reappeared source lifecycle behavior.
- [x] 11.5 Add tests proving read/unread, starred, and archived state persist across store reopen.
- [x] 11.6 Add tests proving storage accounting and storage reconciliation report expected byte counts.
- [x] 11.7 Add TUI model/update tests for keybindings, section changes, item actions, status updates, and sync result messages where practical.
- [x] 11.8 Run the end-to-end MVP acceptance scenario with a local or known RSS feed: add source, sync, inspect Markdown, open TUI, navigate, change read state, reopen, and verify persistence.

## 12. Quality and Documentation

- [x] 12.1 Add concise user documentation for the MVP command surface, config file format, source files, sync behavior, and TUI keys.
- [x] 12.2 Ensure generated TOML examples use valid TOML strings and duration strings.
- [x] 12.3 Run `gofmt` and relevant Go checks across the project.
- [x] 12.4 Run the full automated test suite inside the Nix development shell.
- [x] 12.5 Verify OpenSpec status shows the change ready for implementation review.
