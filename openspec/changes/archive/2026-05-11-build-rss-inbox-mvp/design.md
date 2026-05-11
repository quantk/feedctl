## Context

The repository currently contains the reproducible development environment and OpenSpec project setup, but no application implementation. The MVP must establish the initial architecture for a local-first Go terminal application that stores declarative configuration under `~/.config/feedctl`, runtime state under `~/.feedctl`, item content as Markdown files, and operational metadata in SQLite.

The system has two primary entry points: a deterministic CLI for automation and a keyboard-first TUI for daily reading. Both entry points need to share the same application services so command behavior and TUI behavior do not diverge.

## Goals / Non-Goals

**Goals:**

- Produce a single `feedctl` binary, with `feedctl` opening the TUI and explicit subcommands for automation.
- Keep declarative config, runtime SQLite state, and Markdown artifacts clearly separated.
- Support RSS as the only MVP source type.
- Save every new fetched item as Markdown immediately and record its metadata, file path, hash, and version in SQLite.
- Track read/unread, starred, archived-from-inbox, source lifecycle, sync status, and storage usage in SQLite.
- Provide deterministic CLI commands with JSON output for important commands and dry-run/yes behavior for practical mutating commands.
- Provide a Bubble Tea TUI with vim-like navigation, item reading/actions, sync integration, and compact status information.
- Favor simple, recoverable local behavior over complex abstractions.

**Non-Goals:**

- Telegram integration, HTML/CSS selector scraping, JavaScript-rendered page scraping, media downloading, cloud sync, OPML import/export, LLM integration, full-text search, or advanced readability extraction.
- Git integration inside `feedctl`.
- Cross-device conflict resolution.
- Complex plugin systems or source adapter loading mechanisms.

## Decisions

### Shared application service layer

CLI commands and the TUI will call shared app-level services rather than each implementing storage, sync, and source logic directly.

Suggested package shape:

```text
cmd/feedctl/                 CLI entry point
internal/app/                command/TUI-facing services
internal/config/             config and source file loading/validation
internal/store/              SQLite schema, migrations, queries
internal/source/             source interface and RSS adapter
internal/sync/               sync orchestration and concurrency
internal/content/            Markdown rendering, hashing, safe writes, versions
internal/tui/                Bubble Tea models/views/keybindings
```

Rationale: this keeps CLI and TUI behavior consistent and makes the automation interface useful for external agents. The alternative, letting each command directly manipulate storage and files, would be faster to start but would create duplicated behavior and inconsistent edge-case handling.

### Configuration and runtime state boundaries

Declarative config lives under `~/.config/feedctl` by default, with a main `config.toml` and one TOML file per source in `sources.d`. Runtime state lives under `~/.feedctl` by default, with SQLite at `~/.feedctl/feedctl.db`, content under `content`, previous versions under `versions`, and temporary files under `tmp`.

TOML duration values will be stored as strings such as `"5m"` and parsed with Go duration parsing. Source ids and source types will also be strings. Runtime fields such as sync cursors, errors, read state, hashes, versions, and storage usage MUST NOT be written into config files.

Rationale: valid TOML and strict state separation keep source files git-friendly and predictable. The alternative of writing runtime fields back into source files would make config noisy and harder to review.

### SQLite runtime schema and migrations

SQLite will be the authoritative runtime database for sources known to the system, item metadata, item state, content hashes, version metadata, sync cursors/status, and storage accounting. Migrations will run idempotently at startup for CLI/TUI commands that require runtime state.

Initial schema concepts:

```text
runtime_sources(id, type, name, url, lifecycle, enabled, tags_json, last_sync_at, last_error, etag, last_modified, removed_at, ...)
items(id, source_id, source_item_id, identity_kind, title, url, canonical_url, published_at, fetched_at, last_seen_at, content_path, content_hash, version, read_at, starred, archived_at, updated_at, ...)
item_versions(id, item_id, version, content_path, content_hash, created_at, size_bytes, ...)
storage_stats(scope, bytes, item_count, updated_at, ...)
```

Exact column names can evolve during implementation, but the schema must support the required behavior. Content paths should be stored relative to configured content/version roots when practical so data directories can move more easily.

Rationale: SQLite is embedded, local-first, queryable, and appropriate for a single-user terminal tool. Postgres or a daemon would add operational complexity that conflicts with the product goals.

### Source lifecycle reconciliation

On config load for commands that need source state, `feedctl` will reconcile source config files with runtime source records:

```text
config file exists + enabled=true   -> active
config file exists + enabled=false  -> disabled
runtime record exists, config gone  -> removed
config file reappears with same id  -> active or disabled based on config
```

Removed sources keep their items and Markdown files. Normal inbox views hide removed-source items by default, while dedicated filters can show them.

Rationale: users can manage source definitions declaratively without losing historical content. The alternative of deleting runtime state when a file disappears would be surprising and destructive.

### RSS identity and sync behavior

RSS sync will use an adapter based on a feed parser such as `gofeed`. Sync will operate over active sources only, with a configurable concurrency limit and source-level failure isolation.

Item identity will use a stable preference chain:

```text
feed GUID/id if present and stable-looking
else canonical URL
else link URL
else source_id + normalized title + published_at fingerprint
```

The chosen identity kind will be recorded for debugging. Sync result output will include per-source success/failure status, item counts, and structured errors in JSON mode.

Rationale: RSS feeds vary widely in quality. A deterministic identity chain is simple enough for MVP while avoiding obvious duplicates. A more advanced duplicate resolver can be added later.

### Markdown rendering, paths, hashes, and safe writes

Every new item will be rendered to a Markdown file with YAML-style frontmatter and a readable body. The content hash will be calculated from stable normalized content and must exclude volatile fields such as fetched time, save time, version, read state, and starred state.

The configured path template defaults to `{source_id}/{year}/{month}/{slug}.md`, but the implementation must prevent collisions deterministically. If a rendered path already belongs to a different item, the writer will append a stable short item id or numeric suffix. File writes will use temporary files under `~/.feedctl/tmp` and atomic rename into the target content/version directory where supported.

When an existing item changes content hash, the previous current Markdown file will be copied or moved into the versions directory, the new current Markdown will replace it, and SQLite will record the incremented version in one logical operation as far as local filesystem and SQLite boundaries allow.

Rationale: Markdown files are user-facing artifacts, but SQLite remains authoritative for state. Atomic safe writes and reconciliation reduce damage from crashes without requiring complex distributed transactions.

### CLI contract

The CLI will prefer stable, scriptable output. Important commands support `--json`. Mutating commands support `--dry-run` where practical and `--yes` when confirmation would otherwise be required. In `--json --yes` mode commands must not perform hidden prompts.

A common JSON envelope should be used where practical:

```json
{
  "ok": true,
  "action": "sync",
  "dry_run": false,
  "data": {},
  "errors": []
}
```

Rationale: the project is agent-first without embedding an agent. External automation needs predictable output, meaningful exit codes, and no hidden interactivity.

### TUI architecture

The TUI will be implemented with Bubble Tea ecosystem components and will use the same app services as the CLI. The model will maintain current section/filter, item list cursor, preview/read view state, search state, sync status, and status bar data. Background sync will send messages into the TUI update loop rather than mutating view state directly.

MVP views:

```text
Sections/filter pane | Item list | Preview/reader
Status bar
Help/keybindings modal
```

Rationale: Bubble Tea fits a keyboard-first terminal UI and keeps state transitions explicit. A simpler line-oriented UI would be faster but would not satisfy the daily reading UX goals.

## Risks / Trade-offs

- [RSS feed identity is inconsistent] → Record identity kind, use a deterministic fallback chain, and accept that advanced duplicate merging is future work.
- [Filesystem and SQLite can drift after crashes] → Use temp files plus rename, keep operations ordered, and provide `feedctl storage reconcile` for repair/accounting drift.
- [MVP scope is broad] → Implement in vertical layers: CLI/runtime foundation, config/source management, RSS-to-Markdown sync, item/storage state, then TUI.
- [TUI could consume most effort] → Keep the MVP TUI functional and keyboard-complete before making it visually rich.
- [Markdown conversion quality may vary] → Use basic RSS content handling first and avoid promising full article extraction/readability in MVP.
- [JSON contract can sprawl] → Define small common response shapes for status, source, sync, item, and storage commands, then reuse them.

## Migration Plan

There is no existing application data to migrate. Initial implementation will create missing config/runtime directories and initialize the SQLite schema on first use. If a user has partial files from an earlier failed run, commands should either reconcile them or return a structured error explaining the repair command.

Rollback is manual for MVP: remove the generated binary and, if desired, remove `~/.config/feedctl` and `~/.feedctl`. The tool itself must not delete user content without explicit future delete/purge commands.

## Open Questions

- Which exact Go libraries will be selected for CLI, SQLite, TOML, RSS parsing, and Markdown/HTML conversion?
- Should the default path template include a short item id from the start, or only append one on collision?
- Should `feedctl add rss` perform an immediate test fetch by default, or only when requested/when metadata is needed?
- Which CLI commands are required to support `--json` in the first implementation versus soon after MVP?
