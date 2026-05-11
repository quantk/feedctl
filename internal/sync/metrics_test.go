package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"feedctl/internal/config"
	"feedctl/internal/metrics"
	"feedctl/internal/source"
	"feedctl/internal/store"
	feedSync "feedctl/internal/sync"
)

func TestSyncStoresMetricsForNewItemWithoutMarkdownFrontmatter(t *testing.T) {
	runner, st, paths, src, provider := newMetricsRunner(t, source.Item{
		Title:        "Habr Item",
		URL:          "https://habr.com/ru/articles/1033808/",
		CanonicalURL: "https://habr.com/ru/articles/1033808/",
		GUID:         "guid-1",
		Body:         "Body",
	})
	defer st.Close()
	score := 12
	comments := 4
	provider.metrics = metrics.ItemMetrics{Provider: "habr", Score: &score, CommentsCount: &comments, FetchedAt: "2026-05-11T15:00:00Z"}

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].NewItems != 1 {
		t.Fatalf("sync result=%#v", res)
	}
	items, err := st.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	assertMetricValue(t, "score", items[0].Metrics.Score, 12)
	assertMetricValue(t, "comments", items[0].Metrics.CommentsCount, 4)

	markdown, err := os.ReadFile(filepath.Join(paths.ContentDir, items[0].ContentPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"score:", "comments_count:", "votes_count:", "metrics"} {
		if stringContains(string(markdown), forbidden) {
			t.Fatalf("markdown contains volatile metric field %q:\n%s", forbidden, string(markdown))
		}
	}
}

func TestSyncUpdatesMetricsForUnchangedItemWithoutRewritingMarkdown(t *testing.T) {
	feedItem := source.Item{Title: "Habr Item", URL: "https://habr.com/ru/articles/1033808/", GUID: "guid-1", Body: "Body"}
	runner, st, paths, src, provider := newMetricsRunner(t, feedItem)
	defer st.Close()
	firstScore := 1
	provider.metrics = metrics.ItemMetrics{Provider: "habr", Score: &firstScore, FetchedAt: "2026-05-11T15:00:00Z"}

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].NewItems != 1 {
		t.Fatalf("first sync result=%#v", res)
	}
	items, err := st.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	item := items[0]
	path := filepath.Join(paths.ContentDir, item.ContentPath)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(20 * time.Millisecond)
	secondScore := 9
	provider.metrics = metrics.ItemMetrics{Provider: "habr", Score: &secondScore, FetchedAt: "2026-05-11T15:05:00Z"}
	res = runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].UnchangedItems != 1 {
		t.Fatalf("second sync result=%#v", res)
	}
	updated, err := st.GetItem(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Version != item.Version || updated.ContentHash != item.ContentHash {
		t.Fatalf("content changed unexpectedly: before=%#v after=%#v", item, updated)
	}
	assertMetricValue(t, "score", updated.Metrics.Score, 9)
	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !afterInfo.ModTime().Equal(info.ModTime()) {
		t.Fatal("markdown was rewritten when only metrics changed")
	}
}

func TestSyncMetricsProviderFailureDoesNotFailSync(t *testing.T) {
	runner, st, _, src, provider := newMetricsRunner(t, source.Item{Title: "Habr Item", URL: "https://habr.com/ru/articles/1033808/", GUID: "guid-1", Body: "Body"})
	defer st.Close()
	provider.err = errors.New("metrics unavailable")

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].Status != "ok" || res.Sources[0].NewItems != 1 {
		t.Fatalf("sync result=%#v", res)
	}
	item := onlyMetricsItem(t, st)
	if item.Metrics != nil {
		t.Fatalf("metrics=%#v want nil", item.Metrics)
	}

	res = runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].Status != "ok" || res.Sources[0].UnchangedItems != 1 {
		t.Fatalf("second sync result=%#v", res)
	}
}

func TestSyncSkipsMetricsFetchWhenNoProviderMatches(t *testing.T) {
	runner, st, _, src, provider := newMetricsRunner(t, source.Item{Title: "Other Item", URL: "https://example.com/item", GUID: "guid-1", Body: "Body"})
	defer st.Close()
	provider.match = false

	res := runner.RunAll(context.Background(), []config.Source{src}, feedSync.Options{})
	if !res.OK || res.Sources[0].NewItems != 1 {
		t.Fatalf("sync result=%#v", res)
	}
	if provider.fetchCalls != 0 {
		t.Fatalf("fetchCalls=%d want 0", provider.fetchCalls)
	}
	item := onlyMetricsItem(t, st)
	if item.Metrics != nil {
		t.Fatalf("metrics=%#v want nil", item.Metrics)
	}
}

type fakeMetricsAdapter struct{ feed source.Feed }

func (a fakeMetricsAdapter) Fetch(context.Context, config.Source) (source.Feed, error) {
	return a.feed, nil
}
func (a fakeMetricsAdapter) Test(context.Context, config.Source) (source.Metadata, error) {
	return a.feed.Metadata, nil
}

type fakeMetricsProvider struct {
	match      bool
	metrics    metrics.ItemMetrics
	err        error
	fetchCalls int
}

func (p *fakeMetricsProvider) Name() string { return "fake" }
func (p *fakeMetricsProvider) Match(metrics.Candidate) bool {
	return p.match
}
func (p *fakeMetricsProvider) Fetch(context.Context, metrics.Candidate) (metrics.ItemMetrics, error) {
	p.fetchCalls++
	if p.err != nil {
		return metrics.ItemMetrics{}, p.err
	}
	return p.metrics, nil
}

func newMetricsRunner(t *testing.T, item source.Item) (*feedSync.Runner, *store.DB, config.Paths, config.Source, *fakeMetricsProvider) {
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
	src := config.Source{ID: "habr-ai", Type: "rss", Name: "Habr", URL: "https://habr.com/rss", Enabled: true, Interval: "5m"}
	if err := st.UpsertConfiguredSource(src); err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig(root)
	cfg.Markdown.Frontmatter = true
	cfg.Markdown.PathTemplate = config.DefaultPathTemplate
	cfg.Sync.Concurrency = 1
	provider := &fakeMetricsProvider{match: true}
	runner := feedSync.NewRunner(st, paths, cfg)
	runner.Adapter = fakeMetricsAdapter{feed: source.Feed{Metadata: source.Metadata{Title: "Feed", URL: src.URL, FeedURL: src.URL, ItemsFound: 1}, Items: []source.Item{item}}}
	runner.Metrics = &metrics.Enricher{Providers: []metrics.Provider{provider}}
	return runner, st, paths, src, provider
}

func onlyMetricsItem(t *testing.T, st *store.DB) store.Item {
	t.Helper()
	items, err := st.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	return items[0]
}

func assertMetricValue(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s=%d want %d", name, *got, want)
	}
}

func stringContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && contains(s, substr))
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
