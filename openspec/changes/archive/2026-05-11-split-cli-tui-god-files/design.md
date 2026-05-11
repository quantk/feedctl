## Context

The current CLI and TUI implementation is functional and well-tested enough for an MVP, but two files have grown beyond comfortable review size:

- `internal/tui/tui.go` is about 991 lines and contains model state, Bubble Tea update logic, rendering, styles, sync commands, filtering/search helpers, and preview helpers.
- `internal/cli/root.go` is about 425 lines and contains root command construction plus every command group and subcommand.

This change is a structural refactor only. It should reduce cognitive load and future diff size without changing behavior. It is intentionally separate from the sync hardening change so architecture cleanup does not get mixed with resilience fixes.

## Goals / Non-Goals

**Goals:**

- Split CLI and TUI god files into cohesive files by responsibility.
- Preserve existing package names and public/internal entry points used by tests and callers.
- Preserve all existing user-facing CLI/TUI behavior.
- Keep tests passing throughout the refactor with small, reviewable moves.
- Add characterization coverage only where existing tests do not adequately lock behavior before moving code.

**Non-Goals:**

- No command, flag, JSON output, TUI keybinding, render theme, or data model changes.
- No package-level rewrite or new dependency injection framework.
- No attempt to fix sync race/context/error-handling issues here; those belong to `harden-sync-runtime-architecture`.
- No moving code out of `internal/cli` or `internal/tui` packages unless a later change justifies it.

## Decisions

### 1. Keep package boundaries stable

Both refactors should remain inside the existing packages:

```text
internal/cli/*.go  package cli
internal/tui/*.go  package tui
```

This avoids churn in imports and keeps all existing unexported helpers available during the split.

Alternatives considered:

- Introduce subpackages such as `internal/tui/render` or `internal/cli/commands`: cleaner boundaries, but likely requires exporting more internals and creates more churn than needed.
- Leave files as-is and rely on comments: does not solve review and navigation pain.

### 2. Split TUI by responsibility, not by arbitrary line count

Target file layout:

```text
internal/tui/
  tui.go          # Run, high-level entry points, constants if small
  model.go        # Model type, constructor, core model helpers
  update.go       # Init/Update and key handling
  render.go       # View/render* functions and layout helpers
  styles.go       # lipgloss colors/styles and style helpers
  commands.go     # Bubble Tea commands: sync/tick/etc.
  markdown.go     # existing Markdown rendering stays separate
```

The exact final names can vary if implementation reveals a better cut, but each file should have a clear responsibility.

### 3. Split CLI by command group

Target file layout:

```text
internal/cli/
  root.go         # Execute, newRootCommand, shared options wiring
  add.go          # add rss/telegram/html commands
  sync.go         # sync command
  sources.go      # sources list/show/test/enable/disable/remove
  config.go       # config path/validate/format
  items.go        # items list/open/markdown
  storage.go      # storage/reconcile
  status.go       # status command
  output.go       # existing output/error helpers
```

Keep shared helpers (`splitTags`, `plainBool`, etc.) in `output.go` or a small `helpers.go` if that becomes clearer.

### 4. Use move-only refactoring where possible

The implementation should first move functions without semantic edits. Any cleanup after the move should be tiny and separately testable. This keeps review focused and reduces risk.

Preferred order:

1. Add/confirm tests that lock current behavior.
2. Split CLI functions by group.
3. Run targeted CLI tests.
4. Split TUI functions by group.
5. Run targeted TUI tests.
6. Run full tests.

### 5. Keep behavior-preservation checks explicit

Before and after the split, tests should verify at least:

- root help still lists core commands;
- representative command JSON/plain output remains stable;
- TUI keybindings for section switching, navigation, actions, help, search/filter, and rendering continue to behave as currently specified.

## Risks / Trade-offs

- Moving many functions can create noisy diffs → split by package and file group, with tests after each step.
- Accidental behavior changes can hide inside refactor → prefer move-only edits and characterization tests before cleanup.
- Too many files can make navigation worse → group by existing command/view responsibilities, not one function per file.
- Concurrent active change overlap with sync hardening → avoid touching sync semantics here; if conflicts arise, apply one change at a time and re-run tests.

## Migration Plan

No runtime migration is required. This is source-code organization only.

Implementation plan:

1. Establish/confirm characterization tests.
2. Split `internal/cli/root.go` by command group.
3. Split `internal/tui/tui.go` by TUI responsibility.
4. Run `go test ./...`.

Rollback strategy: revert the refactor commit; no user data, config, or schema changes are involved.

## Open Questions

- Should future behavior changes require a maximum-file-size convention, or is this a one-time cleanup?
- Should `internal/tui` eventually introduce subpackages once the model/render boundaries stabilize?
