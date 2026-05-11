## ADDED Requirements

### Requirement: Default config and data locations
The system SHALL use `~/.config/feedctl` for declarative configuration and `~/.feedctl` for runtime data by default.

#### Scenario: Defaults are used without explicit config
- **WHEN** no custom paths are configured
- **THEN** the effective config file path is `~/.config/feedctl/config.toml`, the source directory is `~/.config/feedctl/sources.d`, and the runtime data root is `~/.feedctl`

#### Scenario: Missing directories are initialized
- **WHEN** a command needs to write config or runtime data and the default directories do not exist
- **THEN** the system creates the required directories before writing

### Requirement: Main TOML config loading
The system SHALL load declarative application settings from TOML config without storing runtime state in that file.

#### Scenario: Main config is loaded
- **WHEN** `~/.config/feedctl/config.toml` exists with data, sources, sync, tui, and markdown sections
- **THEN** the system applies those settings to path resolution, source loading, sync defaults, TUI defaults, and Markdown output behavior

#### Scenario: Runtime fields are rejected from config
- **WHEN** config contains runtime-only fields such as `last_sync_at`, `etag`, `last_error`, item counts, read state, hashes, versions, or disk usage
- **THEN** config validation fails with a structured error identifying the forbidden runtime field

### Requirement: Per-source TOML files
The system SHALL represent each declarative source as exactly one TOML file under the configured sources directory.

#### Scenario: RSS source file is loaded
- **WHEN** a file `~/.config/feedctl/sources.d/habr.toml` contains an RSS source with `id = "habr"`, `type = "rss"`, `name`, `url`, `enabled`, `interval`, and `tags`
- **THEN** the system loads a source definition with the corresponding source id, type, display name, URL, enabled state, sync interval, and user tags

#### Scenario: Source id does not match file-safe rules
- **WHEN** a source file contains an id that is unsafe for filenames or CLI usage
- **THEN** config validation fails with a structured invalid-source-id error

#### Scenario: Duplicate source ids exist
- **WHEN** multiple source files declare the same source id
- **THEN** config validation fails with a structured duplicate-source-id error

### Requirement: Add RSS source command
The CLI SHALL add RSS sources by validating the feed and writing a new per-source TOML file.

#### Scenario: Add RSS source with explicit metadata
- **WHEN** the user runs `feedctl add rss https://example.com/feed.xml --id example --name Example --tags tech,blog --yes`
- **THEN** the system writes `~/.config/feedctl/sources.d/example.toml` with a valid RSS source definition and no runtime fields

#### Scenario: Add RSS source with generated id
- **WHEN** the user runs `feedctl add rss https://example.com/feed.xml --yes` without `--id`
- **THEN** the system fetches feed metadata where needed, generates a stable file-safe source id, and writes one source file for that id

#### Scenario: Add RSS source dry-run
- **WHEN** the user runs `feedctl add rss https://example.com/feed.xml --id example --dry-run --json`
- **THEN** the system reports the planned source id, source type, config path, and discovered feed metadata without writing a source file

#### Scenario: Add RSS source conflict
- **WHEN** the user adds an RSS source whose target source id already exists
- **THEN** the command fails with a structured source-already-exists error and does not overwrite the existing source file unless a future explicit overwrite mode is provided

### Requirement: Source inspection commands
The CLI SHALL support listing, showing, and testing configured sources.

#### Scenario: List sources
- **WHEN** the user runs `feedctl sources list`
- **THEN** the command lists configured sources with id, type, name, URL, enabled state, lifecycle state, tags, and latest sync status when available

#### Scenario: Show source
- **WHEN** the user runs `feedctl sources show habr`
- **THEN** the command shows the declarative source definition and runtime source status for source `habr`

#### Scenario: Test source
- **WHEN** the user runs `feedctl sources test habr`
- **THEN** the command fetches and parses the source without saving items and reports whether the source is usable

### Requirement: Source enable and disable commands
The CLI SHALL allow users to enable or disable configured sources by editing declarative source files only.

#### Scenario: Disable source
- **WHEN** the user runs `feedctl sources disable habr --yes`
- **THEN** the source file for `habr` is updated to `enabled = false` and runtime item data is preserved

#### Scenario: Enable source
- **WHEN** the user runs `feedctl sources enable habr --yes`
- **THEN** the source file for `habr` is updated to `enabled = true` and existing runtime item data remains associated with the source

### Requirement: Source removal command
The CLI SHALL remove a source from declarative config without deleting existing runtime item records or Markdown files.

#### Scenario: Remove source
- **WHEN** the user runs `feedctl sources remove habr --yes`
- **THEN** the source config file for `habr` is removed or moved out of the sources directory, and existing items and Markdown files for `habr` remain intact

#### Scenario: Remove source dry-run
- **WHEN** the user runs `feedctl sources remove habr --dry-run --json`
- **THEN** the system reports the source file that would be removed and the runtime data that would be preserved without changing files

### Requirement: Source lifecycle reconciliation
The system SHALL derive and persist runtime source lifecycle state from configured source files and existing runtime source records.

#### Scenario: Active source
- **WHEN** a source exists in the source directory with `enabled = true`
- **THEN** the runtime lifecycle state for that source is `active`

#### Scenario: Disabled source
- **WHEN** a source exists in the source directory with `enabled = false`
- **THEN** the runtime lifecycle state for that source is `disabled`

#### Scenario: Removed source
- **WHEN** a source has a runtime record but no corresponding source config file exists
- **THEN** the runtime lifecycle state for that source is `removed`

#### Scenario: Removed source reappears
- **WHEN** a source config file reappears with the same source id as a removed source
- **THEN** the runtime lifecycle state returns to `active` or `disabled` based on the source file enabled value

### Requirement: Config validation and formatting
The CLI SHALL validate config and source files and MAY format them without adding runtime state.

#### Scenario: Validate config succeeds
- **WHEN** the user runs `feedctl config validate` and all config files are valid
- **THEN** the command exits successfully and reports that config is valid

#### Scenario: Validate config fails
- **WHEN** the user runs `feedctl config validate --json` and a config file is invalid
- **THEN** the command returns structured JSON with file path, field path, error code, and message

#### Scenario: Format config preserves declarative meaning
- **WHEN** the user runs `feedctl config format --yes`
- **THEN** the system rewrites config files in canonical TOML form without adding runtime-only fields
