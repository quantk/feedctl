## Why

Feed items currently show only source and read/starred state, so high-value articles from sources such as Habr are not visually distinguishable by community signal. Habr exposes article score and activity outside RSS, and feedctl should be able to enrich items with those metrics without hard-coding Habr-specific behavior into the TUI or RSS parser.

## What Changes

- Add an item metrics capability that stores optional, provider-sourced item metrics such as score, comments, votes, favorites, and reading count.
- Add a provider architecture for item metrics so Habr is the first implementation and future sources can add their own providers without changing item rendering or core sync logic.
- Enrich Habr RSS items by deriving the Habr article id from item URLs and fetching statistics from Habr's article JSON endpoint.
- Persist metrics separately from stable content so metric changes do not rewrite Markdown or create item versions.
- Display available score and comment metrics in the TUI item list and selected item details while hiding missing metrics.
- Treat metrics fetch failures as non-fatal warnings: item sync must continue even when metrics are unavailable.

## Capabilities

### New Capabilities
- `item-metrics`: Optional runtime metrics for saved items, including provider discovery, Habr statistics enrichment, persistence, and update behavior.

### Modified Capabilities
- `tui-inbox`: Item rows and item details display available item metrics such as score and comments without requiring metrics for every item.

## Impact

- Affected code: `internal/source`, new metrics/provider package, `internal/sync`, `internal/store`, `internal/tui`, and focused tests.
- Storage: SQLite migration for item metrics data, preferably separated from item content/version records.
- Networking: Habr metrics provider performs additional HTTP requests for recognized Habr article URLs during sync.
- UX: TUI item rows gain compact metric indicators when metrics are present.
- Backward compatibility: Existing databases and items without metrics continue to work; RSS parsing and Markdown content remain compatible.
