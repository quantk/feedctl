## ADDED Requirements

### Requirement: Item metrics display
The TUI SHALL display available item metrics in item list rows and selected item details without requiring metrics to exist for every item.

#### Scenario: Item row shows score and comments
- **WHEN** an item has known score and comments count metrics
- **THEN** the TUI item row shows a compact score indicator and a compact comments indicator for that item

#### Scenario: Missing metrics are hidden
- **WHEN** an item has no stored metrics
- **THEN** the TUI item row does not reserve visible placeholder text for score or comments
- **AND** the item remains selectable and readable normally

#### Scenario: Zero score is displayed as known value
- **WHEN** an item has a known score value of `0`
- **THEN** the TUI displays the score as a known zero value rather than hiding it as missing

#### Scenario: Selected item details include metrics
- **WHEN** the selected item has stored metrics
- **THEN** the preview or details pane includes the available metrics such as score, comments, votes, and reading count
