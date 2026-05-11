package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"feedctl/internal/config"
	"feedctl/internal/store"
)

func TestSourceLifecycleListingAndSyncStatus(t *testing.T) {
	db, dbPath := openStoreTestDB(t)
	defer db.Close()

	upsertStoreSource(t, db, config.Source{ID: "active", Type: "rss", Name: "Active", URL: "https://example.com/feed.xml", Enabled: true, Interval: "10m", Tags: []string{"tech", "rss"}})
	upsertStoreSource(t, db, config.Source{ID: "disabled", Type: "telegram", Name: "Disabled", URL: "https://t.me/s/example", Enabled: false, Interval: "1h", Tags: []string{"telegram"}})

	active, err := db.GetSource("active")
	if err != nil {
		t.Fatal(err)
	}
	if active.Lifecycle != "active" || !active.Enabled || active.Interval != "10m" {
		t.Fatalf("active source=%#v", active)
	}
	assertStrings(t, "active tags", active.Tags, []string{"tech", "rss"})

	listed, err := db.ListSources(false)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceIDs(t, listed, []string{"active", "disabled"})

	if err := db.MarkRemovedExcept(map[string]struct{}{"active": {}}); err != nil {
		t.Fatal(err)
	}
	listed, err = db.ListSources(false)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceIDs(t, listed, []string{"active"})

	all, err := db.ListSources(true)
	if err != nil {
		t.Fatal(err)
	}
	assertSourceIDs(t, all, []string{"active", "disabled"})
	removed, err := db.GetSource("disabled")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Lifecycle != "removed" || removed.Enabled || removed.RemovedAt == "" {
		t.Fatalf("removed source=%#v", removed)
	}

	if err := db.UpdateSourceSyncStatus("active", "ok", "", 2, 1, 3); err != nil {
		t.Fatal(err)
	}
	active, err = db.GetSource("active")
	if err != nil {
		t.Fatal(err)
	}
	if active.LastSyncStatus != "ok" || active.LastSyncAt == "" || active.LastError != "" {
		t.Fatalf("sync status source=%#v", active)
	}

	upsertStoreSource(t, db, config.Source{ID: "disabled", Type: "telegram", Name: "Reappeared", URL: "https://t.me/s/example", Enabled: true, Tags: []string{"back"}})
	reappeared, err := db.GetSource("disabled")
	if err != nil {
		t.Fatal(err)
	}
	if reappeared.Lifecycle != "active" || !reappeared.Enabled || reappeared.RemovedAt != "" || reappeared.Name != "Reappeared" {
		t.Fatalf("reappeared source=%#v", reappeared)
	}
	assertStrings(t, "reappeared tags", reappeared.Tags, []string{"back"})

	status, err := db.Status(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if status.SourceCount != 2 || status.RemovedSourceCount != 0 || status.LatestSyncStatus != "ok" {
		t.Fatalf("status=%#v", status)
	}
}

func TestSourceMappingRejectsUnknownLifecycleAndSyncStatus(t *testing.T) {
	db, _ := openStoreTestDB(t)
	defer db.Close()
	upsertStoreSource(t, db, config.Source{ID: "bad", Type: "rss", Name: "Bad", URL: "https://example.com/feed.xml", Enabled: true})

	if _, err := db.Raw().Exec(`UPDATE runtime_sources SET lifecycle='mystery' WHERE id='bad'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.GetSource("bad"); err == nil {
		t.Fatal("GetSource accepted unknown lifecycle")
	}
	if _, err := db.Raw().Exec(`UPDATE runtime_sources SET lifecycle='active', last_sync_status='mystery' WHERE id='bad'`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ListSources(true); err == nil {
		t.Fatal("ListSources accepted unknown sync status")
	}
}

func TestItemFiltersStateTransitionsAndUpdates(t *testing.T) {
	db, _ := openStoreTestDB(t)
	defer db.Close()

	upsertStoreSource(t, db, config.Source{ID: "active", Type: "rss", Name: "Active", URL: "https://example.com/feed.xml", Enabled: true})
	upsertStoreSource(t, db, config.Source{ID: "removed", Type: "rss", Name: "Removed", URL: "https://example.com/old.xml", Enabled: true})
	if err := db.MarkRemovedExcept(map[string]struct{}{"active": {}}); err != nil {
		t.Fatal(err)
	}

	createStoreItem(t, db, store.Item{ID: "item-active", SourceID: "active", SourceItemID: "guid-active", IdentityKind: "guid", Title: "Active", URL: "https://example.com/a", ContentPath: "active/a.md", ContentHash: "sha256:a", Tags: []string{"one"}})
	createStoreItem(t, db, store.Item{ID: "item-archived", SourceID: "active", SourceItemID: "guid-archived", IdentityKind: "guid", Title: "Archived", URL: "https://example.com/b", ContentPath: "active/b.md", ContentHash: "sha256:b", ReadAt: "2026-01-01T00:00:00Z", Starred: true, ArchivedAt: "2026-01-02T00:00:00Z"})
	createStoreItem(t, db, store.Item{ID: "item-removed", SourceID: "removed", SourceItemID: "guid-removed", IdentityKind: "guid", Title: "Removed", URL: "https://example.com/c", ContentPath: "removed/c.md", ContentHash: "sha256:c"})

	items, err := db.ListItems(store.ItemFilter{})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-active"})

	items, err = db.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-removed", "item-archived", "item-active"})

	items, err = db.ListItems(store.ItemFilter{RemovedSources: true})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-removed"})

	items, err = db.ListItems(store.ItemFilter{Unread: true})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-active"})

	items, err = db.ListItems(store.ItemFilter{AllItems: true, Starred: true})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-archived"})

	items, err = db.ListItems(store.ItemFilter{AllItems: true, SourceID: "active"})
	if err != nil {
		t.Fatal(err)
	}
	assertItemIDs(t, items, []string{"item-archived", "item-active"})

	assigned, err := db.IsContentPathAssigned("active/a.md", "different-item")
	if err != nil {
		t.Fatal(err)
	}
	if !assigned {
		t.Fatal("content path should be assigned to another item")
	}
	assigned, err = db.IsContentPathAssigned("active/a.md", "item-active")
	if err != nil {
		t.Fatal(err)
	}
	if assigned {
		t.Fatal("content path should be ignored for except item")
	}

	if err := db.SetRead("item-active", true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStarred("item-active", true); err != nil {
		t.Fatal(err)
	}
	if err := db.SetArchived("item-active", true); err != nil {
		t.Fatal(err)
	}
	item, err := db.GetItem("item-active")
	if err != nil {
		t.Fatal(err)
	}
	if item.ReadAt == "" || !item.Starred || item.ArchivedAt == "" {
		t.Fatalf("updated item=%#v", item)
	}

	if err := db.SetRead("item-active", false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStarred("item-active", false); err != nil {
		t.Fatal(err)
	}
	if err := db.SetArchived("item-active", false); err != nil {
		t.Fatal(err)
	}
	item, err = db.GetItem("item-active")
	if err != nil {
		t.Fatal(err)
	}
	if item.ReadAt != "" || item.Starred || item.ArchivedAt != "" {
		t.Fatalf("cleared item=%#v", item)
	}

	beforeSeen := item.LastSeenAt
	if err := db.UpdateItemSeen("item-active"); err != nil {
		t.Fatal(err)
	}
	item, err = db.GetItem("item-active")
	if err != nil {
		t.Fatal(err)
	}
	if item.LastSeenAt == "" || item.LastSeenAt == beforeSeen {
		t.Fatalf("last_seen_at=%q before=%q", item.LastSeenAt, beforeSeen)
	}

	changed := item
	changed.Title = "Changed"
	changed.URL = "https://example.com/changed"
	changed.CanonicalURL = "https://example.com/canonical"
	changed.PublishedAt = "2026-05-11T12:00:00Z"
	changed.FetchedAt = "2026-05-11T12:01:00Z"
	changed.LastSeenAt = "2026-05-11T12:02:00Z"
	changed.ContentPath = "active/changed.md"
	changed.ContentHash = "sha256:changed"
	changed.Version = 2
	changed.UpdatedAt = "2026-05-11T12:03:00Z"
	changed.Tags = []string{"changed", "tag"}
	if err := db.UpdateItemChanged(changed); err != nil {
		t.Fatal(err)
	}
	item, err = db.GetItem("item-active")
	if err != nil {
		t.Fatal(err)
	}
	if item.Title != "Changed" || item.Version != 2 || item.ContentHash != "sha256:changed" || item.SourceLifecycle != "active" {
		t.Fatalf("changed item=%#v", item)
	}
	assertStrings(t, "changed tags", item.Tags, []string{"changed", "tag"})

	if err := db.AddItemVersion(store.ItemVersion{ID: "version-1", ItemID: "item-active", Version: 1, ContentPath: "versions/item-active/v1.md", ContentHash: "sha256:old", CreatedAt: "2026-05-11T12:04:00Z", SizeBytes: 123}); err != nil {
		t.Fatal(err)
	}
	var versionCount int
	if err := db.Raw().QueryRow(`SELECT COUNT(*) FROM item_versions WHERE item_id=? AND version=? AND size_bytes=?`, "item-active", 1, 123).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if versionCount != 1 {
		t.Fatalf("versionCount=%d want 1", versionCount)
	}
}

func TestStorageStatsStatusAndFormatting(t *testing.T) {
	db, dbPath := openStoreTestDB(t)
	defer db.Close()

	upsertStoreSource(t, db, config.Source{ID: "active", Type: "rss", Name: "Active", URL: "https://example.com/feed.xml", Enabled: true})
	upsertStoreSource(t, db, config.Source{ID: "removed", Type: "rss", Name: "Removed", URL: "https://example.com/removed.xml", Enabled: true})
	if err := db.MarkRemovedExcept(map[string]struct{}{"active": {}}); err != nil {
		t.Fatal(err)
	}
	createStoreItem(t, db, store.Item{ID: "unread", SourceID: "active", SourceItemID: "guid-unread", IdentityKind: "guid", Title: "Unread", URL: "https://example.com/unread", ContentPath: "active/unread.md", ContentHash: "sha256:unread"})
	createStoreItem(t, db, store.Item{ID: "read", SourceID: "active", SourceItemID: "guid-read", IdentityKind: "guid", Title: "Read", URL: "https://example.com/read", ContentPath: "active/read.md", ContentHash: "sha256:read", ReadAt: "2026-01-01T00:00:00Z"})
	createStoreItem(t, db, store.Item{ID: "removed-unread", SourceID: "removed", SourceItemID: "guid-removed", IdentityKind: "guid", Title: "Removed", URL: "https://example.com/removed", ContentPath: "removed/unread.md", ContentHash: "sha256:removed"})
	if err := db.SetArchived("read", true); err != nil {
		t.Fatal(err)
	}

	if err := db.UpsertStorageStats("current_markdown", 2048, 3); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertStorageStats("versions", 512, 1); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertStorageStats("database", 1, 0); err != nil {
		t.Fatal(err)
	}

	stats, err := db.GetStorageStats(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if stats.ItemsCount != 3 || stats.CurrentMarkdownBytes != 2048 || stats.VersionsBytes != 512 || stats.DatabaseBytes <= 1 || stats.TotalBytes != stats.Total() || stats.UpdatedAt == "" {
		t.Fatalf("stats=%#v", stats)
	}

	count, err := db.CountItems()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("count=%d want 3", count)
	}

	if err := db.UpdateSourceSyncStatus("active", "failed", "boom", 0, 0, 0); err != nil {
		t.Fatal(err)
	}
	status, err := db.Status(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if status.UnreadCount != 1 || status.SourceCount != 1 || status.RemovedSourceCount != 1 || status.LatestSyncStatus != "failed" || status.LatestSyncAt == "" {
		t.Fatalf("status=%#v", status)
	}
	if status.Storage.TotalBytes != stats.TotalBytes {
		t.Fatalf("status storage total=%d want %d", status.Storage.TotalBytes, stats.TotalBytes)
	}

	formatCases := map[int64]string{
		0:           "0B",
		1023:        "1023B",
		1024:        "1.0KB",
		1536:        "1.5KB",
		1024 * 1024: "1.0MB",
	}
	for input, want := range formatCases {
		if got := store.HumanBytes(input); got != want {
			t.Fatalf("HumanBytes(%d)=%q want %q", input, got, want)
		}
	}
}

func TestStoreNotFoundErrors(t *testing.T) {
	db, _ := openStoreTestDB(t)
	defer db.Close()

	if _, err := db.GetSource("missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetSource error=%v want ErrNotFound", err)
	}
	if _, err := db.GetItem("missing"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetItem error=%v want ErrNotFound", err)
	}
	if _, err := db.FindItemBySourceIdentity("missing-source", "missing-item"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("FindItemBySourceIdentity error=%v want ErrNotFound", err)
	}
	if err := db.SetRead("missing", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("SetRead error=%v want ErrNotFound", err)
	}
	if err := db.SetStarred("missing", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("SetStarred error=%v want ErrNotFound", err)
	}
	if err := db.SetArchived("missing", true); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("SetArchived error=%v want ErrNotFound", err)
	}
	if err := (*store.DB)(nil).Close(); err != nil {
		t.Fatalf("nil Close error=%v", err)
	}
}

func openStoreTestDB(t *testing.T) (*store.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "feedctl.db")
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return db, path
}

func upsertStoreSource(t *testing.T, db *store.DB, src config.Source) {
	t.Helper()
	if err := db.UpsertConfiguredSource(src); err != nil {
		t.Fatal(err)
	}
}

func createStoreItem(t *testing.T, db *store.DB, item store.Item) {
	t.Helper()
	if item.PublishedAt == "" {
		item.PublishedAt = "2026-05-11T12:00:00Z"
	}
	if item.FetchedAt == "" {
		item.FetchedAt = "2026-05-11T12:00:00Z"
	}
	if item.LastSeenAt == "" {
		item.LastSeenAt = item.FetchedAt
	}
	if item.Version == 0 {
		item.Version = 1
	}
	if err := db.CreateItem(item); err != nil {
		t.Fatal(err)
	}
}

func assertSourceIDs(t *testing.T, sources []store.Source, want []string) {
	t.Helper()
	got := make([]string, len(sources))
	for i, source := range sources {
		got[i] = source.ID
	}
	assertStrings(t, "source ids", got, want)
}

func assertItemIDs(t *testing.T, items []store.Item, want []string) {
	t.Helper()
	got := make([]string, len(items))
	for i, item := range items {
		got[i] = item.ID
	}
	assertStrings(t, "item ids", got, want)
}

func assertStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s len=%d values=%v want len=%d values=%v", label, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s[%d]=%q values=%v want %q values=%v", label, i, got[i], got, want[i], want)
		}
	}
}
