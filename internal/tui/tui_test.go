package tui

import (
	"context"
	"strings"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/metrics"
	"feedctl/internal/store"
	"feedctl/internal/testutil"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestTUIKeybindingsAndItemActions(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"), testutil.DefaultItem("guid-2", "Second", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := NewModel(a)
	if len(m.items) != 2 {
		t.Fatalf("items=%d", len(m.items))
	}
	m = updateKey(t, m, "j")
	if m.cursor != 1 {
		t.Fatalf("cursor after j=%d", m.cursor)
	}
	m = updateKey(t, m, "g")
	if m.cursor != 0 {
		t.Fatalf("cursor after g=%d", m.cursor)
	}
	m = updateKey(t, m, "?")
	if !m.help {
		t.Fatal("help not opened")
	}
	m = updateKey(t, m, "esc")
	if m.help {
		t.Fatal("help not closed")
	}
	firstID := m.items[m.cursor].ID
	m = updateKey(t, m, " ")
	item, err := a.Item(firstID)
	if err != nil {
		t.Fatal(err)
	}
	if item.ReadAt == "" {
		t.Fatal("space did not mark read")
	}
	m = updateKey(t, m, "s")
	item, _ = a.Item(firstID)
	if !item.Starred {
		t.Fatal("s did not star")
	}
	m = updateKey(t, m, "a")
	item, _ = a.Item(firstID)
	if item.ArchivedAt == "" {
		t.Fatal("a did not archive")
	}
	m = updateKey(t, m, "2")
	if m.section != 1 {
		t.Fatalf("section=%d", m.section)
	}
	m = updateKey(t, m, "A")
	if !m.showRemoved {
		t.Fatal("A did not toggle removed visibility")
	}
}

func TestTUIEnterMarksItemRead(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"), testutil.DefaultItem("guid-2", "Second", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := NewModel(a)
	firstID := m.items[0].ID
	secondID := m.items[1].ID
	m = updateKey(t, m, "enter")
	if !m.reader {
		t.Fatal("enter did not open reader")
	}
	first, err := a.Item(firstID)
	if err != nil {
		t.Fatal(err)
	}
	if first.ReadAt == "" {
		t.Fatal("enter did not mark selected item read")
	}
	if m.items[0].ReadAt == "" {
		t.Fatal("enter did not update local read marker")
	}

	m = updateKey(t, m, "j")
	m = updateKey(t, m, "l")
	second, err := a.Item(secondID)
	if err != nil {
		t.Fatal(err)
	}
	if second.ReadAt != "" {
		t.Fatal("l should open without marking read")
	}
}

func TestTUISearchAcceptsUnicodeRunes(t *testing.T) {
	m := NewModel(nil)
	m = updateKey(t, m, "/")
	m = updateKey(t, m, "п")
	m = updateKey(t, m, "р")
	m = updateKey(t, m, "и")
	if m.search != "при" {
		t.Fatalf("search=%q want %q", m.search, "при")
	}
	m = updateKey(t, m, "backspace")
	if m.search != "пр" {
		t.Fatalf("search after backspace=%q want %q", m.search, "пр")
	}
}

func TestTUILiveFilterNarrowsItems(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "Привет мир", "Body"), testutil.DefaultItem("guid-2", "Other", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := NewModel(a)
	if len(m.items) != 2 {
		t.Fatalf("items=%d", len(m.items))
	}
	m = updateKey(t, m, "f")
	m = updateKey(t, m, "п")
	m = updateKey(t, m, "р")
	if m.filter != "пр" {
		t.Fatalf("filter=%q want %q", m.filter, "пр")
	}
	if len(m.items) != 1 || m.items[0].Title != "Привет мир" {
		t.Fatalf("filtered items=%#v", m.items)
	}
	m = updateKey(t, m, "enter")
	m = updateKey(t, m, "F")
	if m.filter != "" || len(m.items) != 2 {
		t.Fatalf("clear filter failed: filter=%q items=%d", m.filter, len(m.items))
	}
}

func TestTUISearchDoesNotFilterItems(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "Привет мир", "Body"), testutil.DefaultItem("guid-2", "Other", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := NewModel(a)
	m = updateKey(t, m, "/")
	m = updateKey(t, m, "п")
	if m.search != "п" {
		t.Fatalf("search=%q want %q", m.search, "п")
	}
	if len(m.items) != 2 {
		t.Fatalf("search filtered items unexpectedly: %d", len(m.items))
	}
}

func TestFilterItemsMatchesUnicodeCaseInsensitive(t *testing.T) {
	items := []store.Item{
		{Title: "Привет мир", SourceID: "ru"},
		{Title: "Other", SourceID: "en"},
	}
	filtered := filterItems(items, "при")
	if len(filtered) != 1 || filtered[0].Title != "Привет мир" {
		t.Fatalf("filtered=%#v", filtered)
	}
}

func TestTUIItemRowRendersMetrics(t *testing.T) {
	score := 12
	comments := 4
	zero := 0
	m := Model{}

	row := m.renderItemRow(store.Item{Title: "Metric title", SourceID: "habr-ai", Metrics: &metrics.ItemMetrics{Score: &score, CommentsCount: &comments}}, false, 80)
	if !strings.Contains(row, "+12") || !strings.Contains(row, "4c") {
		t.Fatalf("row missing compact metrics: %q", row)
	}

	row = m.renderItemRow(store.Item{Title: "Plain title", SourceID: "src"}, false, 80)
	if strings.Contains(row, "+0") || strings.Contains(row, "0c") {
		t.Fatalf("row contains placeholder metrics: %q", row)
	}

	row = m.renderItemRow(store.Item{Title: "Zero title", SourceID: "src", Metrics: &metrics.ItemMetrics{Score: &zero}}, false, 80)
	if !strings.Contains(row, "0") {
		t.Fatalf("row missing known zero score: %q", row)
	}
}

func TestTUIPreviewRendersExpandedMetrics(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "Metric title", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}
	items, err := a.Items(store.ItemFilter{})
	if err != nil {
		t.Fatal(err)
	}
	score := 12
	comments := 4
	votes := 5
	reads := 23
	if err := a.Store.UpsertItemMetrics(items[0].ID, metrics.ItemMetrics{Provider: "habr", Score: &score, CommentsCount: &comments, VotesCount: &votes, ReadingCount: &reads, FetchedAt: "2026-05-11T15:00:00Z"}); err != nil {
		t.Fatal(err)
	}

	m := NewModel(a)
	preview := m.renderPreviewPane(80, 14)
	for _, want := range []string{"Score: +12", "Comments: 4", "Votes: 5", "Reads: 23"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing %q:\n%s", want, preview)
		}
	}
}

func TestTUIWindowSizeAndFullHeightRender(t *testing.T) {
	m := NewModel(nil)
	m = updateWindowSize(t, m, 72, 16)
	if m.width != 72 || m.height != 16 {
		t.Fatalf("size=%dx%d", m.width, m.height)
	}
	view := m.View()
	lines := strings.Split(view, "\n")
	if got := len(lines); got != 16 {
		t.Fatalf("rendered lines=%d want=16\n%s", got, view)
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != 72 {
			t.Fatalf("line %d width=%d want=72: %q", i, got, line)
		}
	}
	if !strings.Contains(view, "feedctl") {
		t.Fatal("view does not include header")
	}
}

func TestTUISelectionMarkerUsesVerticalBar(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}

	m := updateWindowSize(t, NewModel(a), 80, 18)
	view := m.View()
	if strings.Contains(view, ">") {
		t.Fatalf("item view still contains arrow marker: %q", view)
	}
	if !strings.Contains(view, selectedMarker) {
		t.Fatalf("item view missing selected marker %q: %q", selectedMarker, view)
	}

	m.section = 3
	m.cursor = 0
	m.reload()
	view = m.View()
	if strings.Contains(view, ">") {
		t.Fatalf("source view still contains arrow marker: %q", view)
	}
	if !strings.Contains(view, selectedMarker) {
		t.Fatalf("source view missing selected marker %q: %q", selectedMarker, view)
	}
}

func updateKey(t *testing.T, m Model, key string) Model {
	t.Helper()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	if key == " " {
		msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	}
	if key == "esc" {
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	}
	if key == "enter" {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	}
	if key == "backspace" {
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	}
	model, _ := m.Update(msg)
	return model.(Model)
}

func updateWindowSize(t *testing.T, m Model, width, height int) Model {
	t.Helper()
	model, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return model.(Model)
}
