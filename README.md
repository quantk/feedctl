# feedctl

`feedctl` is a local-first terminal content inbox. The MVP supports RSS feeds and public Telegram channels, stores declarative config under `~/.config/feedctl`, stores runtime state under `~/.feedctl/feedctl.db`, and saves fetched items as Markdown under `~/.feedctl/content`.

## Build

```bash
make build
./feedctl --help
```

## Configuration

Main config path:

```text
~/.config/feedctl/config.toml
```

Source files live one source per file:

```text
~/.config/feedctl/sources.d/<source-id>.toml
```

Example RSS source file:

```toml
id = "example"
type = "rss"
name = "Example"
url = "https://example.com/feed.xml"
enabled = true
interval = "10m"
tags = ["tech", "blog"]
```

Example public Telegram source file:

```toml
id = "tg-llm-under-hood"
type = "telegram"
name = "LLM под капотом"
url = "https://t.me/s/llm_under_hood"
enabled = true
interval = "10m"
tags = ["telegram", "llm"]
max_items = 50
```

Telegram support uses the public web view (`https://t.me/s/<channel>`). It does not use MTProto, does not require login/session secrets, does not support private channels, and does not download media files locally.

Runtime fields such as `last_sync_at`, `last_error`, `etag`, read state, hashes, versions, and disk usage belong in SQLite, not TOML config.

## Common commands

```bash
feedctl                       # open TUI
feedctl tui                   # open TUI explicitly
feedctl config path           # show effective paths
feedctl config validate       # validate config/source files
feedctl add rss URL           # create RSS source config
feedctl add rss URL --dry-run --json
feedctl add telegram @channel # create public Telegram source config
feedctl add telegram https://t.me/channel --max-items 50 --dry-run --json
feedctl sources list
feedctl sources show ID
feedctl sources test ID
feedctl sources enable ID --yes
feedctl sources disable ID --yes
feedctl sources remove ID --yes
feedctl sync
feedctl sync --source ID --json
feedctl items list
feedctl items list --unread
feedctl items list --removed-sources
feedctl items open ITEM_ID
feedctl items markdown ITEM_ID
feedctl storage
feedctl storage reconcile
feedctl status
```

Important non-TUI commands support `--json`. Mutating commands support `--dry-run` where practical and accept `--yes` for non-interactive use.

## TUI keys

Navigation: `j/k`, arrows, `h/l`, `g/G`, `Ctrl+d/u`, `Ctrl+f/b`.

Sections: `1` Inbox, `2` Unread, `3` Starred, `4` Sources, `5` Removed Sources, `6` All Items, `Tab`, `Shift+Tab`.

Search/filter: `/` search with `n/N` next/previous, `f` live-filter items, `F` clear filter, `A` toggle removed-source items.

Item actions: `Enter/l` open, `Space` read/unread, `u` unread, `s` star, `a` archive from inbox, `o` open URL, `e` edit Markdown, `m` show/hide Markdown frontmatter.

Reader/preview renders Markdown by default and hides YAML frontmatter unless `m` is enabled.

Sync: `r` refresh, `R` sync all.

Other: `?` help, `Esc` back/close, `q` quit.
