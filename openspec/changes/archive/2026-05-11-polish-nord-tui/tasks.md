## 1. Fullscreen Program and Sizing

- [x] 1.1 Start the Bubble Tea program with fullscreen alternate-screen mode.
- [x] 1.2 Add terminal width and height fields to the TUI model.
- [x] 1.3 Handle `tea.WindowSizeMsg` in `Update()` and store the latest terminal dimensions.
- [x] 1.4 Add safe default dimensions for tests or early renders before a window-size message arrives.

## 2. Nord Styling Foundation

- [x] 2.1 Import and use `github.com/charmbracelet/lipgloss` directly for TUI styling.
- [x] 2.2 Define a small Nord palette and reusable styles for background, header, tabs, selected rows, muted text, borders/separators, item markers, and status states.
- [x] 2.3 Ensure styled output remains readable when terminal color support is limited.

## 3. Full-Window Layout Rendering

- [x] 3.1 Refactor `View()` into focused render helpers for header, tabs, content, preview/reader, help, and status regions.
- [x] 3.2 Render the main view to fill the stored terminal height with stable header, navigation, body, and status regions.
- [x] 3.3 Replace the fixed item truncation limit with a limit derived from available content height.
- [x] 3.4 Add a wide-layout path that uses available width for a list plus preview/reader region.
- [x] 3.5 Add a narrow-layout path that remains usable as a single-column view.

## 4. Selection and Visual Polish

- [x] 4.1 Replace the selected-row `>` marker for item rows with a vertical bar-style marker.
- [x] 4.2 Replace the selected-row `>` marker for source rows with a vertical bar-style marker.
- [x] 4.3 Apply selected-row styling consistently without changing cursor movement behavior.
- [x] 4.4 Style unread, read, starred, sync, and status indicators using the Nord styles.
- [x] 4.5 Style the help view so it matches the main Nord-themed interface.

## 5. Verification

- [x] 5.1 Update TUI tests to cover window-size handling and preservation of existing keybindings.
- [x] 5.2 Add checks that selected item/source rows no longer use `>` and do use a vertical marker.
- [x] 5.3 Add checks that rendered output respects known terminal height in the main view.
- [x] 5.4 Run `gofmt` on changed Go files.
- [x] 5.5 Run `go test ./...`.
- [x] 5.6 Run OpenSpec validation for `polish-nord-tui`.

## 6. Enter Read Behavior

- [x] 6.1 Update OpenSpec artifacts for Enter-to-read behavior.
- [x] 6.2 Mark the selected item read when `Enter` opens the reader.
- [x] 6.3 Keep `l` and right-arrow open behavior unchanged.
- [x] 6.4 Update tests for Enter-to-read behavior.
- [x] 6.5 Run `gofmt`, `go test ./...`, and OpenSpec validation.
