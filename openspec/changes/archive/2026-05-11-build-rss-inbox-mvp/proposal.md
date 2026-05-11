## Why

`feedctl` needs its first useful end-to-end version: a local-first terminal inbox that can add RSS sources, sync items to Markdown files, track runtime state in SQLite, and provide a keyboard-first TUI for daily reading.

This change turns the project specification into an implementable MVP while keeping future source types, cloud sync, scraping, and LLM features out of scope.

## What Changes

- Add a Go `feedctl` binary where running `feedctl` opens the TUI and explicit subcommands provide deterministic automation-friendly behavior.
- Add TOML-based configuration under `~/.config/feedctl`, including one source file per source under `sources.d`.
- Add source management commands for RSS sources, including add/list/show/test/enable/disable/remove behavior.
- Add runtime state under `~/.feedctl`, backed by SQLite and separated from declarative config.
- Add RSS sync that fetches active sources, normalizes items, saves every new item immediately as Markdown, tracks hashes, and creates versions when content changes.
- Add item state for unread/read, starred, and archived-from-inbox behavior.
- Add source lifecycle handling for active, disabled, and removed sources without deleting existing items or Markdown files.
- Add storage usage accounting and reconciliation commands.
- Add a Bubble Tea-based TUI with vim-like navigation, inbox/list/reader/source views, item actions, sync on startup, periodic sync, and a compact status bar.
- Add JSON output and dry-run/yes behavior for important CLI commands where practical.

## Capabilities

### New Capabilities
- `cli-runtime`: Single binary behavior, command surface, deterministic output, JSON mode, dry-run/yes conventions, path/status commands, and runtime initialization expectations.
- `config-source-management`: Declarative TOML config loading, per-source config files, RSS source creation and validation, source lifecycle transitions, and config/source CLI behavior.
- `rss-sync-markdown`: RSS fetching, source-level sync isolation, item identity, normalization, Markdown rendering, content hashing, deterministic file paths, safe writes, and versioning on updates.
- `runtime-item-storage`: SQLite runtime state for sources/items, read/starred/archived state, item lookup/list/open/markdown behavior, storage accounting, and reconciliation.
- `tui-inbox`: Keyboard-first terminal UI, vim-like navigation, item reading/actions, source/filter sections, removed-source visibility, sync integration, and status bar behavior.

### Modified Capabilities
- None.

## Impact

- Adds the initial Go application structure and `feedctl` binary.
- Adds dependencies for CLI command handling, TOML parsing, SQLite access, RSS parsing, Markdown/HTML conversion helpers, and Bubble Tea TUI components.
- Creates local filesystem state under `~/.config/feedctl` and `~/.feedctl` by default, with config overrides where specified.
- Introduces SQLite schema/migration management for runtime state.
- Establishes CLI and TUI behavior that future source adapters and search features will build on.
