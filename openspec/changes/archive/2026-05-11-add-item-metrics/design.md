## Context

feedctl currently normalizes RSS/Atom/JSON Feed entries into `source.Item`, writes stable Markdown content, and stores runtime item state in SQLite. The TUI renders `store.Item` rows using read/star/source state only. Habr RSS feeds do not include article score or comments, but the article page/API exposes statistics through `https://habr.com/kek/v2/articles/<article-id>/`.

The feature crosses source normalization, sync, storage, and TUI rendering, so the design needs to keep Habr-specific network behavior isolated from RSS parsing and from generic item rendering.

## Goals / Non-Goals

**Goals:**

- Introduce a generic item metrics model with optional score/comment/activity fields.
- Implement Habr as the first metrics provider.
- Make future providers additive: they should plug into a registry/enricher without changing TUI row rendering semantics.
- Persist metrics as runtime metadata separate from Markdown content and content hashes.
- Keep sync resilient when metrics cannot be fetched.
- Show compact metrics in the TUI only when they are available.

**Non-Goals:**

- Ranking or sorting items by score.
- A manual metrics refresh command separate from sync.
- Authentication against provider APIs.
- Embedding volatile metrics into Markdown content/frontmatter.
- Full provider configuration UI or per-source metrics toggles in the first iteration.

## Decisions

### 1. Add a dedicated `internal/metrics` package

Create a package responsible for provider matching, fetching, and normalized metric values.

Sketch:

```go
type ItemMetrics struct {
    Provider       string
    Score          *int
    CommentsCount  *int
    VotesCount     *int
    FavoritesCount *int
    ReadingCount   *int
    FetchedAt      string
}

type Candidate struct {
    SourceID     string
    SourceType   string
    Title        string
    URL          string
    CanonicalURL string
}

type Provider interface {
    Name() string
    Match(Candidate) bool
    Fetch(ctx context.Context, Candidate) (ItemMetrics, error)
}

type Enricher struct {
    Providers []Provider
}
```

Rationale: the RSS adapter should only parse feed data; it should not learn about Habr's separate article API. The TUI should render normalized fields rather than provider-specific JSON.

Alternative considered: add Habr fields directly to `source.Item`. Rejected because it couples RSS parsing to one provider and makes future providers harder to add.

### 2. Use pointer fields for optional metrics

Metrics like score and comments use `*int` rather than `int`.

Rationale: `nil` means unknown/unavailable, while `0` is a valid score or count. This is important for Habr articles with score `0`.

Alternative considered: use sentinel values such as `-1`. Rejected because it leaks display concerns into storage and makes JSON/API semantics less clear.

### 3. Store metrics separately from item content

Add an idempotent SQLite migration/table, for example:

```sql
CREATE TABLE IF NOT EXISTS item_metrics (
    item_id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    metrics_json TEXT NOT NULL,
    FOREIGN KEY(item_id) REFERENCES items(id)
);
```

`store.Item` can expose an optional `Metrics *ItemMetrics`, populated by `ListItems`, `GetItem`, and `FindItemBySourceIdentity` when available.

Rationale: metrics are volatile runtime state. Updating score/comments must not rewrite Markdown or increment item versions.

Alternative considered: add columns to `items`. Rejected for the first iteration because metrics fields are provider-shaped and may grow; a JSON-backed side table is more flexible and keeps existing item storage stable.

### 4. Fetch metrics during sync after item persistence

For each processed item, sync computes the stable item id and performs normal new/unchanged/updated content handling first. After the item row exists, the metrics enricher tries to fetch metrics for the item candidate and upserts `item_metrics` if available.

Rationale: item sync remains authoritative and should not be blocked by metric fetch problems. Existing/unchanged content can still receive updated metrics without changing item version.

Alternative considered: enrich before item processing. Rejected because a metrics failure could complicate item creation paths and because metrics do not affect identity or content hashing.

### 5. Metrics failures are warnings, not sync failures

Provider errors are collected as warnings or silently ignored if no warning surface exists yet. They must not append to fatal sync errors and must not set source status to `failed`.

Rationale: ratings are useful but optional. A temporary Habr API failure should not prevent article ingestion.

Alternative considered: fail source sync if metrics fail. Rejected because it would make optional enrichment reduce reliability.

### 6. Habr provider recognizes article URLs and fetches statistics JSON

The Habr provider matches URLs whose host is `habr.com` or a Habr subdomain and whose path contains `articles/<digits>`. It supports URLs with language/company prefixes and query strings, for example:

- `https://habr.com/ru/articles/1033808/`
- `https://habr.com/ru/companies/kodik/articles/1032884/?utm_source=...`

It fetches:

```text
https://habr.com/kek/v2/articles/<id>/
```

and maps `statistics.score`, `statistics.commentsCount`, `statistics.votesCount`, `statistics.favoritesCount`, and `statistics.readingCount` into `ItemMetrics`.

Rationale: RSS does not contain these fields, but Habr exposes them per article. URL matching keeps the provider independent of source ids.

Alternative considered: scrape HTML. Rejected because the JSON endpoint is simpler and more stable for tests.

### 7. TUI displays normalized compact metrics

Item rows display compact metrics before the source suffix when available, e.g. `+12 · 4c [habr-ai]`. Missing metrics render nothing. The preview/details pane can show expanded metrics such as score, comments, votes, and reads.

Rationale: rows remain readable for sources without metrics and future providers automatically benefit from the normalized metric model.

Alternative considered: Habr-specific labels in TUI. Rejected because rendering should be provider-neutral.

## Risks / Trade-offs

- Habr endpoint may change or rate-limit requests → keep failures non-fatal and isolate parsing in `HabrProvider` tests.
- Additional per-item network requests can slow sync → use the existing sync context and avoid retries in the first iteration; batching/caching can be added later.
- JSON side table is less queryable than columns → acceptable for display-first metrics; add indexed columns later if sorting/filtering by score becomes a requirement.
- Existing migration system is mostly idempotent DDL → implement the new table with `CREATE TABLE IF NOT EXISTS` and tests that opening an existing database preserves items.
- Terminal row width is limited → render only compact score/comments in rows and hide metrics first when width is constrained.

## Migration Plan

1. Add the `item_metrics` table through the existing idempotent migration path.
2. Existing item rows have no metrics until the next sync recognizes and enriches them.
3. If rollback is needed, older code should ignore the extra table; no existing item columns are changed.

## Open Questions

- Should sync results expose metrics warnings in CLI JSON immediately, or is non-fatal logging/message handling enough for the first iteration?
- Should TUI preview show all normalized metrics, or only the same compact score/comments used in rows?
