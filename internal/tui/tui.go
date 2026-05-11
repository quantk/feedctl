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
	"github.com/charmbracelet/lipgloss"
)

const (
	defaultViewWidth  = 100
	defaultViewHeight = 30
	minViewWidth      = 20
	minViewHeight     = 8
	wideViewWidth     = 92
	selectedMarker    = "┃"
)

var sections = []string{"Inbox", "Unread", "Starred", "Sources", "Removed Sources", "All Items"}

var (
	nord0  = lipgloss.Color("#2E3440")
	nord1  = lipgloss.Color("#3B4252")
	nord2  = lipgloss.Color("#434C5E")
	nord3  = lipgloss.Color("#4C566A")
	nord4  = lipgloss.Color("#D8DEE9")
	nord5  = lipgloss.Color("#E5E9F0")
	nord6  = lipgloss.Color("#ECEFF4")
	nord7  = lipgloss.Color("#8FBCBB")
	nord8  = lipgloss.Color("#88C0D0")
	nord9  = lipgloss.Color("#81A1C1")
	nord10 = lipgloss.Color("#5E81AC")
	nord11 = lipgloss.Color("#BF616A")
	nord13 = lipgloss.Color("#EBCB8B")
	nord14 = lipgloss.Color("#A3BE8C")
	nord15 = lipgloss.Color("#B48EAD")

	appStyle = lipgloss.NewStyle().Foreground(nord4).Background(nord0)

	headerStyle = lipgloss.NewStyle().Foreground(nord6).Background(nord1).Bold(true)
	tabsStyle   = lipgloss.NewStyle().Foreground(nord9).Background(nord0)
	bodyStyle   = lipgloss.NewStyle().Foreground(nord4).Background(nord0)
	panelStyle  = lipgloss.NewStyle().Foreground(nord4).Background(nord0)
	mutedStyle  = lipgloss.NewStyle().Foreground(nord3).Background(nord0)
	borderStyle = lipgloss.NewStyle().Foreground(nord3).Background(nord0)

	selectedRowStyle    = lipgloss.NewStyle().Foreground(nord6).Background(nord2).Bold(true)
	selectedMarkerStyle = lipgloss.NewStyle().Foreground(nord8).Background(nord2).Bold(true)
	unreadStyle         = lipgloss.NewStyle().Foreground(nord8).Background(nord0).Bold(true)
	selectedUnreadStyle = lipgloss.NewStyle().Foreground(nord8).Background(nord2).Bold(true)
	readStyle           = lipgloss.NewStyle().Foreground(nord3).Background(nord0)
	selectedReadStyle   = lipgloss.NewStyle().Foreground(nord4).Background(nord2)
	starStyle           = lipgloss.NewStyle().Foreground(nord13).Background(nord0).Bold(true)
	selectedStarStyle   = lipgloss.NewStyle().Foreground(nord13).Background(nord2).Bold(true)
	successStyle        = lipgloss.NewStyle().Foreground(nord14).Background(nord0).Bold(true)
	warningStyle        = lipgloss.NewStyle().Foreground(nord13).Background(nord0).Bold(true)
	errorStyle          = lipgloss.NewStyle().Foreground(nord11).Background(nord0).Bold(true)
	accentStyle         = lipgloss.NewStyle().Foreground(nord7).Background(nord0).Bold(true)
	selectedAccentStyle = lipgloss.NewStyle().Foreground(nord7).Background(nord2).Bold(true)
	selectedMutedStyle  = lipgloss.NewStyle().Foreground(nord9).Background(nord2)
	helpTitleStyle      = lipgloss.NewStyle().Foreground(nord8).Background(nord0).Bold(true)
)

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
	filterMode   bool
	search       string
	filter       string
	message      string
	showRemoved  bool
	syncing      bool
	width        int
	height       int
	lastPeriodic map[string]time.Time
}

func Run(ctx context.Context) error {
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	m := NewModel(a)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func NewModel(a *app.App) Model {
	showRemoved := false
	if a != nil {
		showRemoved = a.Loaded.Config.TUI.ShowRemovedSources
	}
	m := Model{app: a, showRemoved: showRemoved, width: defaultViewWidth, height: defaultViewHeight, lastPeriodic: map[string]time.Time{}}
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
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
				m.reload()
			default:
				if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 {
					m.filter += string(msg.Runes)
					m.reload()
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
		case "esc", "h", "left":
			m.closeReader()
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
		case "enter":
			if m.section != 3 {
				m.markSelectedRead()
				m.reader = true
			}
		case "l", "right":
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
			m.reload()
			m.message = "filter"
		case "F":
			m.filter = ""
			m.filterMode = false
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
	width, height := m.viewSize()
	if m.help {
		return m.renderHelp(width, height)
	}

	header := m.renderHeader(width)
	tabs := m.renderTabs(width)
	status := m.renderStatus(width)
	bodyHeight := height - lipgloss.Height(header) - lipgloss.Height(tabs) - lipgloss.Height(status)
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := m.renderContent(width, bodyHeight)
	view := lipgloss.JoinVertical(lipgloss.Left, header, tabs, body, status)
	return fitHeight(view, width, height, appStyle)
}

func (m Model) viewSize() (int, int) {
	width := m.width
	if width <= 0 {
		width = defaultViewWidth
	}
	if width < minViewWidth {
		width = minViewWidth
	}
	height := m.height
	if height <= 0 {
		height = defaultViewHeight
	}
	if height < minViewHeight {
		height = minViewHeight
	}
	return width, height
}

func (m Model) renderHeader(width int) string {
	right := fmt.Sprintf("%d unread · %s", m.status.UnreadCount, m.currentSyncStatus())
	left := " feedctl"
	space := width - lipgloss.Width(left) - lipgloss.Width(right) - 1
	if space < 1 {
		space = 1
	}
	line := left + strings.Repeat(" ", space) + right + " "
	return renderPlainLine(line, width, headerStyle)
}

func (m Model) renderTabs(width int) string {
	parts := make([]string, 0, len(sections))
	for i, name := range sections {
		label := fmt.Sprintf("%d %s", i+1, name)
		if i == m.section {
			label = "[" + label + "]"
		}
		parts = append(parts, label)
	}
	line := " " + strings.Join(parts, "  ")
	if m.searchMode {
		line += "  /" + m.search
	} else if m.filterMode {
		line += "  filter:" + m.filter
	} else if m.filter != "" {
		line += "  filtered:" + m.filter
	}
	return renderPlainLine(line, width, tabsStyle)
}

func (m Model) renderContent(width, height int) string {
	if height <= 0 {
		return ""
	}
	if width >= wideViewWidth {
		leftWidth := width * 45 / 100
		if leftWidth < 34 {
			leftWidth = 34
		}
		rightWidth := width - leftWidth - 1
		if rightWidth < 20 {
			return m.renderNarrowContent(width, height)
		}
		left := m.renderList(leftWidth, height)
		rule := verticalRule(height)
		right := m.renderPreviewPane(rightWidth, height)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, rule, right)
	}
	return m.renderNarrowContent(width, height)
}

func (m Model) renderNarrowContent(width, height int) string {
	if m.reader && m.section != 3 && height > 8 {
		listHeight := height / 2
		previewHeight := height - listHeight
		list := m.renderList(width, listHeight)
		preview := m.renderPreviewPane(width, previewHeight)
		return fitHeight(lipgloss.JoinVertical(lipgloss.Left, list, preview), width, height, bodyStyle)
	}
	return m.renderList(width, height)
}

func (m Model) renderList(width, height int) string {
	lines := make([]string, 0, height)
	if m.searchMode {
		lines = append(lines, renderPlainLine(" /"+m.search, width, accentStyle))
	} else if m.filterMode {
		lines = append(lines, renderPlainLine(" filter:"+m.filter, width, accentStyle))
	} else if m.filter != "" {
		lines = append(lines, renderPlainLine(" filtered:"+m.filter, width, mutedStyle))
	}
	rowsAvailable := height - len(lines)
	if rowsAvailable < 0 {
		rowsAvailable = 0
	}

	if m.section == 3 {
		start := visibleStart(m.cursor, len(m.sources), rowsAvailable)
		end := start + rowsAvailable
		if end > len(m.sources) {
			end = len(m.sources)
		}
		for i := start; i < end; i++ {
			lines = append(lines, m.renderSourceRow(m.sources[i], i == m.cursor, width))
		}
	} else {
		start := visibleStart(m.cursor, len(m.items), rowsAvailable)
		end := start + rowsAvailable
		if end > len(m.items) {
			end = len(m.items)
		}
		for i := start; i < end; i++ {
			lines = append(lines, m.renderItemRow(m.items[i], i == m.cursor, width))
		}
	}

	if len(lines) == 0 && height > 0 {
		lines = append(lines, renderPlainLine("  No entries", width, mutedStyle))
	}
	return fitLines(lines, width, height, bodyStyle)
}

func (m Model) renderItemRow(item store.Item, selected bool, width int) string {
	baseStyle := bodyStyle
	markerStyle := bodyStyle
	readMarkerStyle := readStyle
	unreadMarkerStyle := unreadStyle
	starMarkerStyle := starStyle
	mutedRowStyle := mutedStyle
	if selected {
		baseStyle = selectedRowStyle
		markerStyle = selectedMarkerStyle
		readMarkerStyle = selectedReadStyle
		unreadMarkerStyle = selectedUnreadStyle
		starMarkerStyle = selectedStarStyle
		mutedRowStyle = selectedMutedStyle
	}

	marker := " "
	if selected {
		marker = selectedMarker
	}
	read := readMarkerStyle.Render(" ")
	if item.ReadAt == "" {
		read = unreadMarkerStyle.Render("●")
	}
	star := readMarkerStyle.Render(" ")
	if item.Starred {
		star = starMarkerStyle.Render("★")
	}
	source := " [" + item.SourceID + "]"
	sourceWidth := lipgloss.Width(source)
	titleWidth := width - 6 - sourceWidth
	if titleWidth < 8 {
		titleWidth = width - 6
		source = ""
	}
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := truncateText(item.Title, titleWidth)
	row := markerStyle.Render(marker) + baseStyle.Render(" ") + read + star + baseStyle.Render(" ") + baseStyle.Render(title) + mutedRowStyle.Render(source)
	return fillStyled(row, width, baseStyle)
}

func (m Model) renderSourceRow(src store.Source, selected bool, width int) string {
	baseStyle := bodyStyle
	markerStyle := bodyStyle
	accentRowStyle := accentStyle
	mutedRowStyle := mutedStyle
	if selected {
		baseStyle = selectedRowStyle
		markerStyle = selectedMarkerStyle
		accentRowStyle = selectedAccentStyle
		mutedRowStyle = selectedMutedStyle
	}

	marker := " "
	if selected {
		marker = selectedMarker
	}
	status := src.Lifecycle
	if status == "" {
		status = "active"
	}
	statusText := syncStateStyle(status, selected).Render("[" + status + "]")
	name := strings.TrimSpace(src.Name)
	if name == "" {
		name = src.URL
	}
	prefixWidth := lipgloss.Width(marker) + 1 + lipgloss.Width(src.ID) + 1 + lipgloss.Width("["+status+"]") + 1 + lipgloss.Width(src.Type) + 1
	nameWidth := width - prefixWidth
	if nameWidth < 1 {
		nameWidth = 1
	}
	row := markerStyle.Render(marker) + baseStyle.Render(" ") + accentRowStyle.Render(src.ID) + baseStyle.Render(" ") + statusText + baseStyle.Render(" ") + mutedRowStyle.Render(src.Type) + baseStyle.Render(" ") + baseStyle.Render(truncateText(name, nameWidth))
	return fillStyled(row, width, baseStyle)
}

func (m Model) renderPreviewPane(width, height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, 0, height)
	if m.section == 3 {
		lines = append(lines, renderPlainLine(" Source", width, helpTitleStyle))
		lines = append(lines, renderPlainLine("", width, panelStyle))
		if m.cursor < 0 || m.cursor >= len(m.sources) {
			lines = append(lines, renderPlainLine(" No source selected", width, mutedStyle))
			return fitLines(lines, width, height, panelStyle)
		}
		src := m.sources[m.cursor]
		lines = append(lines,
			renderPlainLine(" ID: "+src.ID, width, panelStyle),
			renderPlainLine(" Name: "+src.Name, width, panelStyle),
			renderPlainLine(" Type: "+src.Type, width, panelStyle),
			renderPlainLine(" State: "+src.Lifecycle, width, panelStyle),
			renderPlainLine(" URL: "+src.URL, width, mutedStyle),
		)
		if src.LastError != "" {
			lines = append(lines, renderPlainLine(" Error: "+src.LastError, width, errorStyle))
		}
		return fitLines(lines, width, height, panelStyle)
	}

	label := " Preview"
	if m.reader {
		label = " Reader"
	}
	lines = append(lines, renderPlainLine(label, width, helpTitleStyle))
	lines = append(lines, renderPlainLine("", width, panelStyle))
	if m.cursor < 0 || m.cursor >= len(m.items) {
		lines = append(lines, renderPlainLine(" No item selected", width, mutedStyle))
		return fitLines(lines, width, height, panelStyle)
	}
	item := m.items[m.cursor]
	lines = append(lines, renderPlainLine(" "+item.Title, width, accentStyle))
	lines = append(lines, renderPlainLine(" "+item.SourceID+" · "+item.PublishedAt, width, mutedStyle))
	lines = append(lines, renderPlainLine("", width, panelStyle))
	for _, line := range m.previewLines(height - len(lines)) {
		lines = append(lines, renderPlainLine(" "+line, width, panelStyle))
	}
	return fitLines(lines, width, height, panelStyle)
}

func (m Model) renderStatus(width int) string {
	syncStatus := m.currentSyncStatus()
	parts := []string{
		unreadStyle.Render(fmt.Sprintf("●%d unread", m.status.UnreadCount)),
		mutedStyle.Render(fmt.Sprintf("src:%d", m.status.SourceCount)),
		mutedStyle.Render(fmt.Sprintf("removed:%d", m.status.RemovedSourceCount)),
		mutedStyle.Render("disk:" + store.HumanBytes(m.status.Storage.Total())),
		syncStateStyle(syncStatus).Render("sync " + syncStatus),
	}
	if m.status.LatestSyncAt != "" {
		parts = append(parts, mutedStyle.Render(m.status.LatestSyncAt))
	}
	if m.message != "" {
		parts = append(parts, accentStyle.Render(m.message))
	}
	parts = append(parts, mutedStyle.Render("? help · q quit"))

	line := appStyle.Render(" ")
	separator := appStyle.Render(" │ ")
	for i, part := range parts {
		if i > 0 {
			line += separator
		}
		line += part
	}
	return fillStyled(line, width, appStyle)
}

func (m Model) renderHelp(width, height int) string {
	lines := []string{
		renderPlainLine(" feedctl help", width, helpTitleStyle),
		renderPlainLine("", width, appStyle),
		renderPlainLine(" Navigation", width, accentStyle),
		renderPlainLine("   j/k or arrows, h/l, g/G, Ctrl+d/u, Ctrl+f/b", width, bodyStyle),
		renderPlainLine(" Sections", width, accentStyle),
		renderPlainLine("   1 Inbox, 2 Unread, 3 Starred, 4 Sources, 5 Removed Sources, 6 All Items", width, bodyStyle),
		renderPlainLine("   Tab/Shift+Tab", width, bodyStyle),
		renderPlainLine(" Search/filter", width, accentStyle),
		renderPlainLine("   / search, n/N next/prev, f live filter, F clear, A toggle removed-source items", width, bodyStyle),
		renderPlainLine(" Items", width, accentStyle),
		renderPlainLine("   Enter/l open, Space read/unread, u unread, s star, a archive", width, bodyStyle),
		renderPlainLine("   o open URL, e edit Markdown", width, bodyStyle),
		renderPlainLine(" Sync", width, accentStyle),
		renderPlainLine("   r refresh, R sync all", width, bodyStyle),
		renderPlainLine(" Other", width, accentStyle),
		renderPlainLine("   ? help, Esc back/close, q quit", width, bodyStyle),
	}
	return fitLines(lines, width, height, appStyle)
}

func (m Model) currentSyncStatus() string {
	syncStatus := m.status.LatestSyncStatus
	if m.syncing {
		syncStatus = "syncing"
	}
	if syncStatus == "" {
		syncStatus = "never"
	}
	return syncStatus
}

func syncStateStyle(status string, selected ...bool) lipgloss.Style {
	background := nord0
	if len(selected) > 0 && selected[0] {
		background = nord2
	}
	s := strings.ToLower(status)
	switch {
	case strings.Contains(s, "syncing") || strings.Contains(s, "running"):
		return lipgloss.NewStyle().Foreground(nord13).Background(background).Bold(true)
	case strings.Contains(s, "fail") || strings.Contains(s, "error"):
		return lipgloss.NewStyle().Foreground(nord11).Background(background).Bold(true)
	case strings.Contains(s, "ok") || strings.Contains(s, "success"):
		return lipgloss.NewStyle().Foreground(nord14).Background(background).Bold(true)
	case strings.Contains(s, "removed") || strings.Contains(s, "disabled"):
		return lipgloss.NewStyle().Foreground(nord13).Background(background).Bold(true)
	default:
		return lipgloss.NewStyle().Foreground(nord3).Background(background)
	}
}

func visibleStart(cursor, total, rows int) int {
	if rows <= 0 || total <= rows || cursor < rows {
		return 0
	}
	start := cursor - rows + 1
	if start < 0 {
		return 0
	}
	if start > total-rows {
		return total - rows
	}
	return start
}

func verticalRule(height int) string {
	if height <= 0 {
		return ""
	}
	lines := make([]string, height)
	for i := range lines {
		lines[i] = borderStyle.Render("│")
	}
	return strings.Join(lines, "\n")
}

func renderPlainLine(text string, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	line := truncateText(text, width)
	if padding := width - lipgloss.Width(line); padding > 0 {
		line += strings.Repeat(" ", padding)
	}
	return style.Render(line)
}

func fillStyled(text string, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	visible := lipgloss.Width(text)
	if visible >= width {
		return text
	}
	return text + style.Render(strings.Repeat(" ", width-visible))
}

func fitLines(lines []string, width, height int, style lipgloss.Style) string {
	if height <= 0 {
		return ""
	}
	out := make([]string, 0, height)
	for _, line := range lines {
		if len(out) >= height {
			break
		}
		out = append(out, line)
	}
	for len(out) < height {
		out = append(out, renderPlainLine("", width, style))
	}
	return strings.Join(out, "\n")
}

func fitHeight(view string, width, height int, style lipgloss.Style) string {
	if height <= 0 {
		return ""
	}
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	return fitLines(lines, width, height, style)
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	runes := []rune(text)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
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
		if m.filter != "" {
			items = filterItems(items, m.filter)
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

func (m *Model) closeReader() {
	if !m.reader {
		return
	}
	m.reader = false
	m.reload()
}

func (m *Model) markSelectedRead() {
	if m.app == nil || m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	item := m.items[m.cursor]
	if item.ReadAt != "" {
		return
	}
	if err := m.app.SetRead(item.ID, true); err != nil {
		return
	}
	m.items[m.cursor].ReadAt = time.Now().Format(time.RFC3339)
	m.status, _ = m.app.Status()
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

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}

func (m Model) preview() string {
	return strings.Join(m.previewLines(25), "\n")
}

func (m Model) previewLines(limit int) []string {
	if limit <= 0 {
		return nil
	}
	if m.cursor < 0 || m.cursor >= len(m.items) {
		return []string{"No item selected"}
	}
	path, err := m.app.MarkdownPath(m.items[m.cursor].ID)
	if err != nil {
		return []string{err.Error()}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return []string{err.Error()}
	}
	lines := strings.Split(string(b), "\n")
	if len(lines) > limit {
		lines = lines[:limit]
	}
	return lines
}

func (m Model) statusLine() string {
	return fmt.Sprintf("●%d unread | src:%d | removed:%d | disk:%s | sync %s | %s %s", m.status.UnreadCount, m.status.SourceCount, m.status.RemovedSourceCount, store.HumanBytes(m.status.Storage.Total()), m.currentSyncStatus(), m.status.LatestSyncAt, m.message)
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
	m := Model{width: defaultViewWidth, height: defaultViewHeight, help: true}
	return m.renderHelp(defaultViewWidth, defaultViewHeight)
}
