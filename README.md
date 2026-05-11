# feedctl

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![Status](https://img.shields.io/badge/status-local--first%20MVP-88C0D0)
![Storage](https://img.shields.io/badge/storage-SQLite%20%2B%20Markdown-5E81AC)

**feedctl** — локальный терминальный inbox для статей, постов и лент. Он забирает контент из RSS и публичных Telegram-каналов, хранит состояние локально в SQLite, а сами материалы сохраняет как обычные Markdown-файлы.

Главная идея: **контент принадлежит вам**. Конфигурация декларативная, runtime-состояние отдельно, Markdown можно читать, искать, версионировать и открывать любыми внешними инструментами.

---

## Возможности

- 🗂 **Local-first inbox** — база и Markdown живут локально, без серверной части.
- 📰 **RSS-источники** — классические фиды, статьи, блоги.
- ✈️ **Публичные Telegram-каналы**
- 📝 **Markdown-архив** — каждый материал сохраняется в `~/.feedctl/content`.
- 🧾 **YAML frontmatter** — метаданные в Markdown, в TUI скрываются по умолчанию.
- 🔁 **Версии материалов** — изменение контента создаёт новую версию вместо потери истории.
- ⭐ **Inbox workflow** — read/unread, starred, archive, removed-source items.
- 🔎 **Поиск и фильтр в TUI**.
- 🎨 **Nord-style TUI** — полноэкранный терминальный интерфейс с Markdown preview/reader.
- 🤖 **CLI-friendly output** — важные команды поддерживают `--json`, мутации — `--dry-run` и `--yes`.

---

## Быстрый старт

```bash
# 1. Собрать бинарь
make build

# 2. Посмотреть эффективные пути
./feedctl config path

# 3. Добавить RSS-источник
./feedctl add rss https://example.com/feed.xml \
  --id example \
  --name Example \
  --tags tech,blog \
  --yes

# 4. Добавить публичный Telegram-канал
./feedctl add telegram @llm_under_hood \
  --id tg-llm-under-hood \
  --tags telegram,llm \
  --max-items 50 \
  --yes

# 5. Синхронизировать источники
./feedctl sync

# 6. Открыть TUI
./feedctl
```

Для безопасной проверки без записи используйте `--dry-run`:

```bash
./feedctl add telegram https://t.me/llm_under_hood \
  --max-items 50 \
  --dry-run \
  --json
```

---

## Установка и разработка

### Требования

- Go `1.25+`
- SQLite
- `make` для удобных команд

Опционально доступен Nix dev shell:

```bash
nix develop
```

### Сборка

```bash
make build
./feedctl --help
```

### Запуск тестов

```bash
make test
# или напрямую
CGO_ENABLED=0 go test ./...
```

---

## Где лежат данные

По умолчанию `feedctl` разделяет декларативную конфигурацию и runtime-состояние:

| Что | Путь |
| --- | --- |
| Основной config | `~/.config/feedctl/config.toml` |
| Source-файлы | `~/.config/feedctl/sources.d/<source-id>.toml` |
| Runtime root | `~/.feedctl` |
| SQLite database | `~/.feedctl/feedctl.db` |
| Markdown content | `~/.feedctl/content` |
| Старые версии Markdown | `~/.feedctl/versions` |
| Временные файлы | `~/.feedctl/tmp` |
| Логи | `~/.feedctl/logs` |

Пути можно переопределить переменными окружения:

```bash
FEEDCTL_CONFIG_DIR=/path/to/config
FEEDCTL_CONFIG_FILE=/path/to/config.toml
FEEDCTL_DATA_ROOT=/path/to/data
```

Редактор и браузер берутся из окружения:

```bash
EDITOR=nvim
BROWSER=xdg-open
```

---

## Конфигурация

### Main config

`~/.config/feedctl/config.toml` может выглядеть так:

```toml
[data]
root = "~/.feedctl"
database = "~/.feedctl/feedctl.db"
content_dir = "~/.feedctl/content"
versions_dir = "~/.feedctl/versions"

[sources]
dir = "~/.config/feedctl/sources.d"

[sync]
default_interval = "5m"
concurrency = 4
sync_on_startup = true

[tui]
editor = "nvim"
browser = "xdg-open"
show_removed_sources = false

[markdown]
frontmatter = true
path_template = "{source_id}/{year}/{month}/{slug}.md"
```

Поддерживаемые токены `path_template`:

- `{source_id}`
- `{year}`
- `{month}`
- `{day}`
- `{slug}`
- `{item_id}`
- `{item_id_short}`

### Source-файлы

Каждый источник описывается отдельным TOML-файлом в `sources.d`.

#### RSS

```toml
id = "example"
type = "rss"
name = "Example"
url = "https://example.com/feed.xml"
enabled = true
interval = "10m"
tags = ["tech", "blog"]
```

#### Telegram

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

`source id` должен быть безопасен для файлов и CLI: начинаться с латинской буквы или цифры и содержать только `a-z`, `0-9`, `_`, `-`.

### Что нельзя хранить в config

TOML-конфиги — только декларативные. Runtime-поля хранятся в SQLite и не должны попадать в config:

- `last_sync_at`, `last_error`, `etag`, `last_modified`, `cursor`
- read/unread state
- hashes, versions
- item counts
- disk usage

Проверить конфигурацию:

```bash
./feedctl config validate
./feedctl config validate --json
```

---

## Источники контента

### RSS

RSS-источник добавляется командой:

```bash
./feedctl add rss https://example.com/feed.xml \
  --id example \
  --name Example \
  --tags tech,blog \
  --yes
```

Что происходит:

1. `feedctl` загружает feed metadata.
2. Генерирует или принимает `source id`.
3. Создаёт TOML-файл в `sources.d`.
4. При `sync` сохраняет новые материалы как Markdown.

### Telegram public web

Telegram-источник добавляется по username или URL:

```bash
./feedctl add telegram @channel --yes
./feedctl add telegram https://t.me/channel --max-items 50 --yes
./feedctl add telegram https://t.me/s/channel --dry-run --json
```

Поддерживаются публичные каналы, доступные через web-view:

```text
https://t.me/s/<channel>
```

Ограничения Telegram MVP:

- приватные каналы не поддерживаются;
- логин, phone auth, 2FA, MTProto и Bot API не используются;
- session-файлы и Telegram credentials не нужны;
- медиафайлы не скачиваются локально;
- Telegram HTML может измениться, поэтому parser intentionally scoped to public web pages.

Идентичность поста стабильна и строится как:

```text
<channel>/<message_id>
```

Например:

```text
llm_under_hood/831
```

---

## Markdown и версии

Каждый item сохраняется как Markdown. При включённом frontmatter файл содержит служебные поля:

```markdown
---
id: "..."
source_id: "tg-llm-under-hood"
source_name: "LLM под капотом"
source_type: "telegram"
title: "..."
url: "https://t.me/llm_under_hood/831"
canonical_url: "https://t.me/llm_under_hood/831"
published_at: "..."
fetched_at: "..."
content_hash: "sha256:..."
version: 1
tags: ["telegram", "llm"]
---

# Заголовок

Текст материала...
```

Если источник меняет уже сохранённый материал, `feedctl`:

1. считает новый content hash;
2. сохраняет предыдущую версию в `~/.feedctl/versions`;
3. обновляет текущий Markdown;
4. увеличивает номер версии.

Provider metrics, например Habr views/comments/bookmarks/votes, считаются runtime metadata и **не входят** в Markdown hash и versioning.

---

## TUI

Запуск:

```bash
./feedctl
# или явно
./feedctl tui
```

TUI открывается fullscreen, использует Nord-палитру и вертикальный marker `┃` для выбранной строки.

### Основные клавиши

| Группа | Клавиши |
| --- | --- |
| Навигация | `j/k`, arrows, `h/l`, `g/G`, `Ctrl+d/u`, `Ctrl+f/b` |
| Разделы | `1` Inbox, `2` Unread, `3` Starred, `4` Sources, `5` Removed Sources, `6` All Items |
| Переключение разделов | `Tab`, `Shift+Tab` |
| Поиск | `/`, затем `n/N` для next/previous |
| Live-filter | `f` начать фильтр, `F` очистить |
| Removed-source items | `A` показать/скрыть |
| Открыть reader | `Enter` или `l` |
| Read/unread | `Space`, `u` |
| Star | `s` |
| Archive | `a` |
| Открыть URL | `o` |
| Редактировать Markdown | `e` |
| Frontmatter в preview/reader | `m` показать/скрыть |
| Sync | `r` refresh текущего, `R` sync all |
| Help / выход | `?`, `Esc`, `q` |

Reader и preview рендерят Markdown красиво через terminal renderer. YAML frontmatter скрыт по умолчанию, чтобы не мешать чтению; включается клавишей `m`.

---

## CLI reference

### Общие команды

```bash
./feedctl                       # открыть TUI
./feedctl tui                   # открыть TUI явно
./feedctl status                # краткий статус inbox/storage/sync
./feedctl status --json
```

### Config

```bash
./feedctl config path           # эффективные пути
./feedctl config validate       # проверить config/source-файлы
./feedctl config format --yes   # отформатировать существующие TOML-файлы
```

### Sources

```bash
./feedctl sources list
./feedctl sources show ID
./feedctl sources test ID
./feedctl sources test ID --json
./feedctl sources enable ID --yes
./feedctl sources disable ID --yes
./feedctl sources remove ID --dry-run --json
./feedctl sources remove ID --yes
```

Удаление source config не удаляет уже сохранённые items и Markdown. Такие материалы можно видеть через removed-source views.

### Sync

```bash
./feedctl sync                  # синхронизировать все enabled sources
./feedctl sync --source ID      # синхронизировать один source
./feedctl sync --json
```

### Items

```bash
./feedctl items list
./feedctl items list --unread
./feedctl items list --removed-sources
./feedctl items list --json
./feedctl items open ITEM_ID
./feedctl items markdown ITEM_ID
```

### Storage

```bash
./feedctl storage
./feedctl storage --json
./feedctl storage reconcile
./feedctl storage reconcile --json
```

---

## JSON, dry-run и automation

Для скриптов используйте `--json`:

```bash
./feedctl sources test tg-llm-under-hood --json
./feedctl sync --source tg-llm-under-hood --json
./feedctl items list --unread --json
```

Для безопасных изменений используйте `--dry-run`:

```bash
./feedctl add rss https://example.com/feed.xml --dry-run --json
./feedctl add telegram @channel --max-items 50 --dry-run --json
./feedctl sources remove old-source --dry-run --json
```

Для non-interactive режима используйте `--yes`:

```bash
./feedctl sources disable noisy-source --yes
```

---

## Модель данных в двух словах

```text
~/.config/feedctl/
├── config.toml                 # декларативные настройки
└── sources.d/
    ├── habr.toml               # source definitions
    └── tg-llm-under-hood.toml

~/.feedctl/
├── feedctl.db                  # runtime state
├── content/                    # текущие Markdown-файлы
├── versions/                   # старые версии Markdown
├── tmp/
└── logs/
```

Декларативная часть отвечает на вопрос **«что подключено и как синхронизировать»**. Runtime часть отвечает на вопрос **«что уже найдено, прочитано, изменено и где лежит»**.

---

## Разработка

Проект следует TDD-процессу:

1. сначала regression/unit/integration тест;
2. убедиться, что он падает ожидаемо;
3. минимальная реализация;
4. targeted tests;
5. refactor;
6. полный прогон.

Полная проверка:

```bash
go test ./...
```

Форматирование:

```bash
gofmt -w cmd internal
```

OpenSpec specs:

```bash
openspec validate --specs --strict
```

---

## Статус проекта

`feedctl` сейчас — local-first MVP с фокусом на терминальный workflow, RSS, публичные Telegram-каналы и Markdown-архив. Приоритеты проекта: простота, воспроизводимость, локальные данные, понятные CLI-команды и TDD для всех изменений поведения.
