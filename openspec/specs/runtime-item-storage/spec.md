# runtime-item-storage Specification

## Purpose
TBD - created by archiving change build-rss-inbox-mvp. Update Purpose after archive.
## Requirements
### Requirement: SQLite runtime database
The system SHALL store runtime state in SQLite under the configured data root.

#### Scenario: Runtime database is initialized
- **WHEN** a command or TUI session needs runtime state and the database does not exist
- **THEN** the system creates the SQLite database and initializes the required schema

#### Scenario: Runtime database already exists
- **WHEN** a command or TUI session starts and the database already exists
- **THEN** the system opens the database and applies any pending idempotent migrations before use

### Requirement: Runtime state is separated from Markdown artifacts
SQLite SHALL remain the authoritative runtime store while Markdown files remain readable content artifacts.

#### Scenario: Item metadata is stored
- **WHEN** a fetched item is saved
- **THEN** SQLite stores the item id, source id, source-specific identity, title, URLs, timestamps, Markdown path, content hash, version, and runtime state fields

#### Scenario: Markdown frontmatter disagrees with runtime state
- **WHEN** a Markdown file contains stale frontmatter for read, starred, archived, or other runtime state
- **THEN** SQLite runtime state is treated as authoritative

### Requirement: Item listing commands
The CLI SHALL list items using filters suitable for automation and human inspection.

#### Scenario: List all normal inbox items
- **WHEN** the user runs `feedctl items list`
- **THEN** the command lists non-archived items from active and disabled sources while hiding removed-source items by default

#### Scenario: List unread items
- **WHEN** the user runs `feedctl items list --unread`
- **THEN** the command lists unread items matching the normal item visibility rules

#### Scenario: List removed-source items
- **WHEN** the user runs `feedctl items list --removed-sources`
- **THEN** the command lists items whose source lifecycle state is `removed`

#### Scenario: List items JSON
- **WHEN** the user runs `feedctl items list --json`
- **THEN** the command returns structured JSON item summaries including item id, source id, title, URL, read state, starred state, archived state, published time, fetched time, and Markdown path

### Requirement: Item opening commands
The CLI SHALL expose item commands for opening original URLs and local Markdown paths.

#### Scenario: Open original item URL
- **WHEN** the user runs `feedctl items open ITEM_ID`
- **THEN** the system opens the item's original URL using the configured browser command

#### Scenario: Print item Markdown path
- **WHEN** the user runs `feedctl items markdown ITEM_ID`
- **THEN** the command prints the absolute path to the current Markdown file for that item

#### Scenario: Missing item id
- **WHEN** the user runs an item command with an unknown item id
- **THEN** the command fails with a structured item-not-found error

### Requirement: Read and unread state
The system SHALL track item read/unread state in SQLite and persist it across CLI and TUI sessions.

#### Scenario: New item starts unread
- **WHEN** a new item is saved from sync
- **THEN** the item is unread by default

#### Scenario: Item is marked read
- **WHEN** the user marks an item as read
- **THEN** SQLite records the item as read and the unread count decreases accordingly

#### Scenario: Item is marked unread
- **WHEN** the user marks a read item as unread
- **THEN** SQLite records the item as unread and the unread count increases accordingly

#### Scenario: Read state persists
- **WHEN** the user exits and restarts `feedctl`
- **THEN** previously changed read/unread state is preserved

### Requirement: Starred item state
The system SHALL track starred state in SQLite and persist it across CLI and TUI sessions.

#### Scenario: Item is starred
- **WHEN** the user stars an item
- **THEN** SQLite records the item as starred and it appears in starred filters

#### Scenario: Item is unstarred
- **WHEN** the user unstars an item
- **THEN** SQLite records the item as not starred and it no longer appears in starred filters

### Requirement: Archived-from-inbox item state
The system SHALL allow users to archive items from the active inbox without deleting item records or Markdown files.

#### Scenario: Item is archived from inbox
- **WHEN** the user archives an item from the inbox
- **THEN** SQLite records the item as archived and normal inbox views hide the item

#### Scenario: Archived item content remains available
- **WHEN** an item has been archived from the inbox
- **THEN** its SQLite metadata and Markdown file remain available through all-items or direct item lookup behavior

### Requirement: Removed-source item preservation
The system SHALL preserve items and Markdown files for removed sources.

#### Scenario: Source becomes removed
- **WHEN** a source config file is removed after items have been saved for that source
- **THEN** the source lifecycle becomes removed and the existing item records and Markdown files remain unchanged

#### Scenario: Removed-source item visibility
- **WHEN** items belong to a removed source
- **THEN** normal inbox views hide them by default and removed-source filters can show them

### Requirement: Storage usage accounting
The system SHALL track disk usage for current Markdown files, version files, and the SQLite database.

#### Scenario: Storage command is run
- **WHEN** the user runs `feedctl storage`
- **THEN** the command reports item count, current Markdown size, versions size, database size, and total size

#### Scenario: Storage JSON command is run
- **WHEN** the user runs `feedctl storage --json`
- **THEN** the command reports storage usage as structured JSON with byte counts and human-readable display values

#### Scenario: New item changes storage usage
- **WHEN** a new Markdown item file is saved
- **THEN** the system updates storage accounting for current Markdown files

#### Scenario: New version changes storage usage
- **WHEN** a previous Markdown file is saved as a version
- **THEN** the system updates storage accounting for version files

### Requirement: Storage reconciliation
The system SHALL provide a command to recalculate storage accounting from disk and runtime metadata.

#### Scenario: Reconcile storage
- **WHEN** the user runs `feedctl storage reconcile`
- **THEN** the system scans the configured content directory, versions directory, and database file and updates stored storage accounting

#### Scenario: Reconcile storage JSON
- **WHEN** the user runs `feedctl storage reconcile --json`
- **THEN** the system returns structured JSON describing scanned files, recalculated byte counts, and any missing or orphaned files discovered

### Requirement: Runtime state must not mutate declarative config
Runtime item and source operations SHALL NOT write runtime-only fields into config files.

#### Scenario: Sync updates runtime state
- **WHEN** sync records last sync time, errors, item counts, hashes, versions, or read state
- **THEN** those values are written to SQLite and not to `config.toml` or source TOML files

#### Scenario: User state changes
- **WHEN** the user changes read, starred, or archived state
- **THEN** those values are written to SQLite and not to declarative config files

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

