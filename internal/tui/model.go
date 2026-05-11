package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"
)

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
