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
	loaded config.Loaded
	store  *store.DB
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

type AddTelegramParams struct {
	ID       string
	Name     string
	Tags     []string
	MaxItems int
	DryRun   bool
}

type AddTelegramResult struct {
	Action       string          `json:"action"`
	DryRun       bool            `json:"dry_run"`
	SourceID     string          `json:"source_id"`
	SourceType   string          `json:"source_type"`
	ConfigPath   string          `json:"config_path"`
	CanonicalURL string          `json:"canonical_url"`
	ItemsFound   int             `json:"items_found"`
	Metadata     source.Metadata `json:"metadata"`
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
	return &App{loaded: loaded, store: st}, nil
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}
	return a.store.Close()
}

func (a *App) Paths() config.Paths {
	if a == nil {
		return config.Paths{}
	}
	return a.loaded.Paths
}

func (a *App) Config() config.Config {
	if a == nil {
		return config.Config{}
	}
	return a.loaded.Config
}

func (a *App) ConfiguredSources() []config.Source {
	if a == nil {
		return nil
	}
	return append([]config.Source(nil), a.loaded.Sources...)
}

func (a *App) ReconcileSources(ctx context.Context) error {
	if a == nil || a.store == nil {
		return nil
	}
	return a.store.ReconcileConfiguredSources(ctx, a.loaded.Sources)
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

func AddTelegram(ctx context.Context, rawChannel string, p AddTelegramParams) (AddTelegramResult, error) {
	loaded, err := config.Load()
	if err != nil {
		return AddTelegramResult{}, err
	}
	if err := config.EnsureConfigDirs(loaded.Paths); err != nil {
		return AddTelegramResult{}, err
	}
	channel, err := source.NormalizeTelegramChannelInput(rawChannel)
	if err != nil {
		return AddTelegramResult{}, AppError("invalid-url", "invalid Telegram channel", err)
	}
	adapter := source.NewTelegramAdapter()
	testSource := config.Source{ID: "test", Type: config.SourceTypeTelegram, Name: p.Name, URL: channel.PublicURL, Enabled: true, Tags: p.Tags, MaxItems: p.MaxItems}
	metadata, err := adapter.Test(ctx, testSource)
	if err != nil {
		return AddTelegramResult{}, AppError("source-test-failed", "Telegram channel could not be fetched or parsed", err)
	}
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = metadata.Title
	}
	if name == "" {
		name = channel.Channel
	}
	id := strings.TrimSpace(p.ID)
	if id == "" {
		id = strings.ToLower(channel.Channel)
	}
	if !config.ValidateSourceID(id) {
		return AddTelegramResult{}, AppError("invalid-source-id", "source id is not file-safe", nil)
	}
	path := config.SourcePath(loaded.Paths.SourcesDir, id)
	if _, err := os.Stat(path); err == nil {
		return AddTelegramResult{}, AppError("source-already-exists", "source already exists", nil)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return AddTelegramResult{}, err
	}
	res := AddTelegramResult{Action: "create_source", DryRun: p.DryRun, SourceID: id, SourceType: config.SourceTypeTelegram, ConfigPath: path, CanonicalURL: channel.PublicURL, ItemsFound: metadata.ItemsFound, Metadata: metadata}
	if p.DryRun {
		return res, nil
	}
	src := config.Source{ID: id, Type: config.SourceTypeTelegram, Name: name, URL: channel.PublicURL, Enabled: true, Interval: loaded.Config.Sync.DefaultInterval, Tags: p.Tags, MaxItems: p.MaxItems}
	if err := config.WriteSource(path, src); err != nil {
		return AddTelegramResult{}, err
	}
	return res, nil
}

func (a *App) Sources(includeRemoved bool) ([]Source, error) {
	if err := a.ReconcileSources(context.Background()); err != nil {
		return nil, err
	}
	return a.store.ListSources(includeRemoved)
}

func (a *App) Source(id string) (Source, error) {
	if err := a.ReconcileSources(context.Background()); err != nil {
		return Source{}, err
	}
	s, err := a.store.GetSource(id)
	if errors.Is(err, store.ErrNotFound) {
		return Source{}, AppError("source-not-found", "source not found", err)
	}
	return s, err
}

func (a *App) TestSource(ctx context.Context, id string) (source.Metadata, error) {
	src, err := a.configSource(id)
	if err != nil {
		return source.Metadata{}, err
	}
	return source.NewDefaultAdapter().Test(ctx, src)
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
	loaded, err := config.Load()
	if err != nil {
		return config.Source{}, err
	}
	a.loaded = loaded
	if err := a.ReconcileSources(context.Background()); err != nil {
		return config.Source{}, err
	}
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
	if err := a.ReconcileSources(context.Background()); err != nil {
		return RemoveResult{}, err
	}
	if err := os.Remove(src.FilePath); err != nil {
		return RemoveResult{}, err
	}
	loaded, err := config.Load()
	if err != nil {
		return RemoveResult{}, err
	}
	a.loaded = loaded
	if err := a.ReconcileSources(context.Background()); err != nil {
		return RemoveResult{}, err
	}
	return res, nil
}

func (a *App) Sync(ctx context.Context, sourceID string) feedSync.Result {
	if err := a.ReconcileSources(ctx); err != nil {
		return feedSync.Result{OK: false, Action: "sync", Errors: []string{err.Error()}}
	}
	runner := feedSync.NewRunner(a.store, a.loaded.Paths, a.loaded.Config)
	return runner.RunAll(ctx, a.loaded.Sources, feedSync.Options{SourceID: sourceID})
}

func (a *App) Items(filter ItemFilter) ([]Item, error) {
	if err := a.ReconcileSources(context.Background()); err != nil {
		return nil, err
	}
	return a.store.ListItems(filter)
}

func (a *App) Item(id string) (Item, error) {
	item, err := a.store.GetItem(id)
	if errors.Is(err, store.ErrNotFound) {
		return Item{}, AppError("item-not-found", "item not found", err)
	}
	return item, err
}

func (a *App) MarkdownPath(id string) (string, error) {
	item, err := a.Item(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(a.loaded.Paths.ContentDir, item.ContentPath), nil
}

func (a *App) OpenItem(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	if item.URL == "" {
		return AppError("item-url-missing", "item has no URL", nil)
	}
	cmd := exec.CommandContext(ctx, a.loaded.Config.TUI.Browser, item.URL)
	return cmd.Start()
}

func (a *App) OpenMarkdown(ctx context.Context, id string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := a.MarkdownPath(id)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, a.loaded.Config.TUI.Editor, path)
	return cmd.Start()
}

func (a *App) SetRead(id string, read bool) error {
	return a.store.SetRead(id, read)
}

func (a *App) ToggleRead(id string) error {
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	return a.store.SetRead(id, item.ReadAt == "")
}

func (a *App) SetStarred(id string, starred bool) error {
	return a.store.SetStarred(id, starred)
}

func (a *App) ToggleStarred(id string) error {
	item, err := a.Item(id)
	if err != nil {
		return err
	}
	return a.store.SetStarred(id, !item.Starred)
}

func (a *App) Archive(id string) error {
	return a.store.SetArchived(id, true)
}

func (a *App) Storage() (StorageStats, error) {
	return a.store.GetStorageStats(a.loaded.Paths.Database)
}

func (a *App) ReconcileStorage() (StorageReconcileResult, error) {
	items, err := a.Items(ItemFilter{AllItems: true})
	if err != nil {
		return StorageReconcileResult{}, err
	}
	expected := map[string]struct{}{}
	var missing []string
	for _, item := range items {
		expected[item.ContentPath] = struct{}{}
		if _, err := os.Stat(filepath.Join(a.loaded.Paths.ContentDir, item.ContentPath)); err != nil && errors.Is(err, os.ErrNotExist) {
			missing = append(missing, item.ContentPath)
		}
	}
	var scanned int
	var orphaned []string
	_ = filepath.WalkDir(a.loaded.Paths.ContentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		scanned++
		rel, err := filepath.Rel(a.loaded.Paths.ContentDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if _, ok := expected[rel]; !ok {
			orphaned = append(orphaned, rel)
		}
		return nil
	})
	if err := feedSync.ReconcileStorage(a.store, a.loaded.Paths); err != nil {
		return StorageReconcileResult{}, err
	}
	stats, err := a.Storage()
	return StorageReconcileResult{Storage: stats, ScannedFiles: scanned, MissingFiles: missing, OrphanedFiles: orphaned}, err
}

func (a *App) Status() (StatusSummary, error) {
	if err := a.ReconcileSources(context.Background()); err != nil {
		return StatusSummary{}, err
	}
	return a.store.Status(a.loaded.Paths.Database)
}

func (a *App) configSource(id string) (config.Source, error) {
	for _, src := range a.loaded.Sources {
		if src.ID == id {
			return src, nil
		}
	}
	return config.Source{}, AppError("source-not-found", "source not found", store.ErrNotFound)
}
