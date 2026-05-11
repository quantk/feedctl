## ADDED Requirements

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
