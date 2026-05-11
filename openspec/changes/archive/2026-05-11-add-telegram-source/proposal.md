## Why

Telegram channels are a common source of long-form technical and news content, but today feedctl can only ingest RSS-compatible feeds. Adding public Telegram channels lets users keep Telegram reading in the same local-first inbox without requiring MTProto login, API credentials, or account sessions.

## What Changes

- Add native `telegram` source support for public channels available through Telegram's public web view (`https://t.me/s/<channel>`).
- Add CLI support for creating Telegram sources from `@channel`, `https://t.me/<channel>`, or `https://t.me/s/<channel>` inputs.
- Normalize Telegram posts into feedctl items with stable identities, canonical post URLs, publish timestamps, readable Markdown bodies, and source tags.
- Support bounded pagination/backfill through Telegram public web `?before=<message_id>` pages without using MTProto.
- Keep Telegram authentication, private channels, MTProto sessions, local media downloads, comments syncing, and reaction metrics out of the initial scope.

## Capabilities

### New Capabilities
- `telegram-source`: Public Telegram channel discovery, fetching, normalization, pagination, and sync behavior.

### Modified Capabilities
- `config-source-management`: Source validation and CLI source management shall accept `telegram` sources in addition to RSS sources.

## Impact

- Affected code: `internal/config`, `internal/source`, `internal/sync`, `internal/app`, `internal/cli`, tests, README, and OpenSpec specs.
- Adds an HTML parsing path for Telegram public web pages and likely reuses the existing HTML-to-Markdown conversion dependency.
- No database schema changes are expected for basic item ingestion because Telegram posts can use existing item/source tables.
- No secret storage, account login, MTProto dependency, or background daemon is introduced by this change.
