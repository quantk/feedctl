package store_test

import (
	"path/filepath"
	"testing"

	"feedctl/internal/config"
	"feedctl/internal/metrics"
	"feedctl/internal/store"
)

func TestItemMetricsMigrationUpsertAndLookup(t *testing.T) {
	db := openMetricsTestDB(t)
	defer db.Close()

	var tableName string
	if err := db.Raw().QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='item_metrics'`).Scan(&tableName); err != nil {
		t.Fatalf("item_metrics table missing: %v", err)
	}

	createMetricsTestItem(t, db)
	item, err := db.GetItem("item-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Metrics != nil {
		t.Fatalf("new item metrics=%#v want nil", item.Metrics)
	}

	zero := 0
	votes := 5
	favorites := 2
	reading := 23
	want := metrics.ItemMetrics{
		Provider:       "habr",
		Score:          &zero,
		CommentsCount:  &zero,
		VotesCount:     &votes,
		FavoritesCount: &favorites,
		ReadingCount:   &reading,
		FetchedAt:      "2026-05-11T15:00:00Z",
	}
	if err := db.UpsertItemMetrics("item-1", want); err != nil {
		t.Fatal(err)
	}

	item, err = db.GetItem("item-1")
	if err != nil {
		t.Fatal(err)
	}
	assertMetrics(t, item.Metrics, want)

	found, err := db.FindItemBySourceIdentity("source-1", "guid-1")
	if err != nil {
		t.Fatal(err)
	}
	assertMetrics(t, found.Metrics, want)

	listed, err := db.ListItems(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("listed=%d want 1", len(listed))
	}
	assertMetrics(t, listed[0].Metrics, want)
}

func TestItemMetricsMissingFieldsStayUnknown(t *testing.T) {
	db := openMetricsTestDB(t)
	defer db.Close()
	createMetricsTestItem(t, db)

	score := 0
	want := metrics.ItemMetrics{Provider: "habr", Score: &score, FetchedAt: "2026-05-11T15:00:00Z"}
	if err := db.UpsertItemMetrics("item-1", want); err != nil {
		t.Fatal(err)
	}
	item, err := db.GetItem("item-1")
	if err != nil {
		t.Fatal(err)
	}
	if item.Metrics == nil {
		t.Fatal("metrics is nil")
	}
	assertIntPtr(t, "score", item.Metrics.Score, 0)
	if item.Metrics.CommentsCount != nil {
		t.Fatalf("comments=%v want nil", *item.Metrics.CommentsCount)
	}
}

func openMetricsTestDB(t *testing.T) *store.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "feedctl.db"))
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func createMetricsTestItem(t *testing.T, db *store.DB) {
	t.Helper()
	if err := db.UpsertConfiguredSource(config.Source{ID: "source-1", Type: "rss", Name: "Source", URL: "https://example.com/feed.xml", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateItem(store.Item{
		ID:           "item-1",
		SourceID:     "source-1",
		SourceItemID: "guid-1",
		IdentityKind: "guid",
		Title:        "Title",
		URL:          "https://example.com/item",
		ContentPath:  "source-1/title.md",
		ContentHash:  "sha256:abc",
		Version:      1,
	}); err != nil {
		t.Fatal(err)
	}
}

func assertMetrics(t *testing.T, got *metrics.ItemMetrics, want metrics.ItemMetrics) {
	t.Helper()
	if got == nil {
		t.Fatal("metrics is nil")
	}
	if got.Provider != want.Provider || got.FetchedAt != want.FetchedAt {
		t.Fatalf("metrics provider/fetched=%#v want %#v", got, want)
	}
	assertIntPtr(t, "score", got.Score, *want.Score)
	assertIntPtr(t, "comments", got.CommentsCount, *want.CommentsCount)
	assertIntPtr(t, "votes", got.VotesCount, *want.VotesCount)
	assertIntPtr(t, "favorites", got.FavoritesCount, *want.FavoritesCount)
	assertIntPtr(t, "reading", got.ReadingCount, *want.ReadingCount)
}

func assertIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s=%d want %d", name, *got, want)
	}
}
