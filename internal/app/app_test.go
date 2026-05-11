package app_test

import (
	"context"
	"errors"
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

func TestAppOpenDoesNotImplicitlyReconcileSources(t *testing.T) {
	configDir, dataRoot := testutil.IsolatedEnv(t)
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "example.toml"), config.Source{ID: "example", Type: config.SourceTypeRSS, Name: "Example", URL: "https://example.com/feed.xml", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := store.Open(filepath.Join(dataRoot, "feedctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	sources, err := db.ListSources(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("app.Open reconciled sources implicitly: %#v", sources)
	}
}

func TestExplicitReconcileSourcesSurfacesErrors(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "example.toml"), config.Source{ID: "example", Type: config.SourceTypeRSS, Name: "Example", URL: "https://example.com/feed.xml", Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	db := openAppStore(t, a)
	defer db.Close()
	if _, err := db.Raw().Exec(`CREATE TRIGGER fail_reconcile BEFORE INSERT ON runtime_sources BEGIN SELECT RAISE(FAIL, 'fail reconcile'); END`); err != nil {
		t.Fatal(err)
	}
	if err := a.ReconcileSources(context.Background()); err == nil || !strings.Contains(err.Error(), "fail reconcile") {
		t.Fatalf("ReconcileSources error=%v want trigger failure", err)
	}
	sources, err := db.ListSources(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("failed reconciliation committed sources: %#v", sources)
	}
}

func TestReconcileSourcesRollsBackPartialLifecycleUpdates(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	for _, src := range []config.Source{
		{ID: "first", Type: config.SourceTypeRSS, Name: "First", URL: "https://example.com/first.xml", Enabled: true, Interval: "5m"},
		{ID: "second", Type: config.SourceTypeRSS, Name: "Second", URL: "https://example.com/second.xml", Enabled: true, Interval: "5m"},
	} {
		if err := config.WriteSource(filepath.Join(configDir, "sources.d", src.ID+".toml"), src); err != nil {
			t.Fatal(err)
		}
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	db := openAppStore(t, a)
	defer db.Close()
	if _, err := db.Raw().Exec(`CREATE TRIGGER fail_second_reconcile BEFORE INSERT ON runtime_sources WHEN NEW.id='second' BEGIN SELECT RAISE(FAIL, 'fail second reconcile'); END`); err != nil {
		t.Fatal(err)
	}
	if err := a.ReconcileSources(context.Background()); err == nil || !strings.Contains(err.Error(), "fail second reconcile") {
		t.Fatalf("ReconcileSources error=%v want second-source failure", err)
	}
	sources, err := db.ListSources(true)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 0 {
		t.Fatalf("partial reconciliation was committed: %#v", sources)
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

func TestAppSourceItemStatusAndStorageOperations(t *testing.T) {
	_, _ = testutil.IsolatedEnv(t)
	feed := testutil.RSSFeed("Example", testutil.DefaultItem("guid-1", "First", "Body"))
	server := testutil.FeedServer(t, &feed)
	defer server.Close()
	if _, err := app.AddRSS(context.Background(), server.URL, app.AddRSSParams{ID: "example", Name: "Example"}); err != nil {
		t.Fatal(err)
	}

	loaded, err := app.LoadOnly()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Sources) != 1 || loaded.Sources[0].ID != "example" {
		t.Fatalf("loaded sources=%#v", loaded.Sources)
	}

	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	sources, err := a.Sources(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].ID != "example" || sources[0].Lifecycle != "active" {
		t.Fatalf("sources=%#v", sources)
	}
	metadata, err := a.TestSource(context.Background(), "example")
	if err != nil {
		t.Fatal(err)
	}
	if metadata.Title != "Example" || metadata.ItemsFound != 1 {
		t.Fatalf("metadata=%#v", metadata)
	}

	dryDisabled, err := a.SetSourceEnabled("example", false, true)
	if err != nil {
		t.Fatal(err)
	}
	if dryDisabled.Enabled {
		t.Fatalf("dry disabled source=%#v", dryDisabled)
	}
	stillActive, err := a.Source("example")
	if err != nil {
		t.Fatal(err)
	}
	if stillActive.Lifecycle != "active" {
		t.Fatalf("dry-run changed source lifecycle: %#v", stillActive)
	}
	if _, err := a.SetSourceEnabled("example", false, false); err != nil {
		t.Fatal(err)
	}
	disabled, err := a.Source("example")
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Lifecycle != "disabled" || disabled.Enabled {
		t.Fatalf("disabled source=%#v", disabled)
	}
	if _, err := a.SetSourceEnabled("example", true, false); err != nil {
		t.Fatal(err)
	}

	syncRes := a.Sync(context.Background(), "example")
	if !syncRes.OK || len(syncRes.Sources) != 1 || syncRes.Sources[0].NewItems != 1 {
		t.Fatalf("sync result=%#v", syncRes)
	}
	items, err := a.Items(store.ItemFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%#v", items)
	}
	itemID := items[0].ID

	if err := a.ToggleRead(itemID); err != nil {
		t.Fatal(err)
	}
	readItem, err := a.Item(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if readItem.ReadAt == "" {
		t.Fatalf("item not marked read: %#v", readItem)
	}
	if err := a.ToggleRead(itemID); err != nil {
		t.Fatal(err)
	}
	readItem, err = a.Item(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if readItem.ReadAt != "" {
		t.Fatalf("item not marked unread: %#v", readItem)
	}
	if err := a.SetStarred(itemID, true); err != nil {
		t.Fatal(err)
	}
	starred, err := a.Item(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if !starred.Starred {
		t.Fatalf("item not starred: %#v", starred)
	}

	status, err := a.Status()
	if err != nil {
		t.Fatal(err)
	}
	if status.SourceCount != 1 || status.UnreadCount != 1 || status.LatestSyncStatus != "ok" {
		t.Fatalf("status=%#v", status)
	}

	mdPath, err := a.MarkdownPath(itemID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(mdPath); err != nil {
		t.Fatal(err)
	}
	orphanPath := filepath.Join(a.Paths().ContentDir, "orphan.md")
	if err := os.WriteFile(orphanPath, []byte("orphan"), 0o644); err != nil {
		t.Fatal(err)
	}
	reconciled, err := a.ReconcileStorage()
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.ScannedFiles != 1 || len(reconciled.MissingFiles) != 1 || reconciled.MissingFiles[0] != items[0].ContentPath || len(reconciled.OrphanedFiles) != 1 || reconciled.OrphanedFiles[0] != "orphan.md" {
		t.Fatalf("reconciled=%#v", reconciled)
	}

	db := openAppStore(t, a)
	defer db.Close()
	if err := db.CreateItem(store.Item{ID: "no-url", SourceID: "example", SourceItemID: "no-url", IdentityKind: "guid", Title: "No URL", ContentPath: "example/no-url.md", ContentHash: "sha256:no-url", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if err := a.OpenItem(context.Background(), "no-url"); err == nil || !strings.Contains(err.Error(), "item has no URL") {
		t.Fatalf("OpenItem no-url error=%v", err)
	}
}

func TestAppOpenActionsRespectCancelledContext(t *testing.T) {
	_, _ = testutil.IsolatedEnv(t)
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
	items, err := a.Items(store.ItemFilter{})
	if err != nil || len(items) != 1 {
		t.Fatalf("items len=%d err=%v", len(items), err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.OpenItem(ctx, items[0].ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenItem cancelled error=%v want context.Canceled", err)
	}
	if err := a.OpenMarkdown(ctx, items[0].ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("OpenMarkdown cancelled error=%v want context.Canceled", err)
	}
}

func TestAppErrorFormatting(t *testing.T) {
	inner := errors.New("inner")
	withMessage := app.AppError("code", "message", inner)
	if got := withMessage.Error(); got != "message" {
		t.Fatalf("Error with message=%q", got)
	}
	if !errors.Is(withMessage, inner) {
		t.Fatalf("AppError should unwrap inner error")
	}
	withoutMessage := app.AppError("code", "", inner)
	if got := withoutMessage.Error(); got != "inner" {
		t.Fatalf("Error without message=%q", got)
	}
	codeOnly := app.AppError("code", "", nil)
	if got := codeOnly.Error(); got != "code" {
		t.Fatalf("Error code only=%q", got)
	}
}

func openAppStore(t *testing.T, a *app.App) *store.DB {
	t.Helper()
	db, err := store.Open(a.Paths().Database)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func containsAll(value string, needles []string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
