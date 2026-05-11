# cli-runtime Specification

## Purpose
TBD - created by archiving change build-rss-inbox-mvp. Update Purpose after archive.
## Requirements
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

### Requirement: Signal-aware command cancellation
The CLI runtime SHALL propagate process cancellation signals to long-running command execution through context cancellation.

#### Scenario: Interrupt cancels sync command
- **WHEN** the user interrupts a running `feedctl sync` command with an operating-system interrupt signal
- **THEN** the command context is cancelled
- **AND** in-flight source fetches that support context cancellation are asked to stop
- **AND** the command exits with a non-zero status if sync did not complete successfully

#### Scenario: Cancelled JSON command reports failure as JSON
- **WHEN** a command running with `--json` is cancelled before successful completion
- **THEN** the command prints a valid JSON error response with `ok: false`
- **AND** the error response contains a stable cancellation-related error code or message

### Requirement: Long-running command contexts are not replaced
Long-running non-TUI commands SHALL pass the Cobra execution context into app, sync, source test, and external action calls instead of replacing it with an unrelated background context.

#### Scenario: Source test receives command context
- **WHEN** the user runs `feedctl sources test ID`
- **THEN** the source adapter receives the command execution context for network fetching

#### Scenario: Item open receives command context
- **WHEN** the user runs `feedctl items open ITEM_ID`
- **THEN** the configured browser process is started with the command execution context

### Requirement: CLI source-state commands surface reconciliation failures
The CLI SHALL report source reconciliation failures deterministically in both plain text and JSON modes when a command requires reconciled runtime source state.

#### Scenario: Plain text command encounters reconciliation failure
- **WHEN** a source-state CLI command cannot reconcile declarative config with runtime storage
- **THEN** the command SHALL exit non-zero and print a clear error message instead of returning stale success output

#### Scenario: JSON command encounters reconciliation failure
- **WHEN** a JSON-mode source-state CLI command cannot reconcile declarative config with runtime storage
- **THEN** the command SHALL exit non-zero and emit a JSON error response consistent with existing JSON failure contracts

### Requirement: CLI setup does not hide runtime mutations
The CLI SHALL avoid mutating runtime storage during generic command setup before the selected command has chosen whether reconciliation or sync is required.

#### Scenario: Command opens the application for path discovery only
- **WHEN** a CLI command only needs config or data path discovery
- **THEN** application setup SHALL NOT reconcile sources or otherwise mutate runtime storage

### Requirement: Cancelled sync reports failure after any stage
The CLI SHALL treat cancellation after fetch, render, filesystem write, or store persistence boundaries as a failed sync rather than successful completion.

#### Scenario: Interrupt occurs after source fetch
- **WHEN** the user interrupts a sync command after source fetch has returned but before sync success is recorded
- **THEN** the command SHALL exit with cancellation failure and SHALL NOT report the source as successfully synced

