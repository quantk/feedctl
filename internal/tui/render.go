package tui

import (
	"fmt"
	"os"
	"strings"

	"feedctl/internal/app"
	"feedctl/internal/metrics"

	"github.com/charmbracelet/lipgloss"
	xansi "github.com/charmbracelet/x/ansi"
)

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

func (m Model) renderItemRow(item app.Item, selected bool, width int, multiSelected ...bool) string {
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

func (m Model) renderSourceRow(src app.Source, selected bool, width int) string {
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
		mutedStyle.Render("disk:" + app.HumanBytes(m.status.Storage.Total())),
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
	text = inlineText(text)
	visible := lipgloss.Width(text)
	if visible > width {
		text = xansi.Truncate(text, width, "")
		visible = lipgloss.Width(text)
	}
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
	text = inlineText(text)
	if xansi.StringWidth(text) <= width {
		return text
	}
	if width == 1 {
		return "…"
	}
	return xansi.Truncate(text, width, "…")
}

func inlineText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", " ")
	text = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(text)
	return strings.Map(func(r rune) rune {
		if r == '\x1b' {
			return r
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, text)
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
	return fmt.Sprintf("●%d unread | src:%d | removed:%d | disk:%s | sync %s | %s %s", m.status.UnreadCount, m.status.SourceCount, m.status.RemovedSourceCount, app.HumanBytes(m.status.Storage.Total()), m.currentSyncStatus(), m.status.LatestSyncAt, m.message)
}
