# internal-code-organization Specification

## Purpose
TBD - created by archiving change split-cli-tui-god-files. Update Purpose after archive.
## Requirements
### Requirement: Behavior-preserving CLI organization
The implementation SHALL organize CLI command construction into cohesive files by command group while preserving existing CLI behavior.

#### Scenario: Core command surface is unchanged after CLI split
- **WHEN** the user runs `feedctl --help` after the CLI refactor
- **THEN** the output still includes commands for `tui`, `sync`, `add`, `sources`, `config`, `items`, `storage`, and `status`

#### Scenario: CLI command outputs are unchanged after CLI split
- **WHEN** existing CLI tests exercise plain text and JSON command outputs after the CLI refactor
- **THEN** those outputs remain compatible with the behavior specified by existing CLI runtime requirements

### Requirement: Behavior-preserving TUI organization
The implementation SHALL organize TUI model, update, rendering, styling, and command helpers into cohesive files while preserving existing TUI behavior.

#### Scenario: TUI keybindings are unchanged after TUI split
- **WHEN** existing TUI model tests exercise navigation, sections, search, filter, item actions, help, sync triggers, and quit/back behavior after the TUI refactor
- **THEN** the resulting model state and rendered key strings remain compatible with existing TUI inbox requirements

#### Scenario: TUI visual contract is unchanged after TUI split
- **WHEN** existing TUI rendering tests inspect selected rows, status content, Markdown preview behavior, and responsive layout after the TUI refactor
- **THEN** the rendered output remains compatible with existing TUI inbox requirements

### Requirement: Refactor-only implementation boundary
This change SHALL NOT alter user-facing behavior, data formats, or runtime storage schemas.

#### Scenario: No user-facing format changes
- **WHEN** the refactor is complete
- **THEN** CLI command names, flags, exit behavior, JSON envelope shape, TUI keybindings, config file format, SQLite schema, and Markdown frontmatter remain unchanged

#### Scenario: Full test suite remains green
- **WHEN** the refactor is complete
- **THEN** `go test ./...` passes without weakening or removing existing behavior tests

