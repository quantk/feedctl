## Context

feedctl currently has a source-neutral persistence pipeline after fetching: adapters return `source.Feed` / `source.Item`, sync computes identity and content hashes, Markdown is written locally, and SQLite stores runtime state. The fetch layer is RSS-only today: config validation accepts only `rss`, `sync.NewRunner` installs a single RSS adapter, and source testing/add commands are RSS-specific.

Public Telegram channels expose recent posts through `https://t.me/s/<channel>` without user login. The pages contain stable post ids (`data-post="channel/message_id"`), publish timestamps, HTML message bodies, post URLs, view counts, reactions, media references, and pagination links (`?before=<message_id>`). This is enough to ingest public channel posts as normal feedctl items, while avoiding MTProto account/session complexity.

## Goals / Non-Goals

**Goals:**

- Add `telegram` as a supported source type for public Telegram channels.
- Let users add channels from `@channel`, `https://t.me/<channel>`, or `https://t.me/s/<channel>` inputs.
- Fetch bounded recent history through public Telegram web pages and normalize posts into existing `source.Item` records.
- Preserve the existing local-first sync, Markdown, versioning, read/star/archive, and TUI behavior for Telegram items.
- Keep the implementation testable with local HTTP fixtures instead of relying on live Telegram in unit tests.

**Non-Goals:**

- MTProto login, Bot API support, private channels, or user account sessions.
- Downloading media files into local feedctl storage.
- Syncing comments/replies or discussion threads.
- Storing Telegram views/reactions as item metrics in this change.
- Real-time update subscriptions or daemon behavior.

## Decisions

### Use public Telegram web pages for the first backend

The first implementation SHALL fetch `https://t.me/s/<channel>` style pages and parse their static HTML.

Alternatives considered:

- **MTProto**: more complete and supports private channels, but requires `api_id`, `api_hash`, phone login, 2FA handling, secure session storage, flood-wait handling, and account lifecycle UX. This is too large for the first Telegram source change.
- **RSS bridge/RSSHub**: easiest to integrate but makes Telegram support depend on a third-party bridge and does not feel native.
- **Bot API**: official but requires a bot to be added to channels and is a poor fit for arbitrary channels a user wants to read.

### Represent Telegram posts with existing item storage

A Telegram post SHALL map to `source.Item`:

- `GUID`: `<channel>/<message_id>` from `data-post`
- `URL` and `CanonicalURL`: `https://t.me/<channel>/<message_id>`
- `PublishedAt`: the post datetime when available
- `Author`: channel title or username when available
- `Title`: first meaningful text line, shortened if necessary, or a generated fallback such as `Telegram post <message_id>`
- `Body`: Markdown converted from the message HTML, plus external links to media/post when useful
- `Tags`: source tags

This avoids database schema changes and lets existing content hashing/versioning detect edited posts that remain inside the fetched window.

### Add an adapter router rather than special-casing sync

The sync runner should not contain Telegram-specific branching. Instead, source package should expose a type-aware adapter/router that implements the existing `source.Adapter` interface and dispatches to RSS or Telegram adapters based on `config.Source.Type`.

This keeps `sync.Runner` generic and preserves the existing testing seam: tests can still inject a custom adapter implementing `source.Adapter`.

### Normalize channel input at add time

`feedctl add telegram` SHALL accept:

- `@channel`
- `channel`
- `https://t.me/channel`
- `https://t.me/s/channel`

The stored source URL SHOULD be canonicalized to the public web URL (`https://t.me/s/<channel>`) so sync does not need to infer URL shapes repeatedly. Item URLs SHALL use the canonical user-facing post URL (`https://t.me/<channel>/<message_id>`).

### Bound pagination with an item limit

Telegram public web pages can be paged backward via `?before=<message_id>`. The adapter SHALL follow pagination only until a bounded maximum number of posts is collected or no older page exists.

The source config SHOULD support an optional `max_items` value for Telegram sources. If omitted, the adapter uses a conservative default. This behaves like an RSS feed size: every sync re-fetches the recent window, creates new posts, and catches edits within the window without introducing a Telegram-specific cursor table.

### Treat media as external references

The adapter SHALL NOT download Telegram media into local storage. For posts with media, the Markdown body may include the Telegram post URL and, when safely available, external image/video links or textual placeholders. The item remains useful even when CDN media links expire because the canonical post URL is retained.

## Risks / Trade-offs

- **Telegram changes public HTML structure** → Keep parsing isolated in `TelegramAdapter`, cover selectors with fixture tests, and return source-level parse errors instead of panics.
- **Public web access is rate-limited or blocked** → Use bounded pagination, normal HTTP timeouts, source-level failure isolation, and avoid aggressive defaults.
- **Private channels are not supported** → Document the limitation and leave room for a future MTProto backend under the same `telegram` source type.
- **Edits outside the recent window are missed** → Re-fetch a bounded recent window on every sync and document that full historical edit tracking requires a future stateful backend.
- **Media CDN URLs may expire** → Preserve canonical Telegram post URLs and avoid local media download in this change.
- **HTML-to-Markdown conversion may produce noisy output** → Normalize whitespace and test representative post fixtures with bold, links, lists, blockquotes, emoji, and media.

## Migration Plan

No migration is required for existing users. Existing RSS source files and runtime records remain valid. After implementation, config validation accepts both `rss` and `telegram`; unsupported source types still fail validation.

Rollback is straightforward: removing Telegram source files stops future Telegram syncs while preserving already saved Markdown and runtime item records as removed-source content, following existing source lifecycle behavior.

## Open Questions

- What default `max_items` should be used for Telegram sources? A conservative starting point such as 50 keeps sync useful without excessive pagination.
- Should `max_items` be exposed in the initial CLI flags or only accepted in TOML for advanced users?
