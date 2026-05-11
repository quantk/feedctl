package sync_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"feedctl/internal/app"
	"feedctl/internal/config"
	"feedctl/internal/store"
	"feedctl/internal/testutil"
)

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
