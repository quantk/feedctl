# dev-environment Specification

## Purpose
TBD - created by archiving change setup-nix-dev-shell. Update Purpose after archive.
## Requirements
### Requirement: Nix flake development shell
The repository SHALL provide a Nix flake development shell as the primary reproducible local development environment.

#### Scenario: Developer enters the shell
- **WHEN** a developer runs `nix develop` from the repository root
- **THEN** Nix enters the default development shell without requiring application source code to exist

#### Scenario: Agent runs a command in the shell
- **WHEN** an external coding agent runs a command through `nix develop -c <command>`
- **THEN** the command runs with the repository development toolchain available

### Requirement: Go 1.25 toolchain
The development shell SHALL provide Go 1.25.x as the Go compiler and toolchain.

#### Scenario: Go version is checked
- **WHEN** `go version` is executed inside the development shell
- **THEN** the reported Go version includes `go1.25`

### Requirement: SQLite command-line tooling
The development shell SHALL provide SQLite command-line tooling for inspecting and validating local SQLite databases.

#### Scenario: SQLite version is checked
- **WHEN** `sqlite3 --version` is executed inside the development shell
- **THEN** the command succeeds and prints a SQLite version

### Requirement: Go development helpers
The development shell SHALL include Go development helper tools for editor and implementation workflows.

#### Scenario: Language server is available
- **WHEN** `gopls version` is executed inside the development shell
- **THEN** the command succeeds

### Requirement: CGO-ready SQLite development
The development shell SHALL be compatible with future Go code that uses CGO-based SQLite integrations.

#### Scenario: CGO setting is inspected
- **WHEN** `go env CGO_ENABLED` is executed inside the development shell
- **THEN** the value is `1`

#### Scenario: pkg-config is available
- **WHEN** `pkg-config --version` is executed inside the development shell
- **THEN** the command succeeds

### Requirement: Optional direnv entry point
The repository SHALL provide a minimal direnv entry point for users who choose to enable direnv.

#### Scenario: Direnv file is inspected
- **WHEN** `.envrc` is read from the repository root
- **THEN** it contains a minimal instruction to use the Nix flake development shell

