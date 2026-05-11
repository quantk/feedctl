## Context

`feedctl` already has a Bubble Tea TUI in `internal/tui/tui.go`, but the current view is plain text assembled with `strings.Builder`. The program is started without Bubble Tea's alternate screen mode, the model does not track terminal width or height, and `View()` returns only the natural height of the content. Selection is represented by a raw `>` prefix.

The desired change is visual and interaction-preserving: make the TUI feel like a fullscreen Nord-themed terminal application while keeping existing sections, keybindings, sync behavior, item actions, and persistence semantics intact.

## Goals / Non-Goals

**Goals:**

- Start the TUI in a fullscreen terminal presentation.
- Track terminal dimensions and render a stable layout that fills the available window.
- Apply a Nord-inspired color palette consistently across header, tabs, list rows, reader preview, help, and status line.
- Replace the `>` selected-row marker with a vertical bar-style marker and selected-row styling.
- Preserve existing keyboard behavior and application state transitions.
- Keep the implementation understandable and testable within the existing `internal/tui` package.

**Non-Goals:**

- Adding user-configurable themes or new config fields.
- Adding mouse support.
- Replacing Bubble Tea with another TUI framework.
- Changing storage, sync, source management, or command-line JSON behavior.
- Implementing full Markdown rendering; the preview may remain line-based text with improved framing/truncation.

## Decisions

### Use Bubble Tea alternate screen for fullscreen startup

Use `tea.WithAltScreen()` when creating the Bubble Tea program. This gives the TUI a dedicated fullscreen terminal buffer and restores the shell after exit.

Alternatives considered:
- Manually emit terminal escape sequences: lower dependency impact but more error-prone and less idiomatic with Bubble Tea.
- Keep normal screen mode and pad the view: fills more space but still feels like command output and leaves scrollback artifacts.

### Track terminal size in the model

Add width and height fields to the TUI model and update them from `tea.WindowSizeMsg`. `View()` should use these dimensions to compute available areas for header, tabs, body, preview/reader, and status.

Alternatives considered:
- Hard-code row limits like the current `i > 30` item truncation: simple but cannot adapt to small or large terminals.
- Query terminal size directly from `View()`: couples rendering to environment calls and makes tests harder.

### Use lipgloss for styling and layout primitives

Use `github.com/charmbracelet/lipgloss` for Nord palette styles, borders, padding, alignment, width/height constraints, and joining layout blocks. The dependency already exists indirectly through Bubble Tea and can be promoted to direct if imported by the project.

Alternatives considered:
- Hand-written ANSI sequences: smaller import surface but harder to read, maintain, and test.
- No color library: cannot deliver cohesive visual polish cleanly.

### Keep one package-level Nord style set

Define a small set of reusable styles for background, title, tabs, active tab, selected row, muted text, unread marker, star marker, status success/error/warning, border, and help text. The palette should map closely to Nord colors without requiring runtime theme selection.

Alternatives considered:
- Add config-driven theme selection now: more flexible but outside the requested scope and adds validation/documentation work.
- Inline styles in `View()`: fast initially but makes the renderer hard to evolve.

### Render a responsive main layout

The main view should render as a fullscreen shell:

```text
header/title + summary
section tabs
content area
status/help line
```

For sufficiently wide terminals, the content area should support a two-pane feel: list/source rows on the left and item preview/reader on the right when useful. For narrow terminals, the layout should degrade to a single-column list/reader without breaking navigation.

Alternatives considered:
- Always keep preview below the list: simpler and closer to current behavior, but wastes wide terminal space.
- Always use two panes: attractive on wide screens but poor on narrow terminals.

### Use a vertical bar-style selection marker

Selected rows should use a vertical bar-style marker such as `┃` or `│` and must not use `>` as the selected-row indicator. The marker plus selected-row style creates a calmer, more modern visual focus.

Alternatives considered:
- Plain ASCII `|`: maximum compatibility but less visually refined.
- Block glyph `▌`: very visible but can feel heavy and may render inconsistently in some fonts.

## Risks / Trade-offs

- Unicode glyph rendering may vary by terminal/font → keep the selected marker simple and avoid relying on complex box drawing for meaning.
- Color may be unavailable or low contrast in some terminals → use lipgloss styles that still leave readable text when color is degraded.
- Fullscreen alternate screen changes the user's scrollback expectations → this is standard for TUIs and the shell should be restored on exit.
- Small terminal windows can constrain layout → define minimum-safe rendering paths and truncate/pad content rather than panic or wrap unpredictably.
- Styling can make tests brittle → test semantic output and key state where possible, with focused checks for key visual requirements such as no `>` selected marker and full-window sizing behavior.
