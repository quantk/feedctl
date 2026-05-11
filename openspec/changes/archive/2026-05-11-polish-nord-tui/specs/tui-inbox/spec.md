## ADDED Requirements

### Requirement: Fullscreen TUI presentation
The TUI SHALL run as a fullscreen terminal interface and render a complete view sized to the current terminal window.

#### Scenario: TUI starts in fullscreen mode
- **WHEN** the user runs `feedctl` or `feedctl tui`
- **THEN** the TUI uses a fullscreen terminal presentation rather than rendering as a short command-output block

#### Scenario: TUI receives terminal dimensions
- **WHEN** the terminal reports its current window size
- **THEN** the TUI stores the width and height for layout rendering

#### Scenario: TUI fills available height
- **WHEN** the TUI renders the main view with known terminal dimensions
- **THEN** the rendered view includes header, navigation, content, and status regions sized to occupy the available terminal height without leaving the interface as a small partial block

### Requirement: Nord-themed visual styling
The TUI SHALL use a Nord-inspired visual theme for the main interface, help view, selected rows, status information, and item state indicators.

#### Scenario: Main interface uses Nord styling
- **WHEN** the TUI renders the main view
- **THEN** titles, tabs, content rows, preview text, borders or separators, and status text use a cohesive Nord-inspired palette with dark backgrounds and cool blue highlights

#### Scenario: Item state indicators are styled
- **WHEN** the TUI renders unread or starred items
- **THEN** unread and starred indicators are visually distinct from normal item text while remaining readable on the Nord background

#### Scenario: Sync status is styled
- **WHEN** the TUI renders sync status in the status region
- **THEN** success, active, and failure states are visually distinguishable while preserving the existing status information

### Requirement: Vertical selection marker
The TUI SHALL render the selected list or source row with a vertical bar-style selection marker and SHALL NOT use `>` as the selected-row indicator.

#### Scenario: Selected item row uses vertical marker
- **WHEN** an item row is selected in an item section
- **THEN** the selected row is prefixed or highlighted with a vertical bar-style marker such as `|`, `│`, or `┃`
- **AND** the selected row is not prefixed with `>`

#### Scenario: Selected source row uses vertical marker
- **WHEN** a source row is selected in the Sources section
- **THEN** the selected row is prefixed or highlighted with a vertical bar-style marker such as `|`, `│`, or `┃`
- **AND** the selected row is not prefixed with `>`

### Requirement: Responsive reading layout
The TUI SHALL adapt the main content layout to the available terminal size while preserving existing navigation and actions.

#### Scenario: Wide terminal renders reading-focused layout
- **WHEN** the terminal is wide enough for multiple content regions
- **THEN** the TUI presents the item or source list and preview or reader content in a clear reading-focused layout that uses the available width

#### Scenario: Narrow terminal remains usable
- **WHEN** the terminal is too narrow for multiple content regions
- **THEN** the TUI renders a single-column layout that keeps the selected list, reader or preview content, and status information usable

#### Scenario: Existing keybindings are preserved
- **WHEN** the TUI layout changes due to terminal size or visual styling
- **THEN** existing keybindings for navigation, sections, search, item actions, sync, help, and quit continue to behave as specified

## MODIFIED Requirements

### Requirement: Item reading and actions
The TUI SHALL allow users to read items and perform item actions from the keyboard.

#### Scenario: Open selected item with Enter
- **WHEN** the user selects an item and presses `Enter`
- **THEN** the TUI opens the item in the reader view
- **AND** the TUI marks the item read and persists the change in SQLite

#### Scenario: Open selected item without changing read state
- **WHEN** the user selects an item and presses `l` or the right arrow
- **THEN** the TUI opens the item in the reader view

#### Scenario: Toggle read state
- **WHEN** the user selects an item and presses `Space`
- **THEN** the TUI toggles the item's read/unread state and persists the change in SQLite

#### Scenario: Mark unread
- **WHEN** the user selects an item and presses `u`
- **THEN** the TUI marks the item unread and persists the change in SQLite

#### Scenario: Toggle starred state
- **WHEN** the user selects an item and presses `s`
- **THEN** the TUI toggles the item's starred state and persists the change in SQLite

#### Scenario: Archive item from inbox
- **WHEN** the user selects an item and presses `a`
- **THEN** the TUI archives the item from the normal inbox and persists the change in SQLite without deleting the Markdown file

#### Scenario: Open original URL
- **WHEN** the user selects an item and presses `o`
- **THEN** the TUI opens the item's original URL using the configured browser command

#### Scenario: Open Markdown in editor
- **WHEN** the user selects an item and presses `e`
- **THEN** the TUI opens the item's current Markdown file using the configured editor command
