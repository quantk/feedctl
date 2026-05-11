## ADDED Requirements

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
