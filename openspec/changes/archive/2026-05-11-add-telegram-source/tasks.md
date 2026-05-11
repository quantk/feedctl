## 1. Config and Adapter Routing

- [x] 1.1 Add failing config tests showing `telegram` source files are accepted, unsupported source types are still rejected, and optional `max_items` is loaded.
- [x] 1.2 Add `telegram` source type and optional Telegram item limit support to config loading/validation.
- [x] 1.3 Add failing source adapter routing tests showing RSS sources use the RSS adapter, Telegram sources use the Telegram adapter, and unknown types fail cleanly.
- [x] 1.4 Implement a type-aware source adapter router while preserving the existing `source.Adapter` test seam.
- [x] 1.5 Update `sync.NewRunner` and source testing paths to use the adapter router by default.

## 2. Telegram Source CLI and App API

- [x] 2.1 Add failing tests for Telegram channel input normalization from `@channel`, bare channel names, `https://t.me/channel`, and `https://t.me/s/channel`.
- [x] 2.2 Implement Telegram channel input normalization and canonical public web URL generation.
- [x] 2.3 Add failing app tests for `AddTelegram` dry-run, successful source creation, generated id, explicit id, tags, canonical URL, and existing-source conflict.
- [x] 2.4 Implement `app.AddTelegram` using the Telegram adapter's test/metadata path.
- [x] 2.5 Add failing CLI tests for `feedctl add telegram` text and JSON output.
- [x] 2.6 Implement the `feedctl add telegram` command with `--id`, `--name`, `--tags`, `--max-items`, `--dry-run`, and `--yes` flags.
- [x] 2.7 Add failing tests that `feedctl sources test` works for configured Telegram sources through the adapter router.
- [x] 2.8 Implement Telegram-aware source testing without saving items.

## 3. Telegram Public Web Adapter

- [x] 3.1 Add HTML fixture tests for Telegram metadata extraction from public channel pages.
- [x] 3.2 Add HTML fixture tests for post extraction: `data-post`, datetime, message text, canonical post URL, author/channel name, and source tags.
- [x] 3.3 Add fixture tests for Markdown conversion of Telegram rich text including bold, italic, links, blockquotes, line breaks, emoji, and lists.
- [x] 3.4 Add fixture tests for media-only or low-text posts ensuring generated titles and useful body placeholders/post links.
- [x] 3.5 Implement the Telegram adapter fetch/test flow against public web pages.
- [x] 3.6 Implement robust post parsing and Markdown normalization while excluding Telegram page chrome.
- [x] 3.7 Add pagination tests showing `before` links are followed until `max_items` or no older page.
- [x] 3.8 Implement bounded Telegram pagination with a conservative default item limit.
- [x] 3.9 Add error-path tests for HTTP errors, empty/non-public pages, and malformed post HTML.
- [x] 3.10 Implement structured Telegram adapter errors that surface as source-level sync/test failures.

## 4. Sync and Persistence Behavior

- [x] 4.1 Add sync tests showing Telegram posts create normal item records and Markdown files using existing storage tables.
- [x] 4.2 Add sync tests showing repeated Telegram syncs do not duplicate posts with the same `channel/message_id` identity.
- [x] 4.3 Add sync tests showing edited Telegram post content within the fetched window creates a normal item version.
- [x] 4.4 Add sync tests showing a failed Telegram source does not prevent other sources from syncing.
- [x] 4.5 Make any minimal sync integration changes needed for Telegram sources to pass through the existing item pipeline.

## 5. Documentation and Validation

- [x] 5.1 Update README with Telegram source examples, public-only limitation, `--max-items`, and no-MTProto/no-private-channel scope.
- [x] 5.2 Run `gofmt` on changed Go files.
- [x] 5.3 Run targeted tests for config, source, app, CLI, and sync packages.
- [x] 5.4 Run `go test ./...`.
- [x] 5.5 Run `openspec validate add-telegram-source --strict` and fix any artifact issues.
