package app

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"feedctl/internal/config"
	"feedctl/internal/source"
	"feedctl/internal/store"
	feedSync "feedctl/internal/sync"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e Error) Unwrap() error { return e.Err }

func AppError(code, message string, err error) Error {
	return Error{Code: code, Message: message, Err: err}
}

type App struct {
	Loaded config.Loaded
	Store  *store.DB
}

type AddRSSParams struct {
	ID     string
	Name   string
	Tags   []string
	DryRun bool
}

type AddRSSResult struct {
	Action     string          `json:"action"`
	DryRun     bool            `json:"dry_run"`
	SourceID   string          `json:"source_id"`
	SourceType string          `json:"source_type"`
	ConfigPath string          `json:"config_path"`
	ItemsFound int             `json:"items_found"`
	Metadata   source.Metadata `json:"metadata"`
}

type RemoveResult struct {
	Action       string `json:"action"`
	DryRun       bool   `json:"dry_run"`
	SourceID     string `json:"source_id"`
	ConfigPath   string `json:"config_path"`
	RuntimeKept  bool   `json:"runtime_kept"`
	MarkdownKept bool   `json:"markdown_kept"`
}

type StorageReconcileResult struct {
	Storage       store.StorageStats `json:"storage"`
	ScannedFiles  int                `json:"scanned_files"`
	MissingFiles  []string           `json:"missing_files,omitempty"`
	OrphanedFiles []string           `json:"orphaned_files,omitempty"`
}

func Open(ctx context.Context) (*App, error) {
	_ = ctx
	loaded, err := config.Load()
	if err != nil {
		return nil, err
	}
	if err := config.EnsureConfigDirs(loaded.Paths); err != nil {
		return nil, err
	}
	if err := config.EnsureRuntimeDirs(loaded.Paths); err != nil {
		return nil, err
	}
	st, err := store.Open(loaded.Paths.Database)
	if err != nil {
		return nil, err
	}
	a := &App{Loaded: loaded, Store: st}
	if err := a.ReconcileSources(); err != nil {
		_ = st.Close()
		return nil, err
	}
	return a, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	return a.Store.Close()
}

func (a *App) ReconcileSources() error {
	ids := make(map[string]struct{}, len(a.Loaded.Sources))
	for _, src := range a.Loaded.Sources {
		ids[src.ID] = struct{}{}
		if err := a.Store.UpsertConfiguredSource(src); err != nil {
			return err
		}
	}
	return a.Store.MarkRemovedExcept(ids)
}

func LoadOnly() (config.Loaded, error) {
	loaded, err := config.Load()
	if err != nil {
		return loaded, err
	}
	return loaded, loaded.Validate()
}

func AddRSS(ctx context.Context, rawURL string, p AddRSSParams) (AddRSSResult, error) {
	loaded, err := config.Load()
	if err != nil {
		return AddRSSResult{}, err
	}
	if err := config.EnsureConfigDirs(loaded.Paths); err != nil {
		return AddRSSResult{}, err
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		if err == nil {
			err = fmt.Errorf("url must include scheme and host")
		}
		return AddRSSResult{}, AppError("invalid-url", "invalid RSS URL", err)
	}
	adapter := source.NewRSSAdapter()
	metadata, err := adapter.Test(ctx, config.Source{ID: "test", Type: config.SourceTypeRSS, Name: p.Name, URL: rawURL, Enabled: true, Tags: p.Tags})
	if err != nil {
		return AddRSSResult{}, AppError("source-test-failed", "RSS feed could not be fetched or parsed", err)
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = metadata.Title
	}
	if name == "" {
		name = parsed.Host
	}
	id := strings.TrimSpace(p.ID)
	if id == "" {
		id = config.GenerateSourceID(name, rawURL)
	}
	if !config.ValidateSourceID(id) {
		return AddRSSResult{}, AppError("invalid-source-id", "source id is not file-safe", nil)
	}
	path := config.SourcePath(loaded.Paths.SourcesDir, id)
	if _, err := os.Stat(path); err == nil {
		return AddRSSResult{}, AppError("source-already-exists", "source already exists", nil)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return AddRSSResult{}, err
	}
	res := AddRSSResult{Action: "create_source", DryRun: p.DryRun, SourceID: id, SourceType: config.SourceTypeRSS, ConfigPath: path, ItemsFound: metadata.ItemsFound, Metadata: metadata}
	if p.DryRun {
		return res, nil
	}
	src := config.Source{ID: id, Type: config.SourceTypeRSS, Name: name, URL: rawURL, Enabled: true, Interval: loaded.Config.Sync.DefaultInterval, Tags: p.Tags}
	if err := config.WriteSource(path, src); err != nil {
		return AddRSSResult{}, err
	}
	return res, nil
}

func (a *App) Sources(includeRemoved bool) ([]store.Source, error) {
	return a.Store.ListSources(includeRemoved)
}

func (a *App) Source(id string) (store.Source, error) {
	s, err := a.Store.GetSource(id)
	if errors.Is(err, store.ErrNotFound) {
		return store.Source{}, AppError("source-not-found", "source not found", err)
	}
	return s, err
}

func (a *App) TestSource(ctx context.Context, id string) (source.Metadata, error) {
	src, err := a.configSource(id)
	if err != nil {
		return source.Metadata{}, err
	}
	return source.NewRSSAdapter().Test(ctx, src)
}

func (a *App) SetSourceEnabled(id string, enabled bool, dryRun bool) (config.Source, error) {
	src, err := a.configSource(id)
	if err != nil {
		return config.Source{}, err
	}
	src.Enabled = enabled
	if dryRun {
		return src, nil
	}
	if err := config.WriteSource(src.FilePath, src); err != nil {
		return config.Source{}, err
	}
	a.Loaded, _ = config.Load()
	_ = a.ReconcileSources()
	return src, nil
}

func (a *App) RemoveSource(id string, dryRun bool) (RemoveResult, error) {
	src, err := a.configSource(id)
	if err != nil {
		return RemoveResult{}, err
	}
	res := RemoveResult{Action: "remove_source", DryRun: dryRun, SourceID: id, ConfigPath: src.FilePath, RuntimeKept: true, MarkdownKept: true}
	if dryRun {
		return res, nil
	}
	if err := os.Remove(src.FilePath); err != nil {
		return RemoveResult{}, err
	}
	a.Loaded, _ = config.Load()
	_ = a.ReconcileSources()
	return res, nil
}

func (a *App) Sync(ctx context.Context, sourceID string) feedSync.Result {
	runner := feedSync.NewRunner(a.Store, a.Loaded.Paths, a.Loaded.Config)
	return runner.RunAll(ctx, a.Loaded.Sources, feedSync.Options{SourceID: sourceID})
}

func (a *App) Items(filter store.ItemFilter) ([]store.Item, error) {
	return a.Store.ListItems(filter)
}

func (a *App) Item(id string) (store.Item, error) {
	item, err := a.Store.GetItem(id)
	if errors.Is(err, store.ErrNotFound) {
		return store.Item{}, AppError("item-not-found", "item not found", err)
	}
	return item, err
}

func (a *App) MarkdownPath(id string) (string, error) {
	item, err := a.Item(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(a.Loaded.Paths.ContentDir, item.ContentPath), nil
}

func (a *App) OpenItem(ctx context.Context, id string) error {
	_ = ctx
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	if item.URL == "" {
		return AppError("item-url-missing", "item has no URL", nil)
	}
	cmd := exec.Command(a.Loaded.Config.TUI.Browser, item.URL)
	return cmd.Start()
}

func (a *App) OpenMarkdown(ctx context.Context, id string) error {
	_ = ctx
	path, err := a.MarkdownPath(id)
	if err != nil {
		return err
	}
	cmd := exec.Command(a.Loaded.Config.TUI.Editor, path)
	return cmd.Start()
}

func (a *App) SetRead(id string, read bool) error {
	return a.Store.SetRead(id, read)
}

func (a *App) ToggleRead(id string) error {
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	return a.Store.SetRead(id, item.ReadAt == "")
}

func (a *App) SetStarred(id string, starred bool) error {
	return a.Store.SetStarred(id, starred)
}

func (a *App) ToggleStarred(id string) error {
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	return a.Store.SetStarred(id, !item.Starred)
}

func (a *App) Archive(id string) error {
	return a.Store.SetArchived(id, true)
}

func (a *App) Storage() (store.StorageStats, error) {
	return a.Store.GetStorageStats(a.Loaded.Paths.Database)
}

func (a *App) ReconcileStorage() (StorageReconcileResult, error) {
	items, err := a.Items(store.ItemFilter{AllItems: true})
	if err != nil {
		return StorageReconcileResult{}, err
	}
	expected := map[string]struct{}{}
	var missing []string
	for _, item := range items {
		expected[item.ContentPath] = struct{}{}
		if _, err := os.Stat(filepath.Join(a.Loaded.Paths.ContentDir, item.ContentPath)); err != nil && errors.Is(err, os.ErrNotExist) {
			missing = append(missing, item.ContentPath)
		}
	}
	var scanned int
	var orphaned []string
	_ = filepath.WalkDir(a.Loaded.Paths.ContentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		scanned++
		rel, err := filepath.Rel(a.Loaded.Paths.ContentDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, ok := expected[rel]; !ok {
			orphaned = append(orphaned, rel)
		}
		return nil
	})
	if err := feedSync.ReconcileStorage(a.Store, a.Loaded.Paths); err != nil {
		return StorageReconcileResult{}, err
	}
	stats, err := a.Storage()
	return StorageReconcileResult{Storage: stats, ScannedFiles: scanned, MissingFiles: missing, OrphanedFiles: orphaned}, err
}

func (a *App) Status() (store.StatusSummary, error) {
	return a.Store.Status(a.Loaded.Paths.Database)
}

func (a *App) configSource(id string) (config.Source, error) {
	for _, src := range a.Loaded.Sources {
		if src.ID == id {
			return src, nil
		}
	}
	return config.Source{}, AppError("source-not-found", "source not found", store.ErrNotFound)
}
