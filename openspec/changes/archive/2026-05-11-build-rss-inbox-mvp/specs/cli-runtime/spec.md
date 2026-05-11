## ADDED Requirements

### Requirement: Single binary entry point
The system SHALL build and expose a single executable named `feedctl`.

#### Scenario: Run without arguments
- **WHEN** the user runs `feedctl` with no arguments
- **THEN** the system opens the terminal user interface

#### Scenario: Run explicit TUI command
- **WHEN** the user runs `feedctl tui`
- **THEN** the system opens the terminal user interface

### Requirement: Core command surface
The CLI SHALL provide the MVP command groups and commands for TUI launch, sync, source management, config inspection, item inspection, storage accounting, and status reporting.

#### Scenario: Help lists core commands
- **WHEN** the user runs `feedctl --help`
- **THEN** the output includes commands for `tui`, `sync`, `add`, `sources`, `config`, `items`, `storage`, and `status`

#### Scenario: Unsupported MVP-excluded source type
- **WHEN** the user requests an MVP-excluded source type such as HTML source ingestion
- **THEN** the CLI returns a structured unsupported-source-type error rather than silently creating a source

### Requirement: Deterministic CLI output
The CLI SHALL produce deterministic, automation-friendly output for non-TUI commands.

#### Scenario: Plain text command succeeds
- **WHEN** the user runs a successful non-TUI command without `--json`
- **THEN** the command prints stable human-readable output and exits with status code `0`

#### Scenario: Plain text command fails
- **WHEN** a non-TUI command fails without `--json`
- **THEN** the command prints a stable error message to stderr and exits with a non-zero status code

### Requirement: JSON mode
Important non-TUI commands SHALL support `--json` with structured success and error responses.

#### Scenario: JSON command succeeds
- **WHEN** the user runs an important command with `--json` and the command succeeds
- **THEN** the command prints valid JSON containing `ok: true`, the performed action, and command-specific data

#### Scenario: JSON command fails
- **WHEN** the user runs an important command with `--json` and the command fails
- **THEN** the command prints valid JSON containing `ok: false`, a stable error code, and a human-readable message

### Requirement: Non-interactive mutation controls
Mutating commands SHALL support `--dry-run` where practical and `--yes` where confirmation would otherwise be required.

#### Scenario: Dry-run does not mutate state
- **WHEN** the user runs a mutating command with `--dry-run`
- **THEN** the command reports the planned action without writing config files, SQLite rows, or Markdown files

#### Scenario: Yes mode avoids hidden prompts
- **WHEN** the user runs a mutating command with `--json --yes`
- **THEN** the command completes or fails without hidden interactive prompts

### Requirement: Runtime path discovery
The CLI SHALL expose commands that report the effective config and data paths.

#### Scenario: Config path is shown
- **WHEN** the user runs `feedctl config path`
- **THEN** the command prints the effective config file path, source directory path, data root path, database path, content directory path, and versions directory path

#### Scenario: Config path JSON is shown
- **WHEN** the user runs `feedctl config path --json`
- **THEN** the command prints the effective paths as structured JSON

### Requirement: Status reporting
The CLI SHALL provide a status command summarizing inbox and sync state.

#### Scenario: Status is shown
- **WHEN** the user runs `feedctl status`
- **THEN** the command reports unread count, source count, removed source count, storage usage, and latest sync status

#### Scenario: Status JSON is shown
- **WHEN** the user runs `feedctl status --json`
- **THEN** the command reports status fields as structured JSON
