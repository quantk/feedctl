## ADDED Requirements

### Requirement: Telegram source files
The system SHALL support declarative Telegram source files in addition to RSS source files.

#### Scenario: Telegram source file is loaded
- **WHEN** a file `~/.config/feedctl/sources.d/tg-example.toml` contains a source with `id = "tg-example"`, `type = "telegram"`, `name`, `url`, `enabled`, `interval`, `tags`, and optional `max_items`
- **THEN** the system loads a source definition with the corresponding source id, type, display name, URL, enabled state, sync interval, tags, and Telegram item limit when present

#### Scenario: Unsupported source type is rejected
- **WHEN** a source file contains a source type other than `rss` or `telegram`
- **THEN** config validation fails with a structured unsupported-source-type error

### Requirement: Add Telegram source command
The CLI SHALL add public Telegram channel sources by validating the public channel page and writing a new per-source TOML file.

#### Scenario: Add Telegram source from username
- **WHEN** the user runs `feedctl add telegram @llm_under_hood --id tg-llm-under-hood --name "LLM под капотом" --tags telegram,llm --yes`
- **THEN** the system writes `~/.config/feedctl/sources.d/tg-llm-under-hood.toml` with a valid Telegram source definition and no runtime fields
- **AND** the stored source URL is canonicalized to the Telegram public web URL for that channel

#### Scenario: Add Telegram source from URL
- **WHEN** the user runs `feedctl add telegram https://t.me/llm_under_hood --yes`
- **THEN** the system validates the public channel page, generates or accepts a file-safe source id, and writes one Telegram source file for that channel

#### Scenario: Add Telegram source dry-run
- **WHEN** the user runs `feedctl add telegram @llm_under_hood --id tg-llm-under-hood --dry-run --json`
- **THEN** the system reports the planned source id, source type, config path, canonical URL, and discovered public channel metadata without writing a source file

#### Scenario: Add Telegram source conflict
- **WHEN** the user adds a Telegram source whose target source id already exists
- **THEN** the command fails with a structured source-already-exists error and does not overwrite the existing source file unless a future explicit overwrite mode is provided

### Requirement: Telegram source testing
The CLI SHALL test configured Telegram sources through the same source inspection workflow used for other supported source types.

#### Scenario: Test Telegram source
- **WHEN** the user runs `feedctl sources test tg-llm-under-hood` for a configured Telegram source
- **THEN** the command fetches and parses the public Telegram channel page without saving items and reports whether the source is usable

#### Scenario: Test Telegram source JSON output
- **WHEN** the user runs `feedctl sources test tg-llm-under-hood --json`
- **THEN** the command returns structured metadata including title, canonical URL or public web URL, and the number of posts found in the bounded fetch
