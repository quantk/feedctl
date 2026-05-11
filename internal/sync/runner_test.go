package sync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"
	"feedctl/internal/testutil"
)

func TestRSSSyncConcurrentSourcesRaceRegression(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	feedOne := testutil.RSSFeed("One", testutil.DefaultItem("guid-1", "First", "Body"))
	feedTwo := testutil.RSSFeed("Two", testutil.DefaultItem("guid-2", "Second", "Body"))
	release := make(chan struct{})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 2 {
			close(release)
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml")
		switch r.URL.Path {
		case "/one":
			_, _ = w.Write([]byte(feedOne))
		case "/two":
			_, _ = w.Write([]byte(feedTwo))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "one.toml"), config.Source{ID: "one", Type: config.SourceTypeRSS, Name: "One", URL: server.URL + "/one", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "two.toml"), config.Source{ID: "two", Type: config.SourceTypeRSS, Name: "Two", URL: server.URL + "/two", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res := a.Sync(ctx, "")
	if !res.OK || len(res.Sources) != 2 {
		t.Fatalf("concurrent rss sync failed: %#v", res)
	}
	for _, src := range res.Sources {
		if src.Status != "ok" || src.NewItems != 1 {
			t.Fatalf("unexpected source result: %#v", res.Sources)
		}
	}
}

func TestTelegramSyncCreatesMarkdownAndDoesNotDuplicate(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	page := testutil.TelegramWebPage("LLM под капотом", "llm_under_hood", []testutil.TelegramPost{
		{ID: 831, HTML: `<b>AI Ops</b><br/>Readable body`, Datetime: "2026-05-11T15:38:00+00:00"},
	}, "")
	server := testutil.TelegramServer(t, "llm_under_hood", map[string]string{"/s/llm_under_hood": page})
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "tg-llm.toml"), config.Source{ID: "tg-llm", Type: config.SourceTypeTelegram, Name: "LLM", URL: server.URL + "/s/llm_under_hood", Enabled: true, Interval: "5m", Tags: []string{"telegram"}}); err != nil {
		t.Fatal(err)
	}

	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "tg-llm")
	if !res.OK || len(res.Sources) != 1 || res.Sources[0].NewItems != 1 {
		t.Fatalf("bad telegram sync: %#v", res)
	}
	items, err := a.Items(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items len=%d", len(items))
	}
	item := items[0]
	if item.SourceItemID != "llm_under_hood/831" || item.IdentityKind != "guid" || item.SourceID != "tg-llm" {
		t.Fatalf("bad item identity: %#v", item)
	}
	mdPath, err := a.MarkdownPath(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "source_type: \"telegram\"") || !strings.Contains(string(b), "Readable body") {
		t.Fatalf("bad telegram markdown:\n%s", string(b))
	}

	again := a.Sync(context.Background(), "tg-llm")
	if !again.OK || again.Sources[0].UnchangedItems != 1 {
		t.Fatalf("bad repeated telegram sync: %#v", again)
	}
	items, err = a.Items(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("duplicate telegram item count=%d items=%#v", len(items), items)
	}
}

func TestTelegramSyncEditedPostCreatesVersion(t *testing.T) {
	configDir, dataRoot := testutil.IsolatedEnv(t)
	page := testutil.TelegramWebPage("LLM", "llm_under_hood", []testutil.TelegramPost{
		{ID: 831, HTML: `Original body`, Datetime: "2026-05-11T15:38:00+00:00"},
	}, "")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/s/llm_under_hood" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(page))
	}))
	defer server.Close()
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "tg-llm.toml"), config.Source{ID: "tg-llm", Type: config.SourceTypeTelegram, Name: "LLM", URL: server.URL + "/s/llm_under_hood", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	first := a.Sync(context.Background(), "tg-llm")
	if !first.OK || first.Sources[0].NewItems != 1 {
		t.Fatalf("first sync: %#v", first)
	}
	items, err := a.Items(store.ItemFilter{AllItems: true})
	if err != nil || len(items) != 1 {
		t.Fatalf("items len=%d err=%v", len(items), err)
	}
	item := items[0]

	page = testutil.TelegramWebPage("LLM", "llm_under_hood", []testutil.TelegramPost{
		{ID: 831, HTML: `Edited body`, Datetime: "2026-05-11T15:38:00+00:00"},
	}, "")
	changed := a.Sync(context.Background(), "tg-llm")
	if !changed.OK || changed.Sources[0].UpdatedItems != 1 {
		t.Fatalf("changed sync: %#v", changed)
	}
	versionPath := filepath.Join(dataRoot, "versions", item.ID, "v1.md")
	if _, err := os.Stat(versionPath); err != nil {
		t.Fatalf("expected version file: %v", err)
	}
}

func TestTelegramSyncFailureIsIsolated(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	bad := testutil.TelegramServer(t, "bad_channel", map[string]string{})
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "bad-tg.toml"), config.Source{ID: "bad-tg", Type: config.SourceTypeTelegram, Name: "Bad", URL: bad.URL + "/s/bad_channel", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	feed := testutil.RSSFeed("Good", testutil.DefaultItem("guid-1", "Good", "Body"))
	good := testutil.FeedServer(t, &feed)
	defer good.Close()
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "good-rss.toml"), config.Source{ID: "good-rss", Type: "rss", Name: "Good", URL: good.URL, Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if res.OK || len(res.Sources) != 2 {
		t.Fatalf("expected mixed sync result: %#v", res)
	}
	var goodOK, badFailed bool
	for _, src := range res.Sources {
		switch src.SourceID {
		case "good-rss":
			goodOK = src.Status == "ok" && src.NewItems == 1
		case "bad-tg":
			badFailed = src.Status == "failed" && len(src.Errors) > 0
		}
	}
	if !goodOK || !badFailed {
		t.Fatalf("failure not isolated: %#v", res.Sources)
	}
}

func TestRSSSyncParseFailureAndDuplicateIdentity(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("not a feed")) }))
	defer bad.Close()
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "bad.toml"), config.Source{ID: "bad", Type: "rss", Name: "Bad", URL: bad.URL, Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	res := a.Sync(context.Background(), "bad")
	if res.OK || len(res.Sources) != 1 || res.Sources[0].Status != "failed" {
		t.Fatalf("expected parse failure: %#v", res)
	}
	_ = a.Close()

	feed := testutil.RSSFeed("Dup",
		testutil.DefaultItem("same-guid", "First", "Body 1"),
		testutil.DefaultItem("same-guid", "First duplicate", "Body 2"),
	)
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "dup.toml"), config.Source{ID: "dup", Type: "rss", Name: "Dup", URL: server.URL, Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err = app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res = a.Sync(context.Background(), "dup")
	if !res.OK {
		t.Fatalf("sync duplicate feed: %#v", res)
	}
	items, err := a.Items(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range items {
		if item.SourceID == "dup" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one stored item for duplicate identity, got %d items=%#v", count, items)
	}
}
