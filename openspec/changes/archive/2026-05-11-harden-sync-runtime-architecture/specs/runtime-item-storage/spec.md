## ADDED Requirements

### Requirement: Versioned runtime database migrations
The runtime database migration system SHALL track applied schema versions and apply missing migrations in deterministic order while preserving idempotent startup behavior.

#### Scenario: Fresh database applies initial migration
- **WHEN** a command or TUI session opens a runtime database that does not exist
- **THEN** the system creates the database schema
- **AND** records the initial schema migration version as applied

#### Scenario: Existing database is opened without duplicate migration effects
- **WHEN** a command or TUI session opens a database whose current migration version is already applied
- **THEN** migration completes successfully without duplicating schema objects or corrupting existing runtime data

#### Scenario: Older database applies pending migrations
- **WHEN** a database has only a subset of known migration versions recorded
- **THEN** the system applies the remaining migrations in ascending version order before runtime state is used

#### Scenario: Migration failure prevents partial runtime use
- **WHEN** a pending migration fails
- **THEN** opening the runtime store fails
- **AND** normal item/source operations are not executed against a partially migrated database

### Requirement: Storage reconciliation reports consistency gaps
Storage reconciliation SHALL report content files that are missing from disk and Markdown files that are orphaned from runtime item metadata after sync or manual filesystem changes.

#### Scenario: Missing current Markdown is reported
- **WHEN** an item record points to a current Markdown path that no longer exists on disk
- **THEN** storage reconciliation includes that path in the missing files report

#### Scenario: Orphaned current Markdown is reported
- **WHEN** the content directory contains a Markdown file that is not referenced by any item record
- **THEN** storage reconciliation includes that path in the orphaned files report
