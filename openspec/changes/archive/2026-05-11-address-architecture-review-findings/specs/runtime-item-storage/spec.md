## ADDED Requirements

### Requirement: Runtime store operations used by long-running flows are context-aware
The system SHALL accept and honor `context.Context` for runtime database operations used by sync, reconciliation, and user-triggered long-running commands.

#### Scenario: Sync context is cancelled before item insert
- **WHEN** sync attempts to insert or update item metadata after its context has been cancelled
- **THEN** the runtime store SHALL return a cancellation error and SHALL NOT commit the item mutation

#### Scenario: Reconciliation context is cancelled during transaction
- **WHEN** source reconciliation is cancelled while a runtime database transaction is active
- **THEN** the transaction SHALL roll back and the caller SHALL receive a cancellation error

### Requirement: Future schema versions are rejected before runtime writes
The system SHALL fail fast when opening a runtime database that contains a schema migration version newer than the binary knows how to handle.

#### Scenario: Database has a future schema version
- **WHEN** the runtime database records an applied schema version greater than the current binary's maximum known migration version
- **THEN** opening the store SHALL fail with an error explaining that the user should upgrade feedctl or restore a compatible backup

#### Scenario: Future schema is detected before command mutation
- **WHEN** a mutating command opens a database with a future schema version
- **THEN** the command SHALL NOT perform any runtime database writes

### Requirement: Item artifact cleanup is ownership-aware
The system SHALL coordinate runtime item rows and local markdown artifacts so failure cleanup cannot remove artifacts owned by an existing committed item.

#### Scenario: Database insert fails after markdown write
- **WHEN** a sync attempt writes a markdown artifact and then the runtime item insert fails
- **THEN** cleanup SHALL remove only artifacts uniquely owned by that failed attempt and SHALL preserve artifacts referenced by existing item rows

#### Scenario: Metadata update fails after version preparation
- **WHEN** a changed item prepares a new version artifact but the runtime metadata update fails
- **THEN** the current committed item artifact SHALL remain readable and the failed version artifact SHALL not become the current item path
