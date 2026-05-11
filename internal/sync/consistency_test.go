package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"feedctl/internal/config"
	"feedctl/internal/content"
	"feedctl/internal/source"
	"feedctl/internal/store"
	feedSync "feedctl/internal/sync"
)

func TestSyncRestoresCurrentMarkdownWhenChangedItemUpdateFails(t *testing.T) {
	item := source.Item{Title: "Tracked", URL: "https://example.com/tracked", GUID: "guid-1", Body: "Original body"}
	runner, st, paths, src := newConsistencyRunner(t, item)
	defer st.Close()
	if err := st.UpsertConfiguredSource(src); err != nil {
		t.Fatal(err)
	}

	first := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !first.OK || first.Sources[0].NewItems != 1 {
		t.Fatalf("first sync=%#v", first)
	}
	items, err := st.ListItems(store.ItemFilter{AllItems: true})
	if err != nil || len(items) != 1 {
		t.Fatalf("items len=%d err=%v", len(items), err)
	}
	currentPath := filepath.Join(paths.ContentDir, items[0].ContentPath)
	original, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := st.Raw().Exec(`CREATE TRIGGER fail_item_update BEFORE UPDATE ON items BEGIN SELECT RAISE(FAIL, 'fail item update'); END`); err != nil {
		t.Fatal(err)
	}
	runner.Adapter = fakeMetricsAdapter{feed: source.Feed{Metadata: source.Metadata{Title: "Feed", URL: src.URL, FeedURL: src.URL, ItemsFound: 1}, Items: []source.Item{{Title: "Tracked", URL: "https://example.com/tracked", GUID: "guid-1", Body: "Changed body"}}}}

	changed := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if changed.OK || changed.Sources[0].Status != "failed" {
		t.Fatalf("changed sync=%#v want failed source", changed)
	}
	after, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Fatalf("current markdown not restored after failed update:\n%s", string(after))
	}
	stored, err := st.GetItem(items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version != 1 || stored.ContentHash != items[0].ContentHash {
		t.Fatalf("item metadata changed after failed update: before=%#v after=%#v", items[0], stored)
	}
}

func TestSyncRemovesNewMarkdownWhenItemInsertFails(t *testing.T) {
	runner, st, paths, src := newConsistencyRunner(t, source.Item{Title: "Orphan", URL: "https://example.com/orphan", GUID: "guid-1", Body: "Body"})
	defer st.Close()

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if res.OK || len(res.Sources) != 1 || res.Sources[0].Status != "failed" {
		t.Fatalf("sync result=%#v want failed source", res)
	}
	files := markdownFiles(t, paths.ContentDir)
	if len(files) != 0 {
		t.Fatalf("markdown files left after failed insert: %v", files)
	}
}

func TestSameSourceSyncIsSerializedWhileDifferentSourcesRemainConcurrent(t *testing.T) {
	root := t.TempDir()
	paths := config.Paths{
		Database:    filepath.Join(root, "feedctl.db"),
		ContentDir:  filepath.Join(root, "content"),
		VersionsDir: filepath.Join(root, "versions"),
		TmpDir:      filepath.Join(root, "tmp"),
	}
	st, err := store.Open(paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	sources := []config.Source{
		{ID: "same", Type: config.SourceTypeRSS, Name: "Same", URL: "https://example.com/same.xml", Enabled: true, Interval: "5m"},
		{ID: "same", Type: config.SourceTypeRSS, Name: "Same", URL: "https://example.com/same.xml", Enabled: true, Interval: "5m"},
		{ID: "other", Type: config.SourceTypeRSS, Name: "Other", URL: "https://example.com/other.xml", Enabled: true, Interval: "5m"},
	}
	for _, src := range []config.Source{sources[0], sources[2]} {
		if err := st.UpsertConfiguredSource(src); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.DefaultConfig(root)
	cfg.Sync.Concurrency = 3
	runner := feedSync.NewRunner(st, paths, cfg)
	adapter := newConcurrencyTrackingAdapter()
	runner.Adapter = adapter
	runner.Metrics = nil

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	res := runner.RunAll(ctx, sources, feedSync.Options{})
	if !res.OK {
		t.Fatalf("sync result=%#v want ok", res)
	}
	if adapter.sameSourceOverlap() {
		t.Fatalf("duplicate sync for the same source overlapped")
	}
	if !adapter.differentSourceOverlap() {
		t.Fatalf("different sources did not run concurrently")
	}
}

func TestSyncStopsWhenContextCancelledAfterFetch(t *testing.T) {
	runner, st, paths, src := newConsistencyRunner(t, source.Item{Title: "Cancelled", URL: "https://example.com/cancelled", GUID: "guid-cancelled", Body: "Body"})
	defer st.Close()
	if err := st.UpsertConfiguredSource(src); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	runner.Adapter = cancelAfterFetchAdapter{cancel: cancel, feed: source.Feed{Metadata: source.Metadata{Title: "Feed", URL: src.URL, FeedURL: src.URL, ItemsFound: 1}, Items: []source.Item{{Title: "Cancelled", URL: "https://example.com/cancelled", GUID: "guid-cancelled", Body: "Body"}}}}

	res := runner.RunAll(ctx, []config.Source{src}, feedSync.Options{})
	if res.OK || len(res.Sources) != 1 || res.Sources[0].Status != "failed" {
		t.Fatalf("sync result=%#v want cancellation failure", res)
	}
	items, err := st.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("cancelled sync persisted items: %#v", items)
	}
	if files := markdownFiles(t, paths.ContentDir); len(files) != 0 {
		t.Fatalf("cancelled sync wrote markdown: %v", files)
	}
}

func TestSyncDoesNotRemoveCommittedMarkdownWhenNewItemInsertLosesRace(t *testing.T) {
	published := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	feedItem := source.Item{Title: "Race", URL: "https://example.com/race", GUID: "guid-race", PublishedAt: &published, Body: "Race body"}
	runner, st, paths, src := newConsistencyRunner(t, feedItem)
	defer st.Close()
	if err := st.UpsertConfiguredSource(src); err != nil {
		t.Fatal(err)
	}

	identity, kind := source.Identity(src.ID, feedItem)
	itemID := "item_" + content.ShortID(src.ID+"\x00"+identity)
	rel := content.RenderPath(runner.Config.Markdown.PathTemplate, content.PathData{SourceID: src.ID, Title: feedItem.Title, ItemID: itemID, Time: published})
	committed := []byte("committed markdown")
	if _, _, err := content.SafeWrite(paths.ContentDir, rel, paths.TmpDir, committed); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateItem(store.Item{
		ID: itemID, SourceID: src.ID, SourceItemID: "different-stale-identity", IdentityKind: kind,
		Title: "Committed", URL: feedItem.URL, PublishedAt: published.Format(time.RFC3339), FetchedAt: published.Format(time.RFC3339),
		LastSeenAt: published.Format(time.RFC3339), ContentPath: rel, ContentHash: "sha256:committed", Version: 1,
	}); err != nil {
		t.Fatal(err)
	}

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if res.OK || len(res.Sources) != 1 || res.Sources[0].Status != "failed" {
		t.Fatalf("sync result=%#v want failed source", res)
	}
	after, err := os.ReadFile(filepath.Join(paths.ContentDir, rel))
	if err != nil {
		t.Fatalf("committed markdown was removed after losing insert: %v", err)
	}
	if string(after) != string(committed) {
		t.Fatalf("committed markdown changed after losing insert:\n%s", string(after))
	}
}

func newConsistencyRunner(t *testing.T, item source.Item) (*feedSync.Runner, *store.DB, config.Paths, config.Source) {
	t.Helper()
	root := t.TempDir()
	paths := config.Paths{
		Database:    filepath.Join(root, "feedctl.db"),
		ContentDir:  filepath.Join(root, "content"),
		VersionsDir: filepath.Join(root, "versions"),
		TmpDir:      filepath.Join(root, "tmp"),
	}
	st, err := store.Open(paths.Database)
	if err != nil {
		t.Fatal(err)
	}
	src := config.Source{ID: "missing-source", Type: config.SourceTypeRSS, Name: "Missing", URL: "https://example.com/feed.xml", Enabled: true, Interval: "5m"}
	cfg := config.DefaultConfig(root)
	cfg.Markdown.Frontmatter = true
	cfg.Markdown.PathTemplate = config.DefaultPathTemplate
	cfg.Sync.Concurrency = 1
	runner := feedSync.NewRunner(st, paths, cfg)
	runner.Adapter = fakeMetricsAdapter{feed: source.Feed{Metadata: source.Metadata{Title: "Feed", URL: src.URL, FeedURL: src.URL, ItemsFound: 1}, Items: []source.Item{item}}}
	runner.Metrics = nil
	return runner, st, paths, src
}

type cancelAfterFetchAdapter struct {
	cancel context.CancelFunc
	feed   source.Feed
}

func (a cancelAfterFetchAdapter) Fetch(context.Context, config.Source) (source.Feed, error) {
	a.cancel()
	return a.feed, nil
}

func (a cancelAfterFetchAdapter) Test(context.Context, config.Source) (source.Metadata, error) {
	return a.feed.Metadata, nil
}

type concurrencyTrackingAdapter struct {
	mu        sync.Mutex
	active    map[string]int
	release   chan struct{}
	released  bool
	same      bool
	different bool
}

func newConcurrencyTrackingAdapter() *concurrencyTrackingAdapter {
	return &concurrencyTrackingAdapter{active: map[string]int{}, release: make(chan struct{})}
}

func (a *concurrencyTrackingAdapter) Fetch(ctx context.Context, src config.Source) (source.Feed, error) {
	a.mu.Lock()
	if a.active[src.ID] > 0 {
		a.same = true
		a.closeReleaseLocked()
	}
	for id, count := range a.active {
		if id != src.ID && count > 0 {
			a.different = true
			a.closeReleaseLocked()
		}
	}
	a.active[src.ID]++
	a.mu.Unlock()

	select {
	case <-a.release:
	case <-time.After(200 * time.Millisecond):
	case <-ctx.Done():
		return source.Feed{}, ctx.Err()
	}

	a.mu.Lock()
	a.active[src.ID]--
	a.mu.Unlock()

	item := source.Item{Title: "Item " + src.ID, URL: "https://example.com/" + src.ID, GUID: "guid-" + src.ID, Body: "Body"}
	return source.Feed{Metadata: source.Metadata{Title: src.Name, URL: src.URL, FeedURL: src.URL, ItemsFound: 1}, Items: []source.Item{item}}, nil
}

func (a *concurrencyTrackingAdapter) Test(ctx context.Context, src config.Source) (source.Metadata, error) {
	feed, err := a.Fetch(ctx, src)
	return feed.Metadata, err
}

func (a *concurrencyTrackingAdapter) sameSourceOverlap() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.same
}

func (a *concurrencyTrackingAdapter) differentSourceOverlap() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.different
}

func (a *concurrencyTrackingAdapter) closeReleaseLocked() {
	if !a.released {
		close(a.release)
		a.released = true
	}
}

func markdownFiles(t *testing.T, root string) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return files
}
