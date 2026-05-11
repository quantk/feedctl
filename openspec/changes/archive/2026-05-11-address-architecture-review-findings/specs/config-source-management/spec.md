## ADDED Requirements

### Requirement: Source lifecycle reconciliation is explicit and error-visible
The system SHALL expose source lifecycle reconciliation as an explicit application operation instead of performing hidden best-effort runtime mutations during application construction.

#### Scenario: Application is opened for dependency construction
- **WHEN** the application is opened before a command or TUI action chooses a use case
- **THEN** runtime source lifecycle rows SHALL NOT be mutated as an implicit side effect of construction alone

#### Scenario: Command requires reconciled source state
- **WHEN** a command needs runtime source lifecycle state
- **THEN** it SHALL invoke reconciliation through the application operation and SHALL surface reconciliation errors to the caller

### Requirement: Source lifecycle reconciliation is transactional
The system SHALL apply config-to-runtime source lifecycle changes atomically so partial reconciliation cannot leave sources half-updated.

#### Scenario: Reconciliation fails while updating runtime sources
- **WHEN** reconciliation fails after some source lifecycle changes have been prepared
- **THEN** none of the prepared lifecycle changes SHALL be committed to the runtime database

#### Scenario: Removed source reappears during reconciliation
- **WHEN** a previously removed source is present in declarative config again
- **THEN** reconciliation SHALL restore it consistently in one transaction or leave the previous runtime state unchanged on failure

### Requirement: Failed config reload preserves the last valid loaded state
The system SHALL keep the last valid in-memory configuration when a later reload or reconciliation attempt fails.

#### Scenario: Reload reads invalid source config
- **WHEN** a reload encounters invalid source configuration after a valid configuration is already loaded
- **THEN** the application SHALL report the reload error and SHALL continue using the last valid loaded configuration until a valid reload succeeds

### Requirement: Source config writes are atomic on the destination filesystem
The system SHALL persist source and main config files through atomic writes in the destination directory.

#### Scenario: Source config write is interrupted
- **WHEN** writing a source TOML file fails or is interrupted
- **THEN** the system SHALL leave either the previous complete file or no file, and SHALL NOT leave a partially written TOML file as the active config
