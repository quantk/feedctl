package tui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/config"
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

func TestTUIManualSyncFailureIsVisible(t *testing.T) {
	m, closeApp := newFailingSyncModel(t)
	defer closeApp()

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})
	m = model.(Model)
	if cmd == nil {
		t.Fatal("manual sync did not return command")
	}
	model, _ = m.Update(cmd())
	m = model.(Model)
	if !strings.Contains(m.message, "sync failed") {
		t.Fatalf("message=%q want visible sync failure", m.message)
	}
	if m.syncing {
		t.Fatal("syncing should be false after failed sync result")
	}
}

func TestTUIPeriodicSyncFailureIsVisible(t *testing.T) {
	m, closeApp := newFailingSyncModel(t)
	defer closeApp()

	model, _ := m.Update(syncSourcesCmd(m.ctx, m.app, []string{"bad"})())
	m = model.(Model)
	if !strings.Contains(m.message, "sync failed") {
		t.Fatalf("message=%q want visible periodic sync failure", m.message)
	}
}

func TestTUIReloadFailureIsVisibleAndKeepsPreviousState(t *testing.T) {
	m := newTUITestModel(t, "Body")
	if len(m.items) != 1 {
		t.Fatalf("items=%d want 1", len(m.items))
	}
	if err := m.app.Close(); err != nil {
		t.Fatal(err)
	}

	m = updateKey(t, m, "r")
	if !strings.Contains(m.message, "reload failed") {
		t.Fatalf("message=%q want reload failure", m.message)
	}
	if len(m.items) != 1 {
		t.Fatalf("reload failure should keep previous items, got %d", len(m.items))
	}
}

func TestTUIItemActionFailureIsVisible(t *testing.T) {
	testutil.IsolatedEnv(t)
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Store.UpsertConfiguredSource(config.Source{ID: "example", Type: config.SourceTypeRSS, Name: "Example", URL: "https://example.com/feed.xml", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := a.Store.CreateItem(store.Item{ID: "no-url", SourceID: "example", SourceItemID: "guid-1", IdentityKind: "guid", Title: "No URL", ContentPath: "example/no-url.md", ContentHash: "sha256:no-url", Version: 1}); err != nil {
		t.Fatal(err)
	}

	m := NewModel(a)
	m = updateKey(t, m, "o")
	if !strings.Contains(m.message, "item has no URL") {
		t.Fatalf("message=%q want item action error", m.message)
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

func TestTUIMultiSelectBatchTogglesReadState(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example",
		testutil.DefaultItem("guid-1", "First", "Body"),
		testutil.DefaultItem("guid-2", "Second", "Body"),
		testutil.DefaultItem("guid-3", "Third", "Body"),
	)
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
	if len(m.items) != 3 {
		t.Fatalf("items=%d", len(m.items))
	}
	firstID := m.items[0].ID
	secondID := m.items[1].ID
	thirdID := m.items[2].ID

	m = updateKey(t, m, "v")
	m = updateKey(t, m, "j")
	if got := len(m.selectedItems); got != 2 {
		t.Fatalf("selected items=%d want 2", got)
	}
	if !strings.Contains(visibleText(m.renderList(80, 8)), "✓") {
		t.Fatalf("selected rows do not show check marker:\n%s", visibleText(m.renderList(80, 8)))
	}

	m = updateKey(t, m, " ")
	if got := len(m.selectedItems); got != 0 {
		t.Fatalf("selected items after batch read=%d want 0", got)
	}
	assertTUIReadState(t, a, firstID, true)
	assertTUIReadState(t, a, secondID, true)
	assertTUIReadState(t, a, thirdID, false)

	m = updateKey(t, m, "g")
	m = updateKey(t, m, "v")
	m = updateKey(t, m, "j")
	m = updateKey(t, m, "u")
	assertTUIReadState(t, a, firstID, false)
	assertTUIReadState(t, a, secondID, false)
	assertTUIReadState(t, a, thirdID, false)

	m = updateKey(t, m, "g")
	m = updateKey(t, m, "v")
	m = updateKey(t, m, "j")
	m = updateKey(t, m, " ")
	assertTUIReadState(t, a, firstID, true)
	assertTUIReadState(t, a, secondID, true)
	assertTUIReadState(t, a, thirdID, false)

	m = updateKey(t, m, "g")
	m = updateKey(t, m, "v")
	m = updateKey(t, m, "j")
	m = updateKey(t, m, " ")
	assertTUIReadState(t, a, firstID, false)
	assertTUIReadState(t, a, secondID, false)
	assertTUIReadState(t, a, thirdID, false)
}

func TestTUIVisualSelectionExtendsWithMovementAndEscCancels(t *testing.T) {
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example",
		testutil.DefaultItem("guid-1", "First", "Body"),
		testutil.DefaultItem("guid-2", "Second", "Body"),
		testutil.DefaultItem("guid-3", "Third", "Body"),
	)
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
	m = updateKey(t, m, "j")
	anchorID := m.items[m.cursor].ID
	m = updateKey(t, m, "v")
	if !m.selecting || m.selectionAnchor != anchorID || len(m.selectedItems) != 1 {
		t.Fatalf("selection not started: selecting=%v anchor=%q selected=%v", m.selecting, m.selectionAnchor, m.selectedItems)
	}

	m = updateKey(t, m, "j")
	if got := len(m.selectedItems); got != 2 {
		t.Fatalf("selected after moving down=%d want 2", got)
	}
	if !m.itemSelected(anchorID) || !m.itemSelected(m.items[m.cursor].ID) {
		t.Fatalf("selection does not include anchor/current: anchor=%s cursor=%s selected=%v", anchorID, m.items[m.cursor].ID, m.selectedItems)
	}

	m = updateKey(t, m, "k")
	if got := len(m.selectedItems); got != 1 {
		t.Fatalf("selected after moving back to anchor=%d want 1", got)
	}
	if !m.itemSelected(anchorID) {
		t.Fatalf("anchor not selected after moving back: selected=%v", m.selectedItems)
	}

	m = updateKey(t, m, "k")
	if got := len(m.selectedItems); got != 2 {
		t.Fatalf("selected after moving up=%d want 2", got)
	}
	if !m.itemSelected(anchorID) || !m.itemSelected(m.items[m.cursor].ID) {
		t.Fatalf("upward selection missing anchor/current: anchor=%s cursor=%s selected=%v", anchorID, m.items[m.cursor].ID, m.selectedItems)
	}

	m = updateKey(t, m, "esc")
	if m.selecting || m.selectionAnchor != "" || len(m.selectedItems) != 0 {
		t.Fatalf("esc did not cancel selection: selecting=%v anchor=%q selected=%v", m.selecting, m.selectionAnchor, m.selectedItems)
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

func TestTUIPreviewHidesFrontmatterByDefault(t *testing.T) {
	m := newTUITestModel(t, "Intro body")
	preview := visibleText(m.renderPreviewPane(80, 14))

	for _, hidden := range []string{"---", "source_id:", "content_hash:", "tags:"} {
		if strings.Contains(preview, hidden) {
			t.Fatalf("preview should hide frontmatter field %q by default:\n%s", hidden, preview)
		}
	}
	if !strings.Contains(preview, "Intro body") {
		t.Fatalf("preview missing body:\n%s", preview)
	}
}

func TestTUIFrontmatterToggleShowsFrontmatter(t *testing.T) {
	m := newTUITestModel(t, "Intro body")
	if m.showFrontmatter {
		t.Fatal("frontmatter should be hidden by default")
	}

	m = updateKey(t, m, "m")
	if !m.showFrontmatter {
		t.Fatal("m did not show frontmatter")
	}
	preview := visibleText(m.renderPreviewPane(80, 18))
	if !strings.Contains(preview, "source_id:") || !strings.Contains(preview, "content_hash:") {
		t.Fatalf("frontmatter toggle did not reveal metadata:\n%s", preview)
	}

	m = updateKey(t, m, "m")
	if m.showFrontmatter {
		t.Fatal("second m did not hide frontmatter")
	}
	preview = visibleText(m.renderPreviewPane(80, 18))
	if strings.Contains(preview, "source_id:") {
		t.Fatalf("frontmatter stayed visible after second toggle:\n%s", preview)
	}
}

func TestTUIPreviewRendersMarkdown(t *testing.T) {
	m := newTUITestModel(t, "## Section\n\nA **bold** [link](https://example.com).\n\n- one")
	preview := visibleText(m.renderPreviewPane(80, 18))

	for _, raw := range []string{"## Section", "**bold**", "[link](https://example.com)"} {
		if strings.Contains(preview, raw) {
			t.Fatalf("preview still contains raw markdown %q:\n%s", raw, preview)
		}
	}
	for _, want := range []string{"Section", "bold", "link", "one"} {
		if !strings.Contains(preview, want) {
			t.Fatalf("preview missing rendered markdown text %q:\n%s", want, preview)
		}
	}
}

func TestTUIErrorMessageUsesErrorStyle(t *testing.T) {
	m := Model{message: "sync failed: boom"}
	rendered := m.renderStatus(120)
	if !strings.Contains(rendered, errorStyle.Render(m.message)) {
		t.Fatalf("status did not render error message with error style:\n%s", rendered)
	}
}

func TestTUIWindowSizeAndFullHeightRender(t *testing.T) {
	m := NewModel(nil)
	m = updateWindowSize(t, m, 72, 16)
	if m.width != 72 || m.height != 16 {
		t.Fatalf("size=%dx%d", m.width, m.height)
	}
	view := m.View()
	assertTUIViewFrame(t, view, 72, 16)
	if !strings.Contains(view, "feedctl") {
		t.Fatal("view does not include header")
	}
}

func TestTUIScrollKeepsFrameWhenItemTitleContainsLineBreak(t *testing.T) {
	testutil.IsolatedEnv(t)
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if err := a.Store.UpsertConfiguredSource(config.Source{ID: "example", Type: config.SourceTypeRSS, Name: "Example", URL: "https://example.com/feed.xml", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	for i := range 12 {
		title := "Item title"
		if i == 8 {
			title = "Item with\nline break"
		}
		if err := a.Store.CreateItem(store.Item{ID: fmt.Sprintf("item-%02d", i), SourceID: "example", SourceItemID: fmt.Sprintf("guid-%02d", i), IdentityKind: "guid", Title: title, URL: fmt.Sprintf("https://example.com/%02d", i), ContentPath: fmt.Sprintf("example/%02d.md", i), ContentHash: fmt.Sprintf("sha256:%02d", i), Version: 1, PublishedAt: fmt.Sprintf("2026-05-11T00:%02d:00Z", i)}); err != nil {
			t.Fatal(err)
		}
	}

	m := updateWindowSize(t, NewModel(a), 72, 10)
	for i, item := range m.items {
		if strings.Contains(item.Title, "line break") {
			m.cursor = i
			break
		}
	}
	if !strings.Contains(m.items[m.cursor].Title, "\n") {
		t.Fatalf("test setup did not select multiline item: cursor=%d title=%q", m.cursor, m.items[m.cursor].Title)
	}

	view := m.View()
	assertTUIViewFrame(t, view, 72, 10)
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

func assertTUIReadState(t *testing.T, a *app.App, itemID string, wantRead bool) {
	t.Helper()
	item, err := a.Item(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if gotRead := item.ReadAt != ""; gotRead != wantRead {
		t.Fatalf("item %s read=%v want %v", itemID, gotRead, wantRead)
	}
}

func assertTUIViewFrame(t *testing.T, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if got := len(lines); got != height {
		t.Fatalf("rendered lines=%d want=%d\n%s", got, height, visibleText(view))
	}
	for i, line := range lines {
		if got := lipgloss.Width(visibleText(line)); got != width {
			t.Fatalf("line %d width=%d want=%d: %q\n%s", i, got, width, visibleText(line), visibleText(view))
		}
	}
	if !strings.Contains(view, "feedctl") {
		t.Fatalf("view does not include header:\n%s", visibleText(view))
	}
	lastLine := visibleText(lines[len(lines)-1])
	if !strings.Contains(lastLine, "sync") && !strings.Contains(lastLine, "unread") {
		t.Fatalf("view does not include status bar:\n%s", visibleText(view))
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

func newFailingSyncModel(t *testing.T) (Model, func()) {
	t.Helper()
	configDir, _ := testutil.IsolatedEnv(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not a feed"))
	}))
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "bad.toml"), config.Source{ID: "bad", Type: config.SourceTypeRSS, Name: "Bad", URL: server.URL, Enabled: true, Interval: "5m"}); err != nil {
		server.Close()
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	return NewModel(a), func() {
		_ = a.Close()
		server.Close()
	}
}

func newTUITestModel(t *testing.T, body string) Model {
	t.Helper()
	testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "Markdown title", body))
	server := testutil.FeedServer(t, &feed)
	t.Cleanup(server.Close)
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	res := a.Sync(context.Background(), "")
	if !res.OK {
		t.Fatalf("sync: %#v", res)
	}
	return NewModel(a)
}

var ansiEscapeRE = regexp.MustCompile("\x1b\\[[0-?]*[ -/]*[@-~]")

func visibleText(value string) string {
	return ansiEscapeRE.ReplaceAllString(value, "")
}

func updateWindowSize(t *testing.T, m Model, width, height int) Model {
	t.Helper()
	model, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return model.(Model)
}
