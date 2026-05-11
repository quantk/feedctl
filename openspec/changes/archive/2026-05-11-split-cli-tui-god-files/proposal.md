## Why

The architecture review identified two growing god files: `internal/tui/tui.go` (~991 lines) and `internal/cli/root.go` (~425 lines). They currently mix command construction, state transitions, rendering, I/O wiring, and app calls, which makes future behavior changes harder to review safely.

## What Changes

- Refactor `internal/tui/tui.go` into smaller files by responsibility while preserving package-level APIs and behavior.
- Refactor `internal/cli/root.go` into smaller command-group files while preserving command names, flags, output, exit codes, and tests.
- Keep this as a behavior-preserving structural refactor: no new user-facing features, no CLI/TUI UX changes, no config/data format changes.
- Add or adjust characterization tests only where needed to lock existing behavior before moving code.

## Capabilities

### New Capabilities

- `internal-code-organization`: internal maintainability requirements for behavior-preserving CLI/TUI code organization.

### Modified Capabilities

- None. This change intentionally does not alter existing user-facing spec behavior; existing `cli-runtime` and `tui-inbox` requirements must continue to pass unchanged.

## Impact

- Affected code: `internal/tui`, `internal/cli`, and related tests.
- Affected tests: primarily `internal/tui/tui_test.go` and `internal/cli/root_test.go`; no external behavior assertions should be weakened.
- Compatibility impact: no breaking changes to commands, flags, JSON output, TUI keybindings, config files, SQLite schema, or Markdown artifacts.
- Implementation impact: future CLI/TUI changes should have clearer integration points and smaller diffs.
