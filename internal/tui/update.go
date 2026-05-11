package tui

import (
	"time"

	"feedctl/internal/app"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd(m.interval())}
	if m.app.Config().Sync.SyncOnStartup {
		cmds = append(cmds, syncCmd(m.ctx, m.app))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		due := m.dueSources(time.Now())
		cmds := []tea.Cmd{tickCmd(m.interval())}
		if len(due) > 0 {
			m.syncing = true
			cmds = append(cmds, syncSourcesCmd(m.ctx, m.app, due))
		}
		return m, tea.Batch(cmds...)
	case syncMsg:
		m.syncing = false
		if msg.ok() {
			m.message = "sync ok"
		} else {
			m.message = "sync failed: " + msg.errorText()
		}
		_ = m.reload()
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
				m.search = trimLastRune(m.search)
			default:
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.search += string(msg.Runes)
				}
			}
			return m, nil
		}
		if m.filterMode {
			switch key {
			case "esc", "enter":
				m.filterMode = false
			case "backspace":
				m.filter = trimLastRune(m.filter)
				_ = m.reload()
			default:
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
					_ = m.reload()
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
				m.closeReader()
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.hasSelection() || m.selecting {
				m.cancelSelection()
			} else {
				m.closeReader()
			}
		case "h", "left":
			m.closeReader()
		case "?":
			m.help = true
		case "j", "down":
			m.move(1)
		case "k", "up":
			m.move(-1)
		case "g":
			m.cursor = 0
			m.updateSelectionRange()
		case "G":
			if m.listLen() > 0 {
				m.cursor = m.listLen() - 1
				m.updateSelectionRange()
			}
		case "ctrl+d":
			m.move(10)
		case "ctrl+u":
			m.move(-10)
		case "ctrl+f":
			m.move(20)
		case "ctrl+b":
			m.move(-20)
		case "enter":
			if m.section != 3 {
				m.markSelectedRead()
				m.reader = true
			}
		case "l", "right":
			if m.section != 3 {
				m.reader = true
			}
		case "v":
			m.toggleSelection()
		case " ":
			if m.hasSelection() {
				m.batchToggleRead()
			} else if err := m.withSelectedErr(func(item app.Item) error { return m.app.ToggleRead(item.ID) }); err != nil {
				m.setError("toggle read failed", err)
				return m, nil
			}
			_ = m.reload()
		case "u":
			if m.hasSelection() {
				m.batchSetRead(false)
			} else if err := m.withSelectedErr(func(item app.Item) error { return m.app.SetRead(item.ID, false) }); err != nil {
				m.setError("mark unread failed", err)
				return m, nil
			}
			_ = m.reload()
		case "s":
			if err := m.withSelectedErr(func(item app.Item) error { return m.app.ToggleStarred(item.ID) }); err != nil {
				m.setError("star failed", err)
				return m, nil
			}
			_ = m.reload()
		case "a":
			if err := m.withSelectedErr(func(item app.Item) error { return m.app.Archive(item.ID) }); err != nil {
				m.setError("archive failed", err)
				return m, nil
			}
			_ = m.reload()
		case "o":
			if err := m.withSelectedErr(func(item app.Item) error { return m.app.OpenItem(m.ctx, item.ID) }); err != nil {
				m.setError("open URL failed", err)
			}
		case "e":
			if err := m.withSelectedErr(func(item app.Item) error { return m.app.OpenMarkdown(m.ctx, item.ID) }); err != nil {
				m.setError("open Markdown failed", err)
			}
		case "m":
			if m.section != 3 {
				m.showFrontmatter = !m.showFrontmatter
				if m.showFrontmatter {
					m.message = "frontmatter shown"
				} else {
					m.message = "frontmatter hidden"
				}
			}
		case "tab":
			m.clearSelection()
			m.section = (m.section + 1) % len(sections)
			m.cursor = 0
			_ = m.reload()
		case "shift+tab":
			m.clearSelection()
			m.section = (m.section + len(sections) - 1) % len(sections)
			m.cursor = 0
			_ = m.reload()
		case "1", "2", "3", "4", "5", "6":
			m.clearSelection()
			m.section = int(key[0] - '1')
			m.cursor = 0
			_ = m.reload()
		case "/":
			m.searchMode = true
			m.filterMode = false
			m.search = ""
		case "n":
			m.findNext(1)
		case "N":
			m.findNext(-1)
		case "f":
			m.searchMode = false
			m.filterMode = true
			m.filter = ""
			if err := m.reload(); err == nil {
				m.message = "filter"
			}
		case "F":
			m.filter = ""
			m.filterMode = false
			_ = m.reload()
		case "A":
			m.showRemoved = !m.showRemoved
			_ = m.reload()
		case "r":
			if err := m.reload(); err == nil {
				m.message = "refreshed"
			}
		case "R":
			m.syncing = true
			m.message = "syncing..."
			return m, syncCmd(m.ctx, m.app)
		}
	}
	return m, nil
}
