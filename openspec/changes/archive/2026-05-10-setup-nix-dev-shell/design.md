## Context

`feedctl` is currently a greenfield repository with OpenSpec metadata but no application source code. Before building the Go CLI/TUI and SQLite-backed runtime state, contributors need a repeatable development environment that can be entered locally or by coding agents.

The environment should support the planned implementation language and storage choice: Go 1.25 and SQLite. It should remain small and focused so it does not over-constrain later decisions about the SQLite Go driver, CLI framework, or TUI libraries.

## Goals / Non-Goals

**Goals:**

- Provide a Nix flake dev shell for common development systems.
- Make Go 1.25.x available as the project Go toolchain.
- Make the SQLite CLI available for inspecting and validating local databases.
- Include useful Go development tools such as `gopls` and standard auxiliary Go tools.
- Keep the shell compatible with future SQLite drivers that may require CGO.
- Add a minimal direnv entry point for users who opt into direnv.

**Non-Goals:**

- Do not initialize the Go module or application package layout.
- Do not choose the SQLite Go driver.
- Do not create the runtime database schema.
- Do not implement feedctl commands, sync, storage, or TUI behavior.
- Do not add CI or packaging outputs beyond the local dev shell.

## Decisions

### Use a flake-based Nix shell

Use `flake.nix` with a `devShells.<system>.default` output. This gives a standard entry point via `nix develop` and makes the environment easy for humans and agents to verify.

Alternatives considered:

- Plain `shell.nix`: simpler, but less explicit about locked inputs and less aligned with modern Nix workflows.
- Ad-hoc installation instructions: easier initially, but not reproducible and harder for agents to validate.

### Use nixpkgs `go_1_25`

The shell should expose nixpkgs `go_1_25`, currently resolving to Go 1.25.x in the available nixpkgs channel. The acceptance check should verify `go version` reports Go 1.25.

Alternatives considered:

- Use unversioned `go`: simpler, but may drift across major/minor releases.
- Pin a downloaded upstream Go tarball: more control, but unnecessary when nixpkgs provides the requested version.

### Include SQLite CLI without selecting a Go SQLite driver

The shell should include the `sqlite` package so `sqlite3` is available. This supports manual inspection of the future `~/.feedctl/feedctl.db` without deciding whether the application later uses `modernc.org/sqlite`, `github.com/mattn/go-sqlite3`, or another driver.

Alternatives considered:

- Include only a Go driver dependency: premature because no Go module exists yet.
- Include no SQLite tooling: would make early database debugging and verification less convenient.

### Keep CGO possible

Set `CGO_ENABLED=1` in the shell and include `pkg-config`. This keeps the environment ready for CGO-based SQLite drivers while still allowing a later pure-Go driver choice.

Alternatives considered:

- Force `CGO_ENABLED=0`: simpler for pure-Go builds, but incompatible with common SQLite drivers that depend on CGO.
- Fully configure static linking now: premature before build and packaging requirements exist.

### Add minimal direnv support

Include `.envrc` with `use flake` so users with direnv can auto-enter the shell. Users without direnv can ignore it and use `nix develop` directly.

Alternatives considered:

- Omit `.envrc`: fewer files, but less convenient for common Nix workflows.
- Add custom direnv logic: unnecessary; `use flake` is enough.

## Risks / Trade-offs

- [Risk] nixpkgs package names or versions may differ across channels. → Mitigation: pin nixpkgs in `flake.lock` and verify `go_1_25` and `sqlite` through `nix develop`.
- [Risk] CGO behavior can vary between Linux and Darwin. → Mitigation: keep the shell minimal, rely on Nix-provided toolchain setup, and defer driver-specific linker flags until a driver is chosen.
- [Risk] `nixos-unstable` may update Go patch versions over time. → Mitigation: commit `flake.lock`; update intentionally when desired.
- [Risk] Additional tools can bloat the shell. → Mitigation: include only Go, SQLite, and small development helpers needed immediately.
