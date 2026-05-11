## ADDED Requirements

### Requirement: TUI startup
The system SHALL open the terminal user interface when requested by the default entry point or explicit TUI command.

#### Scenario: Default command starts TUI
- **WHEN** the user runs `feedctl`
- **THEN** the TUI starts using the effective config and runtime database

#### Scenario: Explicit command starts TUI
- **WHEN** the user runs `feedctl tui`
- **THEN** the TUI starts using the effective config and runtime database

#### Scenario: Startup sync is enabled
- **WHEN** the TUI starts and `sync_on_startup` is enabled
- **THEN** the TUI starts a sync for active sources without blocking basic navigation longer than necessary

### Requirement: Main TUI views
The TUI SHALL provide a daily reading interface with sections, item list, item preview or reader, status bar, and help modal.

#### Scenario: TUI layout is shown
- **WHEN** the TUI opens successfully
- **THEN** the user can see an item list, a preview or read pane, a source or filter section area, and a compact status bar

#### Scenario: Help modal is shown
- **WHEN** the user presses `?`
- **THEN** the TUI shows available keybindings and actions

#### Scenario: Help modal closes
- **WHEN** the help modal is open and the user presses `Esc` or `q`
- **THEN** the TUI closes the help modal and returns to the previous view

### Requirement: Inbox sections and filters
The TUI SHALL provide primary sections for Inbox, Unread, Starred, Sources, Removed Sources, and All Items.

#### Scenario: Number keys select sections
- **WHEN** the user presses `1`, `2`, `3`, `4`, `5`, or `6`
- **THEN** the TUI switches to Inbox, Unread, Starred, Sources, Removed Sources, or All Items respectively

#### Scenario: Tab cycles sections
- **WHEN** the user presses `Tab`
- **THEN** the TUI moves to the next section

#### Scenario: Shift Tab cycles sections backward
- **WHEN** the user presses `Shift+Tab`
- **THEN** the TUI moves to the previous section

#### Scenario: Removed-source items hidden by default
- **WHEN** the TUI opens the normal Inbox section
- **THEN** items from removed sources are hidden by default

#### Scenario: Removed-source section shows removed items
- **WHEN** the user opens the Removed Sources section
- **THEN** the TUI shows items whose source lifecycle state is `removed`

### Requirement: Vim-like movement
The TUI SHALL support vim-like keyboard navigation as the primary navigation model, with arrow keys as fallback.

#### Scenario: Move selection down
- **WHEN** the user presses `j` or the down arrow
- **THEN** the selected list item moves down when possible

#### Scenario: Move selection up
- **WHEN** the user presses `k` or the up arrow
- **THEN** the selected list item moves up when possible

#### Scenario: Move between panes or enter item
- **WHEN** the user presses `h`, `l`, left arrow, or right arrow
- **THEN** the TUI moves between panes or enters/goes back according to the current view context

#### Scenario: Jump to top or bottom
- **WHEN** the user presses `g` or `G`
- **THEN** the selection moves to the top or bottom of the current list respectively

#### Scenario: Page movement
- **WHEN** the user presses `Ctrl+d`, `Ctrl+u`, `Ctrl+f`, or `Ctrl+b`
- **THEN** the current list or reader moves by half-page or page increments in the expected direction

### Requirement: Search and filtering
The TUI SHALL support searching and filter control from the keyboard.

#### Scenario: Start search
- **WHEN** the user presses `/`
- **THEN** the TUI enters search input mode for the current list or reader context

#### Scenario: Navigate search results
- **WHEN** search results exist and the user presses `n` or `N`
- **THEN** the TUI moves to the next or previous search result respectively

#### Scenario: Open filter menu
- **WHEN** the user presses `f`
- **THEN** the TUI opens a filter menu or filter input for the current item list

#### Scenario: Clear filters
- **WHEN** the user presses `F`
- **THEN** the TUI clears active filters for the current item list

#### Scenario: Toggle removed-source visibility
- **WHEN** the user presses `A`
- **THEN** the TUI toggles whether removed-source items are included in applicable item lists

### Requirement: Item reading and actions
The TUI SHALL allow users to read items and perform item actions from the keyboard.

#### Scenario: Open selected item
- **WHEN** the user selects an item and presses `Enter` or `l`
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

### Requirement: TUI sync controls
The TUI SHALL support manual and periodic sync while running.

#### Scenario: Refresh current list or source
- **WHEN** the user presses `r`
- **THEN** the TUI refreshes the current source or item list as appropriate

#### Scenario: Sync all sources
- **WHEN** the user presses `R`
- **THEN** the TUI starts a sync of all active sources and updates sync status when it completes

#### Scenario: Periodic sync runs
- **WHEN** the TUI is running and the configured sync interval elapses
- **THEN** the TUI starts periodic sync for due active sources

#### Scenario: Source sync fails in TUI
- **WHEN** a source sync fails while the TUI is running
- **THEN** the TUI shows failed sync status without crashing and keeps other views usable

### Requirement: Status bar
The TUI SHALL show a compact status bar with inbox, source, storage, and sync information.

#### Scenario: Status bar shows required fields
- **WHEN** the TUI is open
- **THEN** the status bar shows unread count, source count, removed source count, storage usage, sync status, and latest sync time or current sync indicator

#### Scenario: Status updates after read state change
- **WHEN** the user marks an item read or unread
- **THEN** the status bar unread count updates to reflect the persisted runtime state

#### Scenario: Status updates after storage change
- **WHEN** sync saves new Markdown files or versions
- **THEN** the status bar storage usage updates after runtime storage accounting is refreshed

### Requirement: Quit and back behavior
The TUI SHALL provide predictable keyboard behavior for closing views and quitting.

#### Scenario: Quit from main view
- **WHEN** the user presses `q` from the main view
- **THEN** the TUI exits cleanly

#### Scenario: Escape goes back
- **WHEN** the user presses `Esc` inside a modal, search mode, filter mode, or reader view
- **THEN** the TUI closes the current transient view or returns to the previous context

### Requirement: State persistence across TUI sessions
The TUI SHALL read and write runtime state through SQLite so user changes persist.

#### Scenario: Read state persists after restart
- **WHEN** the user marks an item read in the TUI, exits, and runs `feedctl` again
- **THEN** the item remains read

#### Scenario: Starred state persists after restart
- **WHEN** the user stars an item in the TUI, exits, and runs `feedctl` again
- **THEN** the item remains starred

#### Scenario: Archived state persists after restart
- **WHEN** the user archives an item from the inbox in the TUI, exits, and runs `feedctl` again
- **THEN** the item remains hidden from the normal inbox
