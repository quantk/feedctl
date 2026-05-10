## Why

The project is starting from a greenfield repository and needs a reproducible local development environment before application code is added. A Nix dev shell with Go 1.25 and SQLite gives contributors and coding agents a consistent toolchain for building the `feedctl` binary and inspecting the local SQLite runtime database.

## What Changes

- Add a Nix flake-based development shell for the repository.
- Provide Go 1.25.x in the shell as the project compiler/toolchain.
- Provide SQLite CLI tools for database inspection and validation.
- Include Go developer tools useful during implementation, such as language server and standard Go tooling.
- Configure the shell so Go code that uses SQLite drivers with CGO can be built when needed.
- Optionally support direnv via a minimal `.envrc` that enters the flake shell.

## Capabilities

### New Capabilities
- `dev-environment`: Reproducible local development shell and toolchain requirements for the feedctl project.

### Modified Capabilities

None.

## Impact

- Adds repository-level Nix development environment files.
- Establishes Go 1.25.x as the development toolchain.
- Adds SQLite command-line availability in the development environment.
- Does not add application runtime code, CLI commands, database schema, or TUI behavior.
