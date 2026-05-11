## ADDED Requirements

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
