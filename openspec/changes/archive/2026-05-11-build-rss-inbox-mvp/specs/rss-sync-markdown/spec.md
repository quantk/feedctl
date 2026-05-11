## ADDED Requirements

### Requirement: RSS sync command
The system SHALL sync RSS sources from the CLI and save fetched items for active sources.

#### Scenario: Sync all active sources
- **WHEN** the user runs `feedctl sync`
- **THEN** the system fetches and processes every active RSS source and skips disabled or removed sources

#### Scenario: Sync one source
- **WHEN** the user runs `feedctl sync --source habr`
- **THEN** the system fetches and processes source `habr` only if it is active

#### Scenario: Sync disabled source
- **WHEN** the user runs `feedctl sync --source habr` and source `habr` is disabled
- **THEN** the command reports that the source is disabled and does not fetch it

### Requirement: Source-level failure isolation
The sync system SHALL isolate source failures so one failed source does not prevent other active sources from syncing.

#### Scenario: One source fails during sync all
- **WHEN** one active source fails to fetch or parse during `feedctl sync`
- **THEN** the system records the failed source error, continues syncing other active sources, and returns a sync result containing both successes and failures

#### Scenario: Sync JSON reports per-source status
- **WHEN** the user runs `feedctl sync --json`
- **THEN** the command returns structured JSON with per-source status, item counts, errors, and overall success state

### Requirement: Sync concurrency limit
The sync system SHALL respect the configured sync concurrency limit.

#### Scenario: Concurrency is configured
- **WHEN** config sets a sync concurrency value
- **THEN** sync processes no more than that many sources concurrently

#### Scenario: Source interval override exists
- **WHEN** a source config defines an interval override
- **THEN** periodic sync scheduling uses the source interval for that source instead of the default interval

### Requirement: RSS feed parsing and normalization
The RSS adapter SHALL fetch RSS, Atom, or JSON Feed compatible feeds and normalize entries into feedctl item candidates.

#### Scenario: Feed parses successfully
- **WHEN** an active RSS source URL returns a valid feed
- **THEN** the adapter extracts feed metadata and item fields including title, URL, canonical URL when available, published time when available, summary/content, author when available, and source-specific id when available

#### Scenario: Feed cannot be parsed
- **WHEN** an active RSS source URL returns content that cannot be parsed as a feed
- **THEN** the source sync fails with a structured parse error and does not create item records or Markdown files for that response

### Requirement: Stable item identity
The sync system SHALL identify items using a deterministic source-specific identity strategy.

#### Scenario: GUID is available
- **WHEN** a feed item contains a stable GUID or item id
- **THEN** the system uses that value as the preferred source item identity

#### Scenario: GUID is unavailable
- **WHEN** a feed item lacks a usable GUID or item id but has a canonical URL or link URL
- **THEN** the system uses the URL as the source item identity

#### Scenario: URL identity is unavailable
- **WHEN** a feed item lacks usable GUID and URL identity fields
- **THEN** the system uses a fingerprint based on source id, normalized title, and published time where available

#### Scenario: Identity kind is recorded
- **WHEN** the system stores or updates an item
- **THEN** it records which identity strategy was used for that item

### Requirement: Markdown is written for new items
The sync system SHALL save every new fetched item immediately as a Markdown file under the configured content directory.

#### Scenario: New item is fetched
- **WHEN** sync processes an item that is not already known for the source
- **THEN** the system writes a Markdown file under the configured content directory before the item becomes visible in normal item views

#### Scenario: Markdown frontmatter is generated
- **WHEN** the system writes a Markdown item file
- **THEN** the file contains YAML-style frontmatter with item id, source id, source name, source type, title, URL, canonical URL when available, published time when available, fetched time, content hash, version, and tags

#### Scenario: Markdown body is readable
- **WHEN** the system writes a Markdown item file
- **THEN** the file contains a readable Markdown body derived from normalized feed content

### Requirement: Deterministic content paths
The content writer SHALL place Markdown files according to the configured path template and prevent path collisions.

#### Scenario: Default path template is used
- **WHEN** no custom Markdown path template is configured
- **THEN** a new item is written under a path shaped like `{source_id}/{year}/{month}/{slug}.md` inside the content directory

#### Scenario: Path collision occurs
- **WHEN** a rendered content path is already assigned to a different item
- **THEN** the system resolves the collision deterministically by adding a stable suffix and records the final path in SQLite

#### Scenario: Unsafe title characters exist
- **WHEN** an item title contains characters unsafe for filenames
- **THEN** the generated slug replaces or removes unsafe characters while keeping the path file-safe

### Requirement: Stable content hashing
The sync system SHALL calculate content hashes from stable normalized content and exclude volatile runtime fields.

#### Scenario: Stable content is unchanged
- **WHEN** an existing item is fetched again and its stable normalized content is unchanged
- **THEN** the calculated content hash matches the stored hash

#### Scenario: Volatile fields change only
- **WHEN** only fetched time, saved time, version, read state, starred state, or other runtime-only fields change
- **THEN** the content hash does not change

### Requirement: Unchanged items are not rewritten
The sync system SHALL avoid rewriting Markdown files when fetched content has not changed.

#### Scenario: Existing item hash is unchanged
- **WHEN** sync processes an existing item and the content hash matches the stored hash
- **THEN** the system updates last-seen or sync metadata without rewriting the current Markdown file or incrementing the item version

### Requirement: Changed items create versions
The sync system SHALL preserve previous Markdown content when an existing item changes.

#### Scenario: Existing item hash changes
- **WHEN** sync processes an existing item and the content hash differs from the stored hash
- **THEN** the system saves the previous current Markdown as a version, writes the new current Markdown, increments the item version, and marks the item as updated

#### Scenario: Version metadata is stored
- **WHEN** a new item version is created
- **THEN** the system records version number, version file path, content hash, creation time, and size metadata in SQLite

### Requirement: Safe local file writes
The content writer SHALL use safe local write behavior for current Markdown files and version files.

#### Scenario: Markdown file write succeeds
- **WHEN** the system writes a Markdown file
- **THEN** it writes through a temporary file and renames it into place where supported by the local filesystem

#### Scenario: Markdown file write fails
- **WHEN** the system cannot write a Markdown file for a fetched item
- **THEN** the item is not marked as successfully saved and the sync result contains a structured file-write error

### Requirement: Sync status recording
The sync system SHALL record source-level sync status in runtime state.

#### Scenario: Source sync succeeds
- **WHEN** a source sync completes successfully
- **THEN** the system records last sync time, success status, item counts, and clears any previous last error for that source

#### Scenario: Source sync fails
- **WHEN** a source sync fails
- **THEN** the system records last sync time, failure status, and a structured last error for that source without changing declarative config
