# item-metrics Specification

## Purpose
The item metrics capability stores optional provider-sourced runtime metrics for saved feed items and exposes them for sync and display without coupling metric providers to RSS parsing or TUI rendering.
## Requirements
### Requirement: Optional item metrics
The system SHALL support optional runtime metrics for saved items, including provider name, fetched time, score, comments count, votes count, favorites count, and reading count.

#### Scenario: Metrics are stored for an item
- **WHEN** sync obtains metrics for a saved item
- **THEN** the system stores the metrics in runtime storage associated with that item id
- **AND** later item lookups and item lists can include those metrics

#### Scenario: Missing metrics remain unknown
- **WHEN** no metrics provider matches an item or a provider returns no value for a metric field
- **THEN** the item remains usable without that metric
- **AND** unknown metric fields are not treated as numeric zero

#### Scenario: Zero metrics are preserved
- **WHEN** a provider reports a score or count value of `0`
- **THEN** the system stores and exposes that value as a known metric value of zero

### Requirement: Provider-based metrics enrichment
The system SHALL obtain item metrics through registered metrics providers selected by item URL or canonical URL rather than through TUI-specific or RSS-parser-specific logic.

#### Scenario: Item matches a registered provider
- **WHEN** sync processes an item whose URL or canonical URL is recognized by a registered metrics provider
- **THEN** the system asks that provider for metrics for the item

#### Scenario: Item has no metrics provider
- **WHEN** sync processes an item whose URL and canonical URL are not recognized by any registered metrics provider
- **THEN** sync saves or updates the item without attempting provider-specific metrics fetching

#### Scenario: Future provider can be added without TUI changes
- **WHEN** a new provider returns normalized item metrics using the generic metrics model
- **THEN** item storage and TUI display can consume those metrics without adding provider-specific rendering rules

### Requirement: Habr article metrics provider
The system SHALL provide a Habr metrics provider that extracts article ids from Habr article URLs and maps Habr article statistics to normalized item metrics.

#### Scenario: Habr article URL is recognized
- **WHEN** an item URL is `https://habr.com/ru/articles/1033808/`
- **THEN** the Habr provider recognizes article id `1033808`

#### Scenario: Habr company article URL is recognized
- **WHEN** an item URL is `https://habr.com/ru/companies/kodik/articles/1032884/?utm_campaign=1032884&utm_source=habrahabr&utm_medium=rss`
- **THEN** the Habr provider recognizes article id `1032884`

#### Scenario: Habr statistics are mapped
- **WHEN** Habr returns article JSON containing `statistics.score`, `statistics.commentsCount`, `statistics.votesCount`, `statistics.favoritesCount`, and `statistics.readingCount`
- **THEN** the Habr provider returns normalized metrics with provider `habr` and the corresponding score, comments, votes, favorites, and reading count values

#### Scenario: Non-Habr URL is ignored
- **WHEN** an item URL does not belong to Habr or does not contain an article id
- **THEN** the Habr provider does not match the item

### Requirement: Metrics sync resilience
The system SHALL treat metrics fetching as optional enrichment that cannot prevent item sync from succeeding.

#### Scenario: Metrics provider fails for a new item
- **WHEN** sync saves a new item and the matched metrics provider returns an error
- **THEN** the item record and Markdown file remain saved successfully
- **AND** the source sync is not marked failed solely because metrics were unavailable

#### Scenario: Metrics provider fails for an existing item
- **WHEN** sync sees an existing item and the matched metrics provider returns an error
- **THEN** the item last-seen/content handling proceeds according to normal sync behavior
- **AND** the source sync is not marked failed solely because metrics were unavailable

### Requirement: Metrics updates do not change content versions
Metrics SHALL be runtime metadata and MUST NOT affect stable content hashes, Markdown files, or item version numbers.

#### Scenario: Only metrics change for unchanged content
- **WHEN** sync processes an existing item whose title, URLs, and body content are unchanged but whose score or comments count has changed
- **THEN** the system updates stored metrics
- **AND** the item content hash remains unchanged
- **AND** the item version number is not incremented
- **AND** the current Markdown file is not rewritten solely because metrics changed

#### Scenario: New item frontmatter excludes volatile metrics
- **WHEN** sync writes a Markdown file for a new item with available metrics
- **THEN** the generated frontmatter contains the stable item fields
- **AND** it does not include score, comments count, votes, favorites, reading count, or provider metrics fetched time

### Requirement: Bounded metrics enrichment
Metrics enrichment SHALL respect caller cancellation and bounded network execution so optional provider calls cannot indefinitely delay item sync.

#### Scenario: Metrics provider exceeds timeout or cancellation
- **WHEN** a metrics provider call does not complete before its context is cancelled or its timeout is reached
- **THEN** metrics enrichment for that item stops
- **AND** the item sync remains successful if the item content and runtime metadata were saved successfully

#### Scenario: Metrics timeout does not mark source failed
- **WHEN** a matched metrics provider times out while processing an item
- **THEN** the source sync is not marked failed solely because metrics were unavailable
- **AND** the item remains available without metrics

#### Scenario: Metrics provider receives item sync context
- **WHEN** sync asks a metrics provider to fetch metrics for an item
- **THEN** the provider receives a context derived from the source sync context

