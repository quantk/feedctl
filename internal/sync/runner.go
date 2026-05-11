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

type SourceRepository interface {
	AcquireSourceSyncLock(context.Context, string) (func() error, error)
	UpdateSourceSyncStatusContext(context.Context, string, string, string, int, int, int) error
}

type ItemRepository interface {
	FindItemBySourceIdentityContext(context.Context, string, string) (store.Item, error)
	IsContentPathAssignedContext(context.Context, string, string) (bool, error)
	CreateItemContext(context.Context, store.Item) error
	UpdateItemSeenContext(context.Context, string) error
	AddItemVersionAndUpdateItemChangedContext(context.Context, store.ItemVersion, store.Item) error
	UpsertItemMetricsContext(context.Context, string, metrics.ItemMetrics) error
}

type StorageRepository interface {
	CountItems() (int, error)
	UpsertStorageStats(string, int64, int) error
}

type Repository interface {
	SourceRepository
	ItemRepository
	StorageRepository
}

type ContentWriter interface {
	SafeWrite(root, rel, tmpDir string, data []byte) (string, int64, error)
	SaveVersion(versionsRoot, itemID string, version int, currentPath string, tmpDir string) (string, int64, error)
}

type MetricsEnricher interface {
	Fetch(context.Context, metrics.Candidate) (metrics.ItemMetrics, bool, error)
}

type Clock interface {
	Now() time.Time
}

type Runner struct {
	Store   Repository
	Paths   config.Paths
	Config  config.Config
	Adapter source.Adapter
	Content ContentWriter
	Metrics MetricsEnricher
	Clock   Clock
}

type filesystemContentWriter struct{}

func (filesystemContentWriter) SafeWrite(root, rel, tmpDir string, data []byte) (string, int64, error) {
	return content.SafeWrite(root, rel, tmpDir, data)
}

func (filesystemContentWriter) SaveVersion(versionsRoot, itemID string, version int, currentPath string, tmpDir string) (string, int64, error) {
	return content.SaveVersion(versionsRoot, itemID, version, currentPath, tmpDir)
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now().UTC() }

var sourceSyncLocks stdsync.Map

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
	return &Runner{Store: st, Paths: paths, Config: cfg, Adapter: source.NewDefaultAdapter(), Content: filesystemContentWriter{}, Metrics: metrics.DefaultEnricher(), Clock: systemClock{}}
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
	unlock := lockSourceSync(src.ID)
	defer unlock()
	releaseDBLock, err := r.Store.AcquireSourceSyncLock(ctx, src.ID)
	if err != nil {
		res.Status = "failed"
		res.Errors = append(res.Errors, err.Error())
		_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "failed", err.Error(), 0, 0, 0)
		return res
	}
	defer func() { _ = releaseDBLock() }()
	if err := ctx.Err(); err != nil {
		res.Status = "failed"
		res.Errors = append(res.Errors, err.Error())
		_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "failed", err.Error(), 0, 0, 0)
		return res
	}
	fetchCtx, cancel := r.fetchContext(ctx)
	defer cancel()
	feed, err := r.Adapter.Fetch(fetchCtx, src)
	if err != nil {
		res.Status = "failed"
		res.Errors = append(res.Errors, err.Error())
		_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "failed", err.Error(), 0, 0, 0)
		return res
	}
	if err := ctx.Err(); err != nil {
		res.Status = "failed"
		res.Errors = append(res.Errors, err.Error())
		_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "failed", err.Error(), 0, 0, 0)
		return res
	}
	return r.runSourceClassified(ctx, src, feed)
}

func (r *Runner) runSourceClassified(ctx context.Context, src config.Source, feed source.Feed) SourceResult {
	res := SourceResult{SourceID: src.ID, ItemsFound: len(feed.Items), Status: "ok"}
	for _, feedItem := range feed.Items {
		if err := ctx.Err(); err != nil {
			res.Errors = append(res.Errors, err.Error())
			break
		}
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
	if err := ctx.Err(); err != nil {
		res.Errors = append(res.Errors, err.Error())
	}
	if len(res.Errors) > 0 {
		res.Status = "failed"
		_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "failed", joinErrors(res.Errors), res.NewItems, res.UpdatedItems, res.UnchangedItems)
		return res
	}
	_ = r.Store.UpdateSourceSyncStatusContext(ctx, src.ID, "ok", "", res.NewItems, res.UpdatedItems, res.UnchangedItems)
	return res
}

func (r *Runner) processItem(src config.Source, item source.Item) error {
	_, err := r.processItemClassified(context.Background(), src, item)
	return err
}

func (r *Runner) processItemClassified(ctx context.Context, src config.Source, item source.Item) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	identity, kind := source.Identity(src.ID, item)
	itemID := "item_" + content.ShortID(src.ID+"\x00"+identity)
	fetched := r.now()
	published := fetched
	publishedString := ""
	if item.PublishedAt != nil {
		published = item.PublishedAt.UTC()
		publishedString = published.Format(time.RFC3339)
	}
	body := item.Body
	hash := content.StableHash(item.Title, item.URL, item.CanonicalURL, body)
	existing, err := r.Store.FindItemBySourceIdentityContext(ctx, src.ID, identity)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return "", err
	}
	if errors.Is(err, store.ErrNotFound) {
		rel := content.RenderPath(r.Config.Markdown.PathTemplate, content.PathData{SourceID: src.ID, Title: item.Title, ItemID: itemID, Time: published})
		rel, err = content.ResolveCollision(rel, itemID, func(candidate string) (bool, error) {
			return r.Store.IsContentPathAssignedContext(ctx, candidate, "")
		})
		if err != nil {
			return "", err
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		rendered := renderMarkdownForSync(src, item, itemID, publishedString, fetched, hash, 1, r.Config.Markdown.Frontmatter)
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if _, _, err := r.contentWriter().SafeWrite(r.Paths.ContentDir, rel, r.Paths.TmpDir, rendered); err != nil {
			return "", fmt.Errorf("write markdown for %s: %w", itemID, err)
		}
		if err := ctx.Err(); err != nil {
			r.removeUnassignedContent(rel)
			return "", err
		}
		if err := r.Store.CreateItemContext(ctx, store.Item{
			ID: itemID, SourceID: src.ID, SourceItemID: identity, IdentityKind: kind,
			Title: item.Title, URL: item.URL, CanonicalURL: item.CanonicalURL, PublishedAt: publishedString,
			FetchedAt: fetched.Format(time.RFC3339), LastSeenAt: fetched.Format(time.RFC3339), ContentPath: rel,
			ContentHash: hash, Version: 1, Tags: item.Tags,
		}); err != nil {
			r.removeUnassignedContent(rel)
			return "", err
		}
		r.enrichItemMetrics(ctx, src, itemID, item)
		return "new", nil
	}
	if classifyItem(existing, hash) == "unchanged" {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if err := r.Store.UpdateItemSeenContext(ctx, existing.ID); err != nil {
			return "", err
		}
		r.enrichItemMetrics(ctx, src, existing.ID, item)
		return "unchanged", nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	currentPath := filepath.Join(r.Paths.ContentDir, existing.ContentPath)
	versionRel, versionSize, err := r.contentWriter().SaveVersion(r.Paths.VersionsDir, existing.ID, existing.Version, currentPath, r.Paths.TmpDir)
	if err != nil {
		return "", fmt.Errorf("save version for %s: %w", existing.ID, err)
	}
	newVersion := existing.Version + 1
	rendered := renderMarkdownForSync(src, item, existing.ID, publishedString, fetched, hash, newVersion, r.Config.Markdown.Frontmatter)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if _, _, err := r.contentWriter().SafeWrite(r.Paths.ContentDir, existing.ContentPath, r.Paths.TmpDir, rendered); err != nil {
		return "", fmt.Errorf("rewrite markdown for %s: %w", existing.ID, err)
	}
	if err := ctx.Err(); err != nil {
		r.restoreCurrentFromVersion(existing.ContentPath, versionRel)
		return "", err
	}
	version := store.ItemVersion{ID: existing.ID + fmt.Sprintf("_v%d", existing.Version), ItemID: existing.ID, Version: existing.Version, ContentPath: versionRel, ContentHash: existing.ContentHash, CreatedAt: fetched.Format(time.RFC3339), SizeBytes: versionSize}
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
	if err := r.Store.AddItemVersionAndUpdateItemChangedContext(ctx, version, existing); err != nil {
		r.restoreCurrentFromVersion(existing.ContentPath, versionRel)
		return "", err
	}
	r.enrichItemMetrics(ctx, src, existing.ID, item)
	return "updated", nil
}

func (r *Runner) now() time.Time {
	if r.Clock == nil {
		return time.Now().UTC()
	}
	return r.Clock.Now().UTC()
}

func (r *Runner) contentWriter() ContentWriter {
	if r.Content == nil {
		return filesystemContentWriter{}
	}
	return r.Content
}

func (r *Runner) removeUnassignedContent(rel string) {
	assigned, err := r.Store.IsContentPathAssignedContext(context.Background(), rel, "")
	if err == nil && assigned {
		return
	}
	_ = os.Remove(filepath.Join(r.Paths.ContentDir, rel))
}

func (r *Runner) restoreCurrentFromVersion(currentRel, versionRel string) {
	b, err := os.ReadFile(filepath.Join(r.Paths.VersionsDir, versionRel))
	if err != nil {
		return
	}
	_, _, _ = r.contentWriter().SafeWrite(r.Paths.ContentDir, currentRel, r.Paths.TmpDir, b)
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
	_ = r.Store.UpsertItemMetricsContext(ctx, itemID, itemMetrics)
}

func renderMarkdownForSync(src config.Source, item source.Item, itemID string, published string, fetched time.Time, hash string, version int, frontmatter bool) []byte {
	return content.RenderMarkdown(content.RenderItem{
		ID:           itemID,
		SourceID:     src.ID,
		SourceName:   sourceName(src, item),
		SourceType:   src.Type,
		Title:        item.Title,
		URL:          item.URL,
		CanonicalURL: item.CanonicalURL,
		PublishedAt:  published,
		FetchedAt:    fetched.Format(time.RFC3339),
		ContentHash:  hash,
		Version:      version,
		Tags:         item.Tags,
		Body:         item.Body,
	}, frontmatter)
}

func classifyItem(existing store.Item, contentHash string) string {
	if existing.ContentHash == contentHash {
		return "unchanged"
	}
	return "updated"
}

func ReconcileStorage(st StorageRepository, paths config.Paths) error {
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

func (r *Runner) fetchContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	d, err := config.ParseDuration(r.Config.Sync.FetchTimeout)
	if err != nil || d <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, d)
}

func lockSourceSync(sourceID string) func() {
	if sourceID == "" {
		sourceID = "<empty>"
	}
	actual, _ := sourceSyncLocks.LoadOrStore(sourceID, &stdsync.Mutex{})
	mu := actual.(*stdsync.Mutex)
	mu.Lock()
	return mu.Unlock
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
