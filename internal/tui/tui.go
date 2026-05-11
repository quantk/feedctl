package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"

	tea "github.com/charmbracelet/bubbletea"
)

var sections = []string{"Inbox", "Unread", "Starred", "Sources", "Removed Sources", "All Items"}

type syncMsg struct{ result any }
type tickMsg struct{}

type Model struct {
	app          *app.App
	items        []store.Item
	sources      []store.Source
	status       store.StatusSummary
	section      int
	cursor       int
	reader       bool
	help         bool
	searchMode   bool
	search       string
	message      string
	showRemoved  bool
	syncing      bool
	lastPeriodic map[string]time.Time
}

func Run(ctx context.Context) error {
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	m := NewModel(a)
	p := tea.NewProgram(m)
	_, err = p.Run()
	return err
}

func NewModel(a *app.App) Model {
	m := Model{app: a, showRemoved: a.Loaded.Config.TUI.ShowRemovedSources, lastPeriodic: map[string]time.Time{}}
	m.reload()
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd(m.interval())}
	if m.app.Loaded.Config.Sync.SyncOnStartup {
		cmds = append(cmds, syncCmd(m.app))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		due := m.dueSources(time.Now())
		cmds := []tea.Cmd{tickCmd(m.interval())}
		if len(due) > 0 {
			m.syncing = true
			cmds = append(cmds, syncSourcesCmd(m.app, due))
		}
		return m, tea.Batch(cmds...)
	case syncMsg:
		m.syncing = false
		m.message = "sync ok"
		m.reload()
		return m, nil
	case tea.KeyMsg:
		key := msg.String()
		if m.searchMode {
			switch key {
			case "esc":
				m.searchMode = false
			case "enter":
				m.searchMode = false
				m.findNext(1)
			case "backspace":
				if len(m.search) > 0 {
					m.search = m.search[:len(m.search)-1]
				}
			default:
				if len(key) == 1 {
					m.search += key
				}
			}
			return m, nil
		}
		if m.help {
			if key == "esc" || key == "q" || key == "?" {
				m.help = false
			}
			return m, nil
		}
		switch key {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if m.reader {
				m.reader = false
				return m, nil
			}
			return m, tea.Quit
		case "esc", "h", "left":
			m.reader = false
		case "?":
			m.help = true
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "g":
			m.cursor = 0
		case "G":
			if m.listLen() > 0 {
				m.cursor = m.listLen() - 1
			}
		case "ctrl+d":
			m.move(10)
		case "ctrl+u":
			m.move(-10)
		case "ctrl+f":
			m.move(20)
		case "ctrl+b":
			m.move(-20)
		case "enter", "l", "right":
			if m.section != 3 {
				m.reader = true
			}
		case " ":
			m.withSelected(func(item store.Item) { _ = m.app.ToggleRead(item.ID) })
			m.reload()
		case "u":
			m.withSelected(func(item store.Item) { _ = m.app.SetRead(item.ID, false) })
			m.reload()
		case "s":
			m.withSelected(func(item store.Item) { _ = m.app.ToggleStarred(item.ID) })
			m.reload()
		case "a":
			m.withSelected(func(item store.Item) { _ = m.app.Archive(item.ID) })
			m.reload()
		case "o":
			m.withSelected(func(item store.Item) { _ = m.app.OpenItem(context.Background(), item.ID) })
		case "e":
			m.withSelected(func(item store.Item) { _ = m.app.OpenMarkdown(context.Background(), item.ID) })
		case "tab":
			m.section = (m.section + 1) % len(sections)
			m.cursor = 0
			m.reload()
		case "shift+tab":
			m.section = (m.section + len(sections) - 1) % len(sections)
			m.cursor = 0
			m.reload()
		case "1", "2", "3", "4", "5", "6":
			m.section = int(key[0] - '1')
			m.cursor = 0
			m.reload()
		case "/":
			m.searchMode = true
			m.search = ""
		case "n":
			m.findNext(1)
		case "N":
			m.findNext(-1)
		case "f":
			m.searchMode = true
			m.search = ""
			m.message = "filter"
		case "F":
			m.search = ""
			m.reload()
		case "A":
			m.showRemoved = !m.showRemoved
			m.reload()
		case "r":
			m.reload()
			m.message = "refreshed"
		case "R":
			m.syncing = true
			m.message = "syncing..."
			return m, syncCmd(m.app)
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.help {
		return helpView()
	}
	var b strings.Builder
	b.WriteString("feedctl\n")
	for i, name := range sections {
		if i == m.section {
			b.WriteString("[" + name + "] ")
		} else {
			b.WriteString(name + " ")
		}
	}
	b.WriteString("\n\n")
	if m.searchMode {
		b.WriteString("/" + m.search + "\n")
	}
	if m.section == 3 {
		for i, src := range m.sources {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s [%s] %s %s\n", prefix, src.ID, src.Lifecycle, src.Type, src.Name))
		}
	} else {
		for i, item := range m.items {
			prefix := "  "
			if i == m.cursor {
				prefix = "> "
			}
			read := "●"
			if item.ReadAt != "" {
				read = " "
			}
			star := " "
			if item.Starred {
				star = "★"
			}
			b.WriteString(fmt.Sprintf("%s%s%s %s [%s]\n", prefix, read, star, item.Title, item.SourceID))
			if i > 30 {
				b.WriteString("  ...\n")
				break
			}
		}
		if m.reader {
			b.WriteString("\n" + m.preview() + "\n")
		}
	}
	b.WriteString("\n")
	b.WriteString(m.statusLine())
	b.WriteString("\n")
	return b.String()
}

func (m *Model) reload() {
	if m.app == nil {
		return
	}
	if m.section == 3 {
		m.sources, _ = m.app.Sources(true)
	} else {
		filter := store.ItemFilter{}
		switch m.section {
		case 1:
			filter.Unread = true
		case 2:
			filter.Starred = true
		case 4:
			filter.RemovedSources = true
		case 5:
			filter.AllItems = true
		}
		if m.showRemoved && m.section != 4 {
			filter.AllItems = true
		}
		items, _ := m.app.Items(filter)
		if m.search != "" {
			items = filterItems(items, m.search)
		}
		m.items = items
	}
	m.status, _ = m.app.Status()
	if m.cursor >= m.listLen() && m.listLen() > 0 {
		m.cursor = m.listLen() - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *Model) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if n := m.listLen(); n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
}

func (m Model) listLen() int {
	if m.section == 3 {
		return len(m.sources)
	}
	return len(m.items)
}

func (m Model) withSelected(fn func(store.Item)) {
	if m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	fn(m.items[m.cursor])
}

func (m *Model) findNext(dir int) {
	if m.search == "" || len(m.items) == 0 {
		return
	}
	needle := strings.ToLower(m.search)
	for step := 1; step <= len(m.items); step++ {
		idx := (m.cursor + dir*step + len(m.items)) % len(m.items)
		if strings.Contains(strings.ToLower(m.items[idx].Title), needle) || strings.Contains(strings.ToLower(m.items[idx].SourceID), needle) {
			m.cursor = idx
			return
		}
	}
}

func filterItems(items []store.Item, q string) []store.Item {
	q = strings.ToLower(q)
	var out []store.Item
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Title), q) || strings.Contains(strings.ToLower(item.SourceID), q) {
			out = append(out, item)
		}
	}
	return out
}

func (m Model) preview() string {
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return "No item selected"
	}
	path, err := m.app.MarkdownPath(m.items[m.cursor].ID)
	if err != nil {
		return err.Error()
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return err.Error()
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > 25 {
		lines = lines[:25]
	}
	return strings.Join(lines, "\n")
}

func (m Model) statusLine() string {
	syncStatus := m.status.LatestSyncStatus
	if m.syncing {
		syncStatus = "syncing"
	}
	return fmt.Sprintf("●%d unread | src:%d | removed:%d | disk:%s | sync %s | %s %s", m.status.UnreadCount, m.status.SourceCount, m.status.RemovedSourceCount, store.HumanBytes(m.status.Storage.Total()), syncStatus, m.status.LatestSyncAt, m.message)
}

func (m Model) interval() time.Duration {
	d, err := config.ParseDuration(m.app.Loaded.Config.Sync.DefaultInterval)
	if err != nil || d <= 0 {
		return 5 * time.Minute
	}
	return d
}

func (m *Model) dueSources(now time.Time) []string {
	if m.lastPeriodic == nil {
		m.lastPeriodic = map[string]time.Time{}
	}
	var due []string
	for _, src := range m.app.Loaded.Sources {
		if !src.Enabled {
			continue
		}
		intervalValue := src.Interval
		if intervalValue == "" {
			intervalValue = m.app.Loaded.Config.Sync.DefaultInterval
		}
		interval, err := config.ParseDuration(intervalValue)
		if err != nil || interval <= 0 {
			interval = 5 * time.Minute
		}
		last := m.lastPeriodic[src.ID]
		if last.IsZero() || now.Sub(last) >= interval {
			due = append(due, src.ID)
			m.lastPeriodic[src.ID] = now
		}
	}
	return due
}

func syncCmd(a *app.App) tea.Cmd {
	return syncSourcesCmd(a, []string{""})
}

func syncSourcesCmd(a *app.App, sourceIDs []string) tea.Cmd {
	return func() tea.Msg {
		for _, sourceID := range sourceIDs {
			_ = a.Sync(context.Background(), sourceID)
		}
		return syncMsg{result: true}
	}
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func helpView() string {
	return `feedctl help

Navigation: j/k or arrows, h/l, g/G, Ctrl+d/u, Ctrl+f/b
Sections: 1 Inbox, 2 Unread, 3 Starred, 4 Sources, 5 Removed Sources, 6 All Items, Tab/Shift+Tab
Search/filter: / search, n/N next/prev, f filter hint, F clear, A toggle removed-source items
Items: Enter/l open, Space read/unread, u unread, s star, a archive, o open URL, e edit Markdown
Sync: r refresh, R sync all
Other: ? help, Esc back/close, q quit
`
}
