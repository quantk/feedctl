## 1. Characterization checks

- [x] 1.1 Run existing CLI tests with `go test ./internal/cli` before refactoring and record the baseline.
- [x] 1.2 Run existing TUI tests with `go test ./internal/tui` before refactoring and record the baseline.
- [x] 1.3 Add focused characterization tests only if an existing command group or TUI behavior that will be moved lacks coverage.
- [x] 1.4 Confirm any new characterization tests fail only when behavior changes, not because of file layout.

## 2. Split CLI command construction

- [x] 2.1 Move `newSyncCommand` into `internal/cli/sync.go` without behavior changes.
- [x] 2.2 Move `newAddCommand` into `internal/cli/add.go` without behavior changes.
- [x] 2.3 Move source commands into `internal/cli/sources.go` without behavior changes.
- [x] 2.4 Move config commands into `internal/cli/config.go` without behavior changes.
- [x] 2.5 Move item commands into `internal/cli/items.go` without behavior changes.
- [x] 2.6 Move storage and status commands into `internal/cli/storage.go` and `internal/cli/status.go` without behavior changes.
- [x] 2.7 Keep `root.go` focused on `Execute`, root command wiring, and shared command registration.
- [x] 2.8 Run `gofmt -w internal/cli` and `go test ./internal/cli`.

## 3. Split TUI implementation

- [x] 3.1 Move model type, constructor, and core model helpers into `internal/tui/model.go` without behavior changes.
- [x] 3.2 Move `Init`, `Update`, and key handling code into `internal/tui/update.go` without behavior changes.
- [x] 3.3 Move Bubble Tea commands (`syncCmd`, `syncSourcesCmd`, `tickCmd`) into `internal/tui/commands.go` without behavior changes.
- [x] 3.4 Move color/style definitions and style helpers into `internal/tui/styles.go` without behavior changes.
- [x] 3.5 Move view/render/layout helpers into `internal/tui/render.go` without behavior changes.
- [x] 3.6 Keep `tui.go` focused on `Run`, package-level constants, and high-level entry points that do not fit better elsewhere.
- [x] 3.7 Run `gofmt -w internal/tui` and `go test ./internal/tui`.

## 4. Refactor cleanup and verification

- [x] 4.1 Remove accidental duplicate imports, dead helpers, or comment drift introduced by file moves.
- [x] 4.2 Verify `internal/cli/root.go` and `internal/tui/tui.go` are no longer god files and each new file has a clear responsibility.
- [x] 4.3 Run `go test ./...`.
- [x] 4.4 Run `openspec validate split-cli-tui-god-files --type change --strict`.
- [x] 4.5 Do not update README unless a user-visible behavior change was accidentally introduced and intentionally kept.
