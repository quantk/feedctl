package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/metrics"
	"feedctl/internal/store"
	feedSync "feedctl/internal/sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

const (
	defaultViewWidth  = 100
	defaultViewHeight = 30
	minViewWidth      = 20
	minViewHeight     = 8
	wideViewWidth     = 92
	selectedMarker    = "┃"
	multiSelectMarker = "✓"
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

type syncMsg struct {
	results []feedSync.Result
}
type tickMsg struct{}

type Model struct {
	ctx             context.Context
	app             *app.App
	items           []store.Item
	sources         []store.Source
	status          store.StatusSummary
	section         int
	cursor          int
	reader          bool
	help            bool
	searchMode      bool
	filterMode      bool
	showFrontmatter bool
	selecting       bool
	selectionAnchor string
	selectedItems   map[string]struct{}
	search          string
	filter          string
	message         string
	showRemoved     bool
	syncing         bool
	width           int
	height          int
	lastPeriodic    map[string]time.Time
}

func Run(ctx context.Context) error {
	a, err := app.Open(ctx)
	if err != nil {
		return err
	}
	defer a.Close()
	m := NewModelWithContext(ctx, a)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

func NewModel(a *app.App) Model {
	return NewModelWithContext(context.Background(), a)
}

func NewModelWithContext(ctx context.Context, a *app.App) Model {
	if ctx == nil {
		ctx = context.Background()
	}
	showRemoved := false
	if a != nil {
		showRemoved = a.Loaded.Config.TUI.ShowRemovedSources
	}
	m := Model{ctx: ctx, app: a, showRemoved: showRemoved, selectedItems: map[string]struct{}{}, width: defaultViewWidth, height: defaultViewHeight, lastPeriodic: map[string]time.Time{}}
	_ = m.reload()
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tickCmd(m.interval())}
	if m.app.Loaded.Config.Sync.SyncOnStartup {
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
			} else if err := m.withSelectedErr(func(item store.Item) error { return m.app.ToggleRead(item.ID) }); err != nil {
				m.setError("toggle read failed", err)
				return m, nil
			}
			_ = m.reload()
		case "u":
			if m.hasSelection() {
				m.batchSetRead(false)
			} else if err := m.withSelectedErr(func(item store.Item) error { return m.app.SetRead(item.ID, false) }); err != nil {
				m.setError("mark unread failed", err)
				return m, nil
			}
			_ = m.reload()
		case "s":
			if err := m.withSelectedErr(func(item store.Item) error { return m.app.ToggleStarred(item.ID) }); err != nil {
				m.setError("star failed", err)
				return m, nil
			}
			_ = m.reload()
		case "a":
			if err := m.withSelectedErr(func(item store.Item) error { return m.app.Archive(item.ID) }); err != nil {
				m.setError("archive failed", err)
				return m, nil
			}
			_ = m.reload()
		case "o":
			if err := m.withSelectedErr(func(item store.Item) error { return m.app.OpenItem(m.ctx, item.ID) }); err != nil {
				m.setError("open URL failed", err)
			}
		case "e":
			if err := m.withSelectedErr(func(item store.Item) error { return m.app.OpenMarkdown(m.ctx, item.ID) }); err != nil {
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
	if selected := len(m.selectedItems); selected > 0 {
		line += fmt.Sprintf("  selected:%d", selected)
	}
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
			lines = append(lines, m.renderItemRow(m.items[i], i == m.cursor, width, m.itemSelected(m.items[i].ID)))
		}
	}

	if len(lines) == 0 && height > 0 {
		lines = append(lines, renderPlainLine("  No entries", width, mutedStyle))
	}
	return fitLines(lines, width, height, bodyStyle)
}

func (m Model) renderItemRow(item store.Item, selected bool, width int, multiSelected ...bool) string {
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
	selection := baseStyle.Render(" ")
	if len(multiSelected) > 0 && multiSelected[0] {
		selection = accentStyle.Render(multiSelectMarker)
		if selected {
			selection = selectedAccentStyle.Render(multiSelectMarker)
		}
	}
	read := readMarkerStyle.Render(" ")
	if item.ReadAt == "" {
		read = unreadMarkerStyle.Render("●")
	}
	star := readMarkerStyle.Render(" ")
	if item.Starred {
		star = starMarkerStyle.Render("★")
	}
	metricsText := formatCompactMetrics(item.Metrics)
	source := " [" + item.SourceID + "]"
	suffix := source
	if metricsText != "" {
		suffix = " " + metricsText + source
	}
	titleWidth := width - 7 - lipgloss.Width(suffix)
	if titleWidth < 8 && metricsText != "" {
		metricsText = ""
		suffix = source
		titleWidth = width - 7 - lipgloss.Width(suffix)
	}
	if titleWidth < 8 {
		titleWidth = width - 7
		suffix = ""
	}
	if titleWidth < 1 {
		titleWidth = 1
	}
	title := truncateText(item.Title, titleWidth)
	row := markerStyle.Render(marker) + baseStyle.Render(" ") + selection + read + star + baseStyle.Render(" ") + baseStyle.Render(title) + mutedRowStyle.Render(suffix)
	return fillStyled(row, width, baseStyle)
}

func formatCompactMetrics(itemMetrics *metrics.ItemMetrics) string {
	if itemMetrics == nil {
		return ""
	}
	var parts []string
	if itemMetrics.Score != nil {
		parts = append(parts, formatScore(*itemMetrics.Score))
	}
	if itemMetrics.CommentsCount != nil {
		parts = append(parts, fmt.Sprintf("%dc", *itemMetrics.CommentsCount))
	}
	return strings.Join(parts, " · ")
}

func formatExpandedMetrics(itemMetrics *metrics.ItemMetrics) string {
	if itemMetrics == nil {
		return ""
	}
	var parts []string
	if itemMetrics.Score != nil {
		parts = append(parts, "Score: "+formatScore(*itemMetrics.Score))
	}
	if itemMetrics.CommentsCount != nil {
		parts = append(parts, fmt.Sprintf("Comments: %d", *itemMetrics.CommentsCount))
	}
	if itemMetrics.VotesCount != nil {
		parts = append(parts, fmt.Sprintf("Votes: %d", *itemMetrics.VotesCount))
	}
	if itemMetrics.FavoritesCount != nil {
		parts = append(parts, fmt.Sprintf("Favorites: %d", *itemMetrics.FavoritesCount))
	}
	if itemMetrics.ReadingCount != nil {
		parts = append(parts, fmt.Sprintf("Reads: %d", *itemMetrics.ReadingCount))
	}
	return strings.Join(parts, " · ")
}

func formatScore(score int) string {
	if score > 0 {
		return fmt.Sprintf("+%d", score)
	}
	return fmt.Sprintf("%d", score)
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
	if metricsText := formatExpandedMetrics(item.Metrics); metricsText != "" {
		lines = append(lines, renderPlainLine(" "+metricsText, width, mutedStyle))
	}
	lines = append(lines, renderPlainLine("", width, panelStyle))
	for _, line := range m.previewLines(height-len(lines), width-2) {
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
		messageStyle := accentStyle
		if isErrorMessage(m.message) {
			messageStyle = errorStyle
		}
		parts = append(parts, messageStyle.Render(m.message))
	}
	parts = append(parts, mutedStyle.Render("? help · v select · m frontmatter · q quit"))

	for len(parts) > 0 {
		line := renderStatusParts(parts)
		if lipgloss.Width(line) <= width {
			return fillStyled(line, width, appStyle)
		}
		parts = parts[:len(parts)-1]
	}
	return renderPlainLine(fmt.Sprintf(" ●%d unread · sync %s", m.status.UnreadCount, syncStatus), width, appStyle)
}

func isErrorMessage(message string) bool {
	lower := strings.ToLower(message)
	return strings.Contains(lower, "failed") || strings.Contains(lower, "error")
}

func renderStatusParts(parts []string) string {
	line := appStyle.Render(" ")
	separator := appStyle.Render(" │ ")
	for i, part := range parts {
		if i > 0 {
			line += separator
		}
		line += part
	}
	return line
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
		renderPlainLine("   Enter/l open, v visual select, Space read/unread, u unread, s star, a archive", width, bodyStyle),
		renderPlainLine("   o open URL, e edit Markdown, m toggle frontmatter", width, bodyStyle),
		renderPlainLine(" Sync", width, accentStyle),
		renderPlainLine("   r refresh, R sync all", width, bodyStyle),
		renderPlainLine(" Other", width, accentStyle),
		renderPlainLine("   ? help, Esc back/close/cancel selection, q quit", width, bodyStyle),
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
	if xansi.StringWidth(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return xansi.Truncate(text, width, "…")
}

func (m *Model) reload() error {
	if m.app == nil {
		return nil
	}
	if m.section == 3 {
		sources, err := m.app.Sources(true)
		if err != nil {
			m.setError("reload failed", err)
			return err
		}
		m.clearSelection()
		m.sources = sources
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
		items, err := m.app.Items(filter)
		if err != nil {
			m.setError("reload failed", err)
			return err
		}
		if m.filter != "" {
			items = filterItems(items, m.filter)
		}
		m.items = items
		m.pruneSelection()
	}
	status, err := m.app.Status()
	if err != nil {
		m.setError("reload failed", err)
		return err
	}
	m.status = status
	if m.cursor >= m.listLen() && m.listLen() > 0 {
		m.cursor = m.listLen() - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return nil
}

func (m *Model) move(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if n := m.listLen(); n > 0 && m.cursor >= n {
		m.cursor = n - 1
	}
	m.updateSelectionRange()
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
	_ = m.reload()
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
		m.setError("mark read failed", err)
		return
	}
	m.items[m.cursor].ReadAt = time.Now().Format(time.RFC3339)
	status, err := m.app.Status()
	if err != nil {
		m.setError("reload failed", err)
		return
	}
	m.status = status
}

func (m Model) withSelected(fn func(store.Item)) {
	if m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	fn(m.items[m.cursor])
}

func (m Model) withSelectedErr(fn func(store.Item) error) error {
	if m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	return fn(m.items[m.cursor])
}

func (m *Model) setError(prefix string, err error) {
	if err == nil {
		return
	}
	m.message = prefix + ": " + err.Error()
}

func (m *Model) toggleSelection() {
	if m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	if m.selecting {
		m.selecting = false
		m.selectionAnchor = ""
		m.setSelectionMessage()
		return
	}
	if m.selectedItems == nil {
		m.selectedItems = map[string]struct{}{}
	}
	m.selectedItems = map[string]struct{}{}
	m.selecting = true
	m.selectionAnchor = m.items[m.cursor].ID
	m.updateSelectionRange()
	m.setSelectionMessage()
}

func (m Model) hasSelection() bool {
	return len(m.selectedItems) > 0
}

func (m Model) itemSelected(id string) bool {
	_, ok := m.selectedItems[id]
	return ok
}

func (m *Model) clearSelection() {
	m.selecting = false
	m.selectionAnchor = ""
	if len(m.selectedItems) == 0 {
		return
	}
	m.selectedItems = map[string]struct{}{}
}

func (m *Model) pruneSelection() {
	if len(m.selectedItems) == 0 {
		return
	}
	visible := make(map[string]struct{}, len(m.items))
	for _, item := range m.items {
		visible[item.ID] = struct{}{}
	}
	for id := range m.selectedItems {
		if _, ok := visible[id]; !ok {
			delete(m.selectedItems, id)
		}
	}
	if len(m.selectedItems) == 0 {
		m.clearSelection()
	}
}

func (m *Model) cancelSelection() {
	m.clearSelection()
	m.message = "selection cancelled"
}

func (m *Model) updateSelectionRange() {
	if !m.selecting || m.section == 3 || m.cursor < 0 || m.cursor >= len(m.items) {
		return
	}
	anchor := -1
	for i, item := range m.items {
		if item.ID == m.selectionAnchor {
			anchor = i
			break
		}
	}
	if anchor < 0 {
		m.clearSelection()
		return
	}
	start, end := anchor, m.cursor
	if start > end {
		start, end = end, start
	}
	selected := make(map[string]struct{}, end-start+1)
	for i := start; i <= end; i++ {
		selected[m.items[i].ID] = struct{}{}
	}
	m.selectedItems = selected
}

func (m Model) selectedVisibleItems() []store.Item {
	if len(m.selectedItems) == 0 {
		return nil
	}
	items := make([]store.Item, 0, len(m.selectedItems))
	for _, item := range m.items {
		if m.itemSelected(item.ID) {
			items = append(items, item)
		}
	}
	return items
}

func (m *Model) batchToggleRead() {
	items := m.selectedVisibleItems()
	if m.app == nil || len(items) == 0 {
		return
	}
	markRead := false
	for _, item := range items {
		if item.ReadAt == "" {
			markRead = true
			break
		}
	}
	m.batchSetReadForItems(items, markRead)
}

func (m *Model) batchSetRead(read bool) {
	items := m.selectedVisibleItems()
	if m.app == nil || len(items) == 0 {
		return
	}
	m.batchSetReadForItems(items, read)
}

func (m *Model) batchSetReadForItems(items []store.Item, read bool) {
	for _, item := range items {
		_ = m.app.SetRead(item.ID, read)
	}
	m.clearSelection()
	if read {
		m.message = fmt.Sprintf("%d marked read", len(items))
	} else {
		m.message = fmt.Sprintf("%d marked unread", len(items))
	}
}

func (m *Model) setSelectionMessage() {
	if len(m.selectedItems) == 0 {
		m.message = "selection cleared"
		return
	}
	m.message = fmt.Sprintf("%d selected", len(m.selectedItems))
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
	return strings.Join(m.previewLines(25, defaultViewWidth), "\n")
}

func (m Model) previewLines(limit, width int) []string {
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
	return renderMarkdownPreview(string(b), m.showFrontmatter, width, limit)
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

func syncCmd(ctx context.Context, a *app.App) tea.Cmd {
	return syncSourcesCmd(ctx, a, []string{""})
}

func syncSourcesCmd(ctx context.Context, a *app.App, sourceIDs []string) tea.Cmd {
	if ctx == nil {
		ctx = context.Background()
	}
	return func() tea.Msg {
		results := make([]feedSync.Result, 0, len(sourceIDs))
		for _, sourceID := range sourceIDs {
			results = append(results, a.Sync(ctx, sourceID))
		}
		return syncMsg{results: results}
	}
}

func (m syncMsg) ok() bool {
	for _, result := range m.results {
		if !result.OK {
			return false
		}
	}
	return true
}

func (m syncMsg) errorText() string {
	var parts []string
	for _, result := range m.results {
		parts = append(parts, result.Errors...)
		for _, source := range result.Sources {
			parts = append(parts, source.Errors...)
		}
	}
	if len(parts) == 0 {
		return "unknown error"
	}
	return strings.Join(parts, "; ")
}

func tickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg{} })
}

func helpView() string {
	m := Model{width: defaultViewWidth, height: defaultViewHeight, help: true}
	return m.renderHelp(defaultViewWidth, defaultViewHeight)
}
