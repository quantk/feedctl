## ADDED Requirements

### Requirement: UI layers depend on app use cases instead of concrete storage
The CLI and TUI SHALL use application-level use cases and DTOs for source, item, status, sync, storage, and markdown operations instead of directly coordinating storage/config internals.

#### Scenario: CLI lists items
- **WHEN** the CLI needs to list or mutate items
- **THEN** it SHALL call an app-level operation and SHALL NOT depend on runtime storage row types in command logic

#### Scenario: TUI refreshes source and item state
- **WHEN** the TUI refreshes lists, syncs, or applies item actions
- **THEN** it SHALL call app-level operations and SHALL NOT read public concrete store or loaded-config fields directly

### Requirement: App concrete dependencies are encapsulated
The application boundary SHALL keep concrete store and loaded config fields private and expose behavior through methods or narrow interfaces.

#### Scenario: Store implementation changes
- **WHEN** runtime storage DTOs or table mappings change internally
- **THEN** CLI and TUI packages SHALL not require direct changes solely because of storage representation changes

### Requirement: Sync side effects are behind narrow ports
The sync runner SHALL orchestrate through small interfaces for fetching, content writing, item persistence, source status recording, metrics enrichment, and clock/time behavior.

#### Scenario: Sync classifies an unchanged item
- **WHEN** tests exercise item classification and render decisions
- **THEN** they SHALL be able to do so without constructing real SQLite databases, network clients, or filesystem directories

#### Scenario: Store persistence fails during sync
- **WHEN** a repository port returns a persistence error
- **THEN** the sync runner SHALL propagate the error and execute only the cleanup defined for artifacts owned by the current attempt

### Requirement: Boundary states use typed domain values
The system SHALL represent source type, source lifecycle, sync status, item state, and metrics snapshots with typed internal values at package boundaries rather than unconstrained strings.

#### Scenario: New source lifecycle value is added
- **WHEN** a source lifecycle value is introduced or renamed
- **THEN** compiler-checked mappings SHALL identify all affected config, store, source, CLI, and TUI boundaries

#### Scenario: Invalid sync status is encountered from storage
- **WHEN** storage contains an unknown sync status value
- **THEN** the mapping layer SHALL return an explicit error instead of silently passing an arbitrary string to UI or sync logic
