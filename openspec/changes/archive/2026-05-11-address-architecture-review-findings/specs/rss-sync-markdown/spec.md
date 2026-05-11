## ADDED Requirements

### Requirement: Same-source sync is serialized and artifact-safe
The system SHALL prevent concurrent sync attempts for the same source from deleting, overwriting, or corrupting markdown artifacts committed by another successful sync attempt.

#### Scenario: Concurrent sync of the same source writes the same new item
- **WHEN** two sync operations for the same source fetch the same new item at the same time
- **THEN** one operation SHALL commit the item and markdown artifact, and the other SHALL finish without removing or corrupting the committed artifact

#### Scenario: Losing insert attempts cleanup after another operation commits
- **WHEN** a sync attempt writes a markdown file and then loses the database insert to an existing committed item for the same content path
- **THEN** cleanup SHALL NOT remove any file path referenced by the committed item row

#### Scenario: Different sources remain concurrently syncable
- **WHEN** sync-all processes multiple active sources
- **THEN** the system SHALL preserve configured concurrency for different source identities while serializing duplicate work for the same source identity

### Requirement: Source fetches are bounded by timeout and context
The system SHALL execute RSS source fetching under the command or TUI context and a configurable timeout with a safe default.

#### Scenario: RSS endpoint does not respond
- **WHEN** an RSS source endpoint stalls beyond the configured fetch timeout
- **THEN** the sync SHALL fail that source with a timeout error and SHALL continue to isolate other sources according to sync failure isolation rules

#### Scenario: User cancels sync during RSS fetch
- **WHEN** the sync context is cancelled while an RSS request is in progress
- **THEN** the RSS request SHALL stop promptly and the source SHALL NOT be marked as successfully synced

### Requirement: Sync cancellation is honored between side-effect stages
The system SHALL check cancellation before rendering, writing markdown, creating versions, enriching metrics, and recording sync success.

#### Scenario: Context is cancelled after fetch before markdown write
- **WHEN** a source fetch succeeds and the sync context is cancelled before markdown content is written
- **THEN** the system SHALL stop before creating or rewriting markdown artifacts and SHALL NOT mark the source sync as successful

#### Scenario: Context is cancelled before database persistence
- **WHEN** markdown rendering is complete but the sync context is cancelled before item persistence starts
- **THEN** the system SHALL avoid committing item metadata and SHALL clean up only artifacts owned by the cancelled attempt
