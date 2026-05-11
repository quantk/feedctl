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

func TestRSSSyncSourceFailureIsolation(t *testing.T) {
	configDir, _ := testutil.IsolatedEnv(t)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("bad")) }))
	defer bad.Close()
	goodFeed := testutil.RSSFeed("Good", testutil.DefaultItem("good-1", "Good Item", "Body"))
	good := testutil.FeedServer(t, &goodFeed)
	defer good.Close()
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "bad.toml"), config.Source{ID: "bad", Type: "rss", Name: "Bad", URL: bad.URL, Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	if err := config.WriteSource(filepath.Join(configDir, "sources.d", "good.toml"), config.Source{ID: "good", Type: "rss", Name: "Good", URL: good.URL, Enabled: true, Interval: "5m"}); err != nil {
		t.Fatal(err)
	}
	a, err := app.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	res := a.Sync(context.Background(), "")
	if res.OK {
		t.Fatalf("overall sync should report failure when one source fails: %#v", res)
	}
	items, err := a.Items(store.ItemFilter{AllItems: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].SourceID != "good" {
		t.Fatalf("good source did not sync despite bad failure: %#v", items)
	}
}
