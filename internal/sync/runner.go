package sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	stdsync "sync"
	"time"

	"feedctl/internal/config"
	"feedctl/internal/content"
	"feedctl/internal/metrics"
	"feedctl/internal/source"
	"feedctl/internal/store"
)

type Runner struct {
	Store   *store.DB
	Paths   config.Paths
	Config  config.Config
	Adapter source.Adapter
	Metrics *metrics.Enricher
}

type Options struct {
	SourceID string
}

type Result struct {
	OK      bool           `json:"ok"`
	Action  string         `json:"action"`
	Sources []SourceResult `json:"sources"`
	Errors  []string       `json:"errors,omitempty"`
}

type SourceResult struct {
	SourceID       string   `json:"source_id"`
	Status         string   `json:"status"`
	NewItems       int      `json:"new_items"`
	UpdatedItems   int      `json:"updated_items"`
	UnchangedItems int      `json:"unchanged_items"`
	ItemsFound     int      `json:"items_found"`
	Errors         []string `json:"errors,omitempty"`
}

func NewRunner(st *store.DB, paths config.Paths, cfg config.Config) *Runner {
	return &Runner{Store: st, Paths: paths, Config: cfg, Adapter: source.NewRSSAdapter(), Metrics: metrics.DefaultEnricher()}
}

func (r *Runner) RunAll(ctx context.Context, sources []config.Source, opts Options) Result {
	result := Result{OK: true, Action: "sync"}
	var selected []config.Source
	for _, src := range sources {
		if opts.SourceID != "" && src.ID != opts.SourceID {
			continue
		}
		selected = append(selected, src)
	}
	if opts.SourceID != "" && len(selected) == 0 {
		result.OK = false
		result.Errors = append(result.Errors, fmt.Sprintf("source %q not found", opts.SourceID))
		return result
	}
	concurrency := r.Config.Sync.Concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	sem := make(chan struct{}, concurrency)
	out := make(chan SourceResult, len(selected))
	var wg stdsync.WaitGroup
	for _, src := range selected {
		src := src
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out <- r.RunSource(ctx, src)
		}()
	}
	wg.Wait()
	close(out)
	for sr := range out {
		if sr.Status == "failed" {
			result.OK = false
			result.Errors = append(result.Errors, sr.Errors...)
		}
		result.Sources = append(result.Sources, sr)
	}
	sort.Slice(result.Sources, func(i, j int) bool { return result.Sources[i].SourceID < result.Sources[j].SourceID })
	_ = ReconcileStorage(r.Store, r.Paths)
	return result
}

func (r *Runner) RunSource(ctx context.Context, src config.Source) SourceResult {
	res := SourceResult{SourceID: src.ID}
	if !src.Enabled {
		res.Status = "skipped"
		res.Errors = append(res.Errors, "source disabled")
		return res
	}
	feed, err := r.Adapter.Fetch(ctx, src)
	if err != nil {
		res.Status = "failed"
		res.Errors = append(res.Errors, err.Error())
		_ = r.Store.UpdateSourceSyncStatus(src.ID, "failed", err.Error(), 0, 0, 0)
		return res
	}
	return r.runSourceClassified(ctx, src, feed)
}

func (r *Runner) runSourceClassified(ctx context.Context, src config.Source, feed source.Feed) SourceResult {
	res := SourceResult{SourceID: src.ID, ItemsFound: len(feed.Items), Status: "ok"}
	for _, feedItem := range feed.Items {
		classification, err := r.processItemClassified(ctx, src, feedItem)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			continue
		}
		switch classification {
		case "new":
			res.NewItems++
		case "updated":
			res.UpdatedItems++
		case "unchanged":
			res.UnchangedItems++
		}
	}
	if len(res.Errors) > 0 {
		res.Status = "failed"
		_ = r.Store.UpdateSourceSyncStatus(src.ID, "failed", joinErrors(res.Errors), res.NewItems, res.UpdatedItems, res.UnchangedItems)
		return res
	}
	_ = r.Store.UpdateSourceSyncStatus(src.ID, "ok", "", res.NewItems, res.UpdatedItems, res.UnchangedItems)
	return res
}

func (r *Runner) processItem(src config.Source, item source.Item) error {
	_, err := r.processItemClassified(context.Background(), src, item)
	return err
}

func (r *Runner) processItemClassified(ctx context.Context, src config.Source, item source.Item) (string, error) {
	identity, kind := source.Identity(src.ID, item)
	itemID := "item_" + content.ShortID(src.ID+"\x00"+identity)
	fetched := time.Now().UTC()
	published := fetched
	publishedString := ""
	if item.PublishedAt != nil {
		published = item.PublishedAt.UTC()
		publishedString = published.Format(time.RFC3339)
	}
	body := item.Body
	hash := content.StableHash(item.Title, item.URL, item.CanonicalURL, body)
	existing, err := r.Store.FindItemBySourceIdentity(src.ID, identity)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	if errors.Is(err, store.ErrNotFound) {
		rel := content.RenderPath(r.Config.Markdown.PathTemplate, content.PathData{SourceID: src.ID, Title: item.Title, ItemID: itemID, Time: published})
		rel, err = content.ResolveCollision(rel, itemID, func(candidate string) (bool, error) {
			return r.Store.IsContentPathAssigned(candidate, itemID)
		})
		if err != nil {
			return "", err
		}
		rendered := content.RenderMarkdown(content.RenderItem{
			ID: itemID, SourceID: src.ID, SourceName: sourceName(src, item), SourceType: src.Type,
			Title: item.Title, URL: item.URL, CanonicalURL: item.CanonicalURL, PublishedAt: publishedString,
			FetchedAt: fetched.Format(time.RFC3339), ContentHash: hash, Version: 1, Tags: item.Tags, Body: body,
		}, r.Config.Markdown.Frontmatter)
		if _, _, err := content.SafeWrite(r.Paths.ContentDir, rel, r.Paths.TmpDir, rendered); err != nil {
			return "", fmt.Errorf("write markdown for %s: %w", itemID, err)
		}
		if err := r.Store.CreateItem(store.Item{
			ID: itemID, SourceID: src.ID, SourceItemID: identity, IdentityKind: kind,
			Title: item.Title, URL: item.URL, CanonicalURL: item.CanonicalURL, PublishedAt: publishedString,
			FetchedAt: fetched.Format(time.RFC3339), LastSeenAt: fetched.Format(time.RFC3339), ContentPath: rel,
			ContentHash: hash, Version: 1, Tags: item.Tags,
		}); err != nil {
			return "", err
		}
		r.enrichItemMetrics(ctx, src, itemID, item)
		return "new", nil
	}
	if existing.ContentHash == hash {
		if err := r.Store.UpdateItemSeen(existing.ID); err != nil {
			return "", err
		}
		r.enrichItemMetrics(ctx, src, existing.ID, item)
		return "unchanged", nil
	}
	currentPath := filepath.Join(r.Paths.ContentDir, existing.ContentPath)
	versionRel, versionSize, err := content.SaveVersion(r.Paths.VersionsDir, existing.ID, existing.Version, currentPath, r.Paths.TmpDir)
	if err != nil {
		return "", fmt.Errorf("save version for %s: %w", existing.ID, err)
	}
	newVersion := existing.Version + 1
	rendered := content.RenderMarkdown(content.RenderItem{
		ID: existing.ID, SourceID: src.ID, SourceName: sourceName(src, item), SourceType: src.Type,
		Title: item.Title, URL: item.URL, CanonicalURL: item.CanonicalURL, PublishedAt: publishedString,
		FetchedAt: fetched.Format(time.RFC3339), ContentHash: hash, Version: newVersion, Tags: item.Tags, Body: body,
	}, r.Config.Markdown.Frontmatter)
	if _, _, err := content.SafeWrite(r.Paths.ContentDir, existing.ContentPath, r.Paths.TmpDir, rendered); err != nil {
		return "", fmt.Errorf("rewrite markdown for %s: %w", existing.ID, err)
	}
	if err := r.Store.AddItemVersion(store.ItemVersion{ID: existing.ID + fmt.Sprintf("_v%d", existing.Version), ItemID: existing.ID, Version: existing.Version, ContentPath: versionRel, ContentHash: existing.ContentHash, CreatedAt: fetched.Format(time.RFC3339), SizeBytes: versionSize}); err != nil {
		return "", err
	}
	existing.Title = item.Title
	existing.URL = item.URL
	existing.CanonicalURL = item.CanonicalURL
	existing.PublishedAt = publishedString
	existing.FetchedAt = fetched.Format(time.RFC3339)
	existing.LastSeenAt = fetched.Format(time.RFC3339)
	existing.ContentHash = hash
	existing.Version = newVersion
	existing.UpdatedAt = fetched.Format(time.RFC3339)
	existing.Tags = item.Tags
	if err := r.Store.UpdateItemChanged(existing); err != nil {
		return "", err
	}
	r.enrichItemMetrics(ctx, src, existing.ID, item)
	return "updated", nil
}

func (r *Runner) enrichItemMetrics(ctx context.Context, src config.Source, itemID string, item source.Item) {
	if r.Metrics == nil {
		return
	}
	itemMetrics, matched, err := r.Metrics.Fetch(ctx, metrics.Candidate{
		SourceID:     src.ID,
		SourceType:   src.Type,
		Title:        item.Title,
		URL:          item.URL,
		CanonicalURL: item.CanonicalURL,
	})
	if err != nil || !matched {
		return
	}
	_ = r.Store.UpsertItemMetrics(itemID, itemMetrics)
}

func ReconcileStorage(st *store.DB, paths config.Paths) error {
	currentBytes, err := scanBytes(paths.ContentDir)
	if err != nil {
		return err
	}
	versionBytes, err := scanBytes(paths.VersionsDir)
	if err != nil {
		return err
	}
	items, err := st.CountItems()
	if err != nil {
		return err
	}
	dbBytes := int64(0)
	if info, err := os.Stat(paths.Database); err == nil {
		dbBytes = info.Size()
	}
	if err := st.UpsertStorageStats("current_markdown", currentBytes, items); err != nil {
		return err
	}
	if err := st.UpsertStorageStats("versions", versionBytes, 0); err != nil {
		return err
	}
	return st.UpsertStorageStats("database", dbBytes, 0)
}

func scanBytes(root string) (int64, error) {
	var total int64
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

func sourceName(src config.Source, item source.Item) string {
	if src.Name != "" {
		return src.Name
	}
	if item.SourceName != "" {
		return item.SourceName
	}
	return src.ID
}

func identityOnly(srcID string, item source.Item) string {
	id, _ := source.Identity(srcID, item)
	return id
}

func joinErrors(values []string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += "; "
		}
		out += v
	}
	return out
}
