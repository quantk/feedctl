## ADDED Requirements

### Requirement: Public Telegram channel fetching
The system SHALL fetch public Telegram channel posts from Telegram public web pages without requiring MTProto credentials, Bot API credentials, user login, or stored account sessions.

#### Scenario: Public channel page is fetched
- **WHEN** sync processes an active Telegram source whose URL points to `https://t.me/s/example`
- **THEN** the Telegram adapter requests the public web page for that channel
- **AND** no Telegram account credentials or session files are required

#### Scenario: Public channel cannot be fetched
- **WHEN** Telegram public web returns an HTTP error, unavailable page, or non-parseable response for a Telegram source
- **THEN** the source sync fails for that source with a structured fetch or parse error
- **AND** other sources remain isolated from the failure

### Requirement: Telegram post normalization
The Telegram adapter SHALL normalize public Telegram posts into feedctl item candidates compatible with the existing sync and Markdown pipeline.

#### Scenario: Telegram post fields are extracted
- **WHEN** a public Telegram page contains a post with `data-post`, timestamp, message text, and channel metadata
- **THEN** the adapter emits an item with source type `telegram`, a title, canonical post URL, publish time, Markdown body, author or channel name when available, and source tags

#### Scenario: Telegram post without text is still represented
- **WHEN** a public Telegram post contains media or attachments but no textual message body
- **THEN** the adapter emits an item with a generated title and a body that preserves the canonical Telegram post URL or an equivalent readable placeholder

### Requirement: Stable Telegram item identity
Telegram items SHALL use stable source-specific identities derived from channel username and Telegram message id.

#### Scenario: Message id identity is available
- **WHEN** a Telegram post has `data-post="example/123"`
- **THEN** the item source identity is `example/123`
- **AND** the identity kind recorded by sync is `guid`

#### Scenario: Same Telegram post is fetched again
- **WHEN** sync fetches the same Telegram post on a later run
- **THEN** the existing item is updated or marked unchanged rather than duplicated

### Requirement: Telegram Markdown body rendering
The Telegram adapter SHALL convert supported Telegram message HTML into readable Markdown before items are saved.

#### Scenario: Rich Telegram text is converted
- **WHEN** a Telegram post contains bold text, italic text, links, blockquotes, line breaks, emoji, or lists in public web HTML
- **THEN** the saved item body contains readable Markdown text preserving the meaningful content and links

#### Scenario: Telegram web chrome is excluded
- **WHEN** a Telegram public page is parsed
- **THEN** the saved item body does not include unrelated page chrome, scripts, styles, navigation text, or neighboring post text

### Requirement: Bounded Telegram pagination
The Telegram adapter SHALL support bounded backfill through Telegram public web pagination and SHALL NOT crawl channel history without a limit.

#### Scenario: Pagination follows older pages within limit
- **WHEN** a Telegram public page contains an older-page `before` link and the configured item limit has not been reached
- **THEN** the adapter fetches older public pages until the item limit is reached or no older page exists

#### Scenario: Pagination stops at item limit
- **WHEN** the adapter has collected the configured maximum number of Telegram posts for a source
- **THEN** it stops requesting older pages even if more `before` links are present

#### Scenario: Default item limit is used
- **WHEN** a Telegram source does not configure an explicit item limit
- **THEN** sync uses a conservative default limit for Telegram posts

### Requirement: Telegram media handling
The system SHALL preserve Telegram post readability without downloading Telegram media files into feedctl-managed storage.

#### Scenario: Post contains public media references
- **WHEN** a Telegram post contains images, videos, or other public media references
- **THEN** the Markdown item may include external references or textual placeholders for that media
- **AND** the canonical Telegram post URL remains available on the item

#### Scenario: Media is not downloaded locally
- **WHEN** sync processes a Telegram post with media
- **THEN** feedctl does not save the media binary into the content or versions directories as part of this change
