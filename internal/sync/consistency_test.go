package sync_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"feedctl/internal/config"
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
