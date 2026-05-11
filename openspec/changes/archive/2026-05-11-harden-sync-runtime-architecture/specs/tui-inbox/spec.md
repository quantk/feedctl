## ADDED Requirements

### Requirement: TUI operation errors are visible
The TUI SHALL surface errors from sync, reload, and selected item actions in the interface instead of silently ignoring them or reporting success.

#### Scenario: Manual sync failure is shown
- **WHEN** the user presses `R` and one or more selected sources fail to sync
- **THEN** the TUI displays a failed sync status or error message
- **AND** the TUI remains usable for navigation and item reading

#### Scenario: Periodic sync failure is shown
- **WHEN** periodic sync runs and a due source fails
- **THEN** the TUI displays a failed sync status or error message instead of `sync ok`
- **AND** other views remain usable

#### Scenario: Reload failure is shown
- **WHEN** the TUI cannot reload items, sources, or status from runtime storage
- **THEN** the TUI displays a concise error message
- **AND** keeps the previous usable model state where practical

#### Scenario: Item action failure is shown
- **WHEN** the user triggers an item action such as open URL, open Markdown, toggle read, star, or archive and the action fails
- **THEN** the TUI displays a concise action error message
- **AND** does not update the visible item state as though the action succeeded

### Requirement: TUI sync commands preserve cancellation context
The TUI SHALL pass its active command context to background sync work instead of using a detached background context.

#### Scenario: TUI exits during background sync
- **WHEN** the TUI is shutting down while a background sync command is running
- **THEN** the sync work receives context cancellation where supported
- **AND** the TUI does not report the cancelled sync as successful after shutdown begins
