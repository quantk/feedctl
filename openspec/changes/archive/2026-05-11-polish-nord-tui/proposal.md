## Why

The current TUI is functional but visually plain: it renders only a short text block, uses a raw `>` selection marker, and does not occupy the full terminal window. Improving the presentation will make `feedctl` feel like a polished reading application rather than a command output view.

## What Changes

- Apply a cohesive Nord-inspired visual theme to the TUI, including muted dark backgrounds, cool blue highlights, readable foreground colors, and status colors for sync/read/star state.
- Start the TUI in a fullscreen terminal experience so the interface occupies the full window instead of only the rendered content area.
- Track terminal dimensions and render a layout that fills available height and width gracefully.
- Replace the current `>` cursor prefix with a vertical selection marker (`|`/`│`/`┃`) and selected-row styling.
- Refine the main layout with a clearer header, section tabs, content area, reader/preview area, and bottom status/help line.
- Make `Enter` open the selected item and mark it read, while keeping `l`/right-arrow as open-only navigation.
- Keep existing keybindings, sections, sync behavior, and persistence semantics otherwise unchanged.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `tui-inbox`: add requirements for fullscreen startup behavior, Nord-themed visual styling, non-arrow selection marker, responsive full-window layout, and Enter-to-read item opening behavior.

## Impact

- Affected code: `internal/tui/tui.go` and TUI tests in `internal/tui/tui_test.go`.
- Affected specs: `openspec/specs/tui-inbox/spec.md` via a change delta.
- Dependencies: may promote the existing indirect `github.com/charmbracelet/lipgloss` dependency to a direct dependency if used for styling.
- CLI/API impact: no command-line flags, config keys, storage schema, or JSON output changes.
