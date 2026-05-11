package app_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"
	"feedctl/internal/testutil"
)

func TestAddRSSSyncMarkdownStateStorageAndVersions(t *testing.T) {
	_, dataRoot := testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First Item", "<p>Hello</p>"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()

	res, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example", Tags: []string{"tech"}})
	if err != nil {
		t.Fatalf("add rss: %v", err)
	}
	if res.SourceID != "example" || res.ItemsFound != 1 {
		t.Fatalf("bad add result: %#v", res)
	}

	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	syncRes := a.Sync(context.Background(), "")
	if !syncRes.OK || len(syncRes.Sources) != 1 || syncRes.Sources[0].NewItems != 1 {
		t.Fatalf("bad sync result: %#v", syncRes)
	}
	items, err := a.Items(store.ItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("items len=%d err=%v", len(items), err)
	}
	item := items[0]
	mdPath, err := a.MarkdownPath(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(string(b), []string{"id: \"" + item.ID + "\"", "source_id: \"example\"", "# First Item", "Hello"}) {
		t.Fatalf("markdown content unexpected:\n%s", string(b))
	}
	if err := a.OpenItem(context.Background(), item.ID); err != nil {
		t.Fatalf("open item: %v", err)
	}
	if err := a.OpenMarkdown(context.Background(), item.ID); err != nil {
		t.Fatalf("open markdown: %v", err)
	}
	firstInfo, err := os.Stat(mdPath)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	again := a.Sync(context.Background(), "")
	if !again.OK || again.Sources[0].UnchangedItems != 1 {
		t.Fatalf("bad unchanged sync: %#v", again)
	}
	secondInfo, _ := os.Stat(mdPath)
	if !secondInfo.ModTime().Equal(firstInfo.ModTime()) {
		t.Fatalf("unchanged item rewrote markdown")
	}

	feed = testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First Item", "<p>Hello changed</p>"))
	changed := a.Sync(context.Background(), "")
	if !changed.OK || changed.Sources[0].UpdatedItems != 1 {
		t.Fatalf("bad changed sync: %#v", changed)
	}
	versionsRoot := filepath.Join(dataRoot, "versions")
	versionPath := filepath.Join(versionsRoot, item.ID, "v1.md")
	if _, err := os.Stat(versionPath); err != nil {
		t.Fatalf("expected version file: %v", err)
	}
	storage, err := a.Storage()
	if err != nil {
		t.Fatal(err)
	}
	if storage.ItemsCount != 1 || storage.CurrentMarkdownBytes == 0 || storage.VersionsBytes == 0 {
		t.Fatalf("bad storage: %#v", storage)
	}
	if err := a.SetRead(item.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := a.ToggleStarred(item.ID); err != nil {
		t.Fatal(err)
	}
	if err := a.Archive(item.ID); err != nil {
		t.Fatal(err)
	}
	_ = a.Close()

	a, err = app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	persisted, err := a.Item(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ReadAt == "" || !persisted.Starred || persisted.ArchivedAt == "" {
		t.Fatalf("state did not persist: %#v", persisted)
	}
}

func TestSourceLifecycleRemoveAndReappear(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
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
	if _, err := a.RemoveSource("example", false); err != nil {
		t.Fatal(err)
	}
	_ = a.Close()

	a, err = app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s, err := a.Source("example")
	if err != nil {
		t.Fatal(err)
	}
	if s.Lifecycle != "removed" {
		t.Fatalf("lifecycle=%s", s.Lifecycle)
	}
	_ = a.Close()

	src := config.Source{ID: "example", Type: "rss", Name: "Example", URL: server.URL, Enabled: false, Interval: "5m"}
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "example.toml"), src); err != nil {
		t.Fatal(err)
	}
	a, err = app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	s, err = a.Source("example")
	if err != nil {
		t.Fatal(err)
	}
	if s.Lifecycle != "disabled" {
		t.Fatalf("reappeared lifecycle=%s", s.Lifecycle)
	}
}

func containsAll(value string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
