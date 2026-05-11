## ADDED Requirements

### Requirement: Race-safe concurrent source sync
The sync system SHALL be safe to execute with configured concurrency greater than one, including when multiple active RSS sources are fetched at the same time.

#### Scenario: Concurrent RSS sync has no data races
- **WHEN** multiple enabled RSS sources are synchronized with sync concurrency greater than one under the Go race detector
- **THEN** source fetching and item processing complete without race detector warnings caused by shared adapter or parser state

#### Scenario: Source concurrency is preserved
- **WHEN** sync concurrency is greater than one and multiple sources are eligible for sync
- **THEN** the system continues to process sources concurrently rather than forcing all source sync to run serially

### Requirement: Cancellable source sync operations
Long-running source sync operations SHALL respect the caller's context cancellation where the underlying network or processing step supports cancellation.

#### Scenario: CLI context is cancelled during source fetch
- **WHEN** the command context is cancelled while a source is being fetched
- **THEN** the fetch stops promptly where supported and the source sync result records a failure caused by cancellation

#### Scenario: Cancelled sync does not mark success
- **WHEN** sync is interrupted by context cancellation before all selected sources complete
- **THEN** the overall sync result is not reported as successful unless every selected source completed successfully before cancellation

### Requirement: Consistent item artifact persistence
The sync system SHALL prevent successfully visible item state from referring to missing or stale Markdown artifacts after expected write or database failures, using best-effort rollback where SQLite and filesystem changes cannot be atomic together.

#### Scenario: New item database insert fails after Markdown write
- **WHEN** sync writes Markdown for a new item but cannot create the corresponding SQLite item record
- **THEN** the item is not listed as saved
- **AND** the system attempts to remove the newly written Markdown artifact
- **AND** the source sync result contains an error for that item

#### Scenario: Changed item metadata update fails after Markdown rewrite
- **WHEN** sync saves the previous Markdown version and rewrites the current Markdown but cannot commit the SQLite version and item metadata update
- **THEN** the item metadata is not reported as successfully updated
- **AND** the system attempts to restore the previous current Markdown from the saved version artifact
- **AND** the source sync result contains an error for that item

#### Scenario: Metrics update does not affect item artifact consistency
- **WHEN** optional metrics enrichment fails after item Markdown and SQLite item metadata are saved
- **THEN** the item sync remains successful
- **AND** Markdown content hash and version are not changed solely because metrics failed
