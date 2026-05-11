## 1. Metrics Provider Model

- [x] 1.1 Add tests for Habr article id extraction covering normal article URLs, company article URLs, RSS query parameters, and non-Habr URLs.
- [x] 1.2 Add tests for Habr statistics JSON mapping, including known zero score/count values.
- [x] 1.3 Implement `internal/metrics` generic types, provider interface, provider registry/enricher, and Habr provider URL matching.
- [x] 1.4 Implement Habr provider HTTP fetching with injectable base URL/client for tests and map statistics into normalized optional metrics.

## 2. Runtime Storage

- [x] 2.1 Add storage tests for `item_metrics` migration, upsert, lookup/list population, missing metrics, and zero-value preservation.
- [x] 2.2 Add the idempotent `item_metrics` SQLite table migration and store-level metrics types.
- [x] 2.3 Implement metrics upsert and scanning so `GetItem`, `FindItemBySourceIdentity`, and `ListItems` expose optional item metrics.

## 3. Sync Integration

- [x] 3.1 Add sync tests proving metrics are stored for new Habr items without adding metrics to Markdown frontmatter.
- [x] 3.2 Add sync tests proving changed metrics for unchanged content update runtime metrics without rewriting Markdown or incrementing item version.
- [x] 3.3 Add sync tests proving metrics provider failures do not fail source sync for new or existing items.
- [x] 3.4 Wire the metrics enricher into `sync.Runner` after normal item persistence and before final source status reporting.
- [x] 3.5 Ensure non-Habr items continue syncing without metrics provider calls or user-visible errors.

## 4. TUI Display

- [x] 4.1 Add TUI rendering tests for compact score/comments in item rows, hidden missing metrics, and visible known zero score.
- [x] 4.2 Add TUI preview/details tests for expanded metrics on the selected item.
- [x] 4.3 Implement provider-neutral item row metric formatting that truncates gracefully with narrow widths.
- [x] 4.4 Implement selected item preview/details metric rendering using normalized stored metrics.

## 5. Verification

- [x] 5.1 Run focused package tests after each Red/Green step.
- [x] 5.2 Run `gofmt -w` on changed Go files.
- [x] 5.3 Run `go test ./...` and document any external failures.
