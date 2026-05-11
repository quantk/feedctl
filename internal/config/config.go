package config

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

const (
	DefaultPathTemplate = "{source_id}/{year}/{month}/{slug}.md"
	SourceTypeRSS       = "rss"
)

var (
	sourceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)
	forbiddenFields = map[string]struct{}{
		"last_sync_at":  {},
		"last_error":    {},
		"etag":          {},
		"last_modified": {},
		"cursor":        {},
		"items_count":   {},
		"read":          {},
		"unread":        {},
		"hash":          {},
		"hashes":        {},
		"version":       {},
		"versions":      {},
		"disk_usage":    {},
	}
)

type Config struct {
	Data     DataConfig     `toml:"data" json:"data"`
	Sources  SourcesConfig  `toml:"sources" json:"sources"`
	Sync     SyncConfig     `toml:"sync" json:"sync"`
	TUI      TUIConfig      `toml:"tui" json:"tui"`
	Markdown MarkdownConfig `toml:"markdown" json:"markdown"`
}

type DataConfig struct {
	Root        string `toml:"root" json:"root"`
	Database    string `toml:"database" json:"database"`
	ContentDir  string `toml:"content_dir" json:"content_dir"`
	VersionsDir string `toml:"versions_dir" json:"versions_dir"`
}

type SourcesConfig struct {
	Dir string `toml:"dir" json:"dir"`
}

type SyncConfig struct {
	DefaultInterval string `toml:"default_interval" json:"default_interval"`
	Concurrency     int    `toml:"concurrency" json:"concurrency"`
	SyncOnStartup   bool   `toml:"sync_on_startup" json:"sync_on_startup"`
}

type TUIConfig struct {
	Editor             string `toml:"editor" json:"editor"`
	Browser            string `toml:"browser" json:"browser"`
	ShowRemovedSources bool   `toml:"show_removed_sources" json:"show_removed_sources"`
}

type MarkdownConfig struct {
	Frontmatter  bool   `toml:"frontmatter" json:"frontmatter"`
	PathTemplate string `toml:"path_template" json:"path_template"`
}

type Source struct {
	ID       string   `toml:"id" json:"id"`
	Type     string   `toml:"type" json:"type"`
	Name     string   `toml:"name" json:"name"`
	URL      string   `toml:"url" json:"url"`
	Enabled  bool     `toml:"enabled" json:"enabled"`
	Interval string   `toml:"interval,omitempty" json:"interval,omitempty"`
	Tags     []string `toml:"tags,omitempty" json:"tags,omitempty"`
	FilePath string   `toml:"-" json:"file_path,omitempty"`
}

type Paths struct {
	HomeDir     string `json:"home_dir"`
	ConfigFile  string `json:"config_file"`
	SourcesDir  string `json:"sources_dir"`
	DataRoot    string `json:"data_root"`
	Database    string `json:"database"`
	ContentDir  string `json:"content_dir"`
	VersionsDir string `json:"versions_dir"`
	TmpDir      string `json:"tmp_dir"`
	LogsDir     string `json:"logs_dir"`
}

type Loaded struct {
	Config  Config
	Sources []Source
	Paths   Paths
}

type ValidationError struct {
	Path    string `json:"path,omitempty"`
	Field   string `json:"field,omitempty"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	if e.Path != "" && e.Field != "" {
		return fmt.Sprintf("%s %s: %s", e.Path, e.Field, e.Message)
	}
	if e.Path != "" {
		return fmt.Sprintf("%s: %s", e.Path, e.Message)
	}
	return e.Message
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	parts := make([]string, len(e))
	for i := range e {
		parts[i] = e[i].Error()
	}
	return strings.Join(parts, "; ")
}

func DefaultConfig(home string) Config {
	configRoot := filepath.Join(home, ".config", "feedctl")
	dataRoot := filepath.Join(home, ".feedctl")
	return Config{
		Data: DataConfig{
			Root:        dataRoot,
			Database:    filepath.Join(dataRoot, "feedctl.db"),
			ContentDir:  filepath.Join(dataRoot, "content"),
			VersionsDir: filepath.Join(dataRoot, "versions"),
		},
		Sources: SourcesConfig{Dir: filepath.Join(configRoot, "sources.d")},
		Sync: SyncConfig{
			DefaultInterval: "5m",
			Concurrency:     4,
			SyncOnStartup:   true,
		},
		TUI: TUIConfig{
			Editor:             getenvDefault("EDITOR", "nvim"),
			Browser:            getenvDefault("BROWSER", "xdg-open"),
			ShowRemovedSources: false,
		},
		Markdown: MarkdownConfig{Frontmatter: true, PathTemplate: DefaultPathTemplate},
	}
}

func DefaultPaths(home string, cfg Config) Paths {
	defaultConfigRoot := filepath.Join(home, ".config", "feedctl")
	defaultDataRoot := filepath.Join(home, ".feedctl")
	configRoot := defaultConfigRoot
	if env := os.Getenv("FEEDCTL_CONFIG_DIR"); env != "" {
		configRoot = expandPath(env, home)
	}
	dataRoot := expandPath(cfg.Data.Root, home)
	if dataRoot == "" {
		dataRoot = defaultDataRoot
	}
	if env := os.Getenv("FEEDCTL_DATA_ROOT"); env != "" {
		dataRoot = expandPath(env, home)
	}
	configFile := filepath.Join(configRoot, "config.toml")
	if env := os.Getenv("FEEDCTL_CONFIG_FILE"); env != "" {
		configFile = expandPath(env, home)
	}
	database := expandPath(cfg.Data.Database, home)
	contentDir := expandPath(cfg.Data.ContentDir, home)
	versionsDir := expandPath(cfg.Data.VersionsDir, home)
	defaultDatabase := filepath.Join(defaultDataRoot, "feedctl.db")
	defaultContentDir := filepath.Join(defaultDataRoot, "content")
	defaultVersionsDir := filepath.Join(defaultDataRoot, "versions")
	if cfg.Data.Database == "" || database == defaultDatabase || database == filepath.Join(defaultDataRoot, "feedctl.db") {
		database = filepath.Join(dataRoot, "feedctl.db")
	}
	if cfg.Data.ContentDir == "" || contentDir == defaultContentDir {
		contentDir = filepath.Join(dataRoot, "content")
	}
	if cfg.Data.VersionsDir == "" || versionsDir == defaultVersionsDir {
		versionsDir = filepath.Join(dataRoot, "versions")
	}
	sourcesDir := expandPath(cfg.Sources.Dir, home)
	defaultSourcesDir := filepath.Join(defaultConfigRoot, "sources.d")
	if cfg.Sources.Dir == "" || sourcesDir == defaultSourcesDir {
		sourcesDir = filepath.Join(configRoot, "sources.d")
	}
	return Paths{
		HomeDir:     home,
		ConfigFile:  configFile,
		SourcesDir:  sourcesDir,
		DataRoot:    dataRoot,
		Database:    database,
		ContentDir:  contentDir,
		VersionsDir: versionsDir,
		TmpDir:      filepath.Join(dataRoot, "tmp"),
		LogsDir:     filepath.Join(dataRoot, "logs"),
	}
}

func Load() (Loaded, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Loaded{}, err
	}
	return LoadForHome(home)
}

func LoadForHome(home string) (Loaded, error) {
	cfg := DefaultConfig(home)
	paths := DefaultPaths(home, cfg)

	if _, err := os.Stat(paths.ConfigFile); err == nil {
		b, err := os.ReadFile(paths.ConfigFile)
		if err != nil {
			return Loaded{}, err
		}
		if err := validateNoForbiddenFields(paths.ConfigFile, b); err != nil {
			return Loaded{}, err
		}
		if err := toml.Unmarshal(b, &cfg); err != nil {
			return Loaded{}, err
		}
		applyDefaults(&cfg, home)
		paths = DefaultPaths(home, cfg)
	} else if !errors.Is(err, os.ErrNotExist) {
		return Loaded{}, err
	}

	sources, err := LoadSources(paths.SourcesDir)
	if err != nil {
		return Loaded{}, err
	}
	loaded := Loaded{Config: cfg, Sources: sources, Paths: paths}
	if err := loaded.Validate(); err != nil {
		return Loaded{}, err
	}
	return loaded, nil
}

func LoadSources(dir string) ([]Source, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	sources := make([]Source, 0, len(files))
	for _, path := range files {
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if err := validateNoForbiddenFields(path, b); err != nil {
			return nil, err
		}
		var src Source
		if err := toml.Unmarshal(b, &src); err != nil {
			return nil, err
		}
		src.FilePath = path
		sources = append(sources, src)
	}
	return sources, nil
}

func (l Loaded) Validate() error {
	var errs ValidationErrors
	if l.Config.Sync.DefaultInterval != "" {
		if _, err := time.ParseDuration(l.Config.Sync.DefaultInterval); err != nil {
			errs = append(errs, ValidationError{Path: l.Paths.ConfigFile, Field: "sync.default_interval", Code: "invalid-duration", Message: err.Error()})
		}
	}
	if l.Config.Sync.Concurrency < 1 {
		errs = append(errs, ValidationError{Path: l.Paths.ConfigFile, Field: "sync.concurrency", Code: "invalid-concurrency", Message: "concurrency must be at least 1"})
	}
	seen := map[string]string{}
	for _, src := range l.Sources {
		if src.ID == "" {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "id", Code: "missing-source-id", Message: "source id is required"})
		} else if !sourceIDPattern.MatchString(src.ID) {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "id", Code: "invalid-source-id", Message: "source id must match ^[a-z0-9][a-z0-9_-]*$"})
		} else if prev, ok := seen[src.ID]; ok {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "id", Code: "duplicate-source-id", Message: fmt.Sprintf("source id already declared in %s", prev)})
		} else {
			seen[src.ID] = src.FilePath
		}
		if src.Type == "" {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "type", Code: "missing-source-type", Message: "source type is required"})
		} else if src.Type != SourceTypeRSS {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "type", Code: "unsupported-source-type", Message: "only rss sources are supported in the MVP"})
		}
		if src.URL == "" {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "url", Code: "missing-source-url", Message: "source url is required"})
		} else if _, err := url.ParseRequestURI(src.URL); err != nil {
			errs = append(errs, ValidationError{Path: src.FilePath, Field: "url", Code: "invalid-url", Message: err.Error()})
		}
		if src.Interval != "" {
			if _, err := time.ParseDuration(src.Interval); err != nil {
				errs = append(errs, ValidationError{Path: src.FilePath, Field: "interval", Code: "invalid-duration", Message: err.Error()})
			}
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func EnsureConfigDirs(paths Paths) error {
	return os.MkdirAll(paths.SourcesDir, 0o755)
}

func EnsureRuntimeDirs(paths Paths) error {
	for _, dir := range []string{paths.DataRoot, filepath.Dir(paths.Database), paths.ContentDir, paths.VersionsDir, paths.TmpDir, paths.LogsDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func WriteSource(path string, src Source) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := MarshalSource(src)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func MarshalSource(src Source) ([]byte, error) {
	src.FilePath = ""
	type sourceFile Source
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	enc.SetIndentTables(true)
	if err := enc.Encode(sourceFile(src)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func SourcePath(dir, id string) string {
	return filepath.Join(dir, id+".toml")
}

func ParseDuration(value string) (time.Duration, error) {
	if value == "" {
		value = "5m"
	}
	return time.ParseDuration(value)
}

func GenerateSourceID(name, rawURL string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
			base = strings.TrimPrefix(u.Host, "www.")
		}
	}
	if base == "" {
		base = "source"
	}
	base = strings.ToLower(base)
	var b strings.Builder
	lastDash := false
	for _, r := range base {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		id = "source"
	}
	return id
}

func ValidateSourceID(id string) bool {
	return sourceIDPattern.MatchString(id)
}

func FormatExisting(loaded Loaded) error {
	if _, err := os.Stat(loaded.Paths.ConfigFile); err == nil {
		data, err := toml.Marshal(loaded.Config)
		if err != nil {
			return err
		}
		if err := os.WriteFile(loaded.Paths.ConfigFile, data, 0o644); err != nil {
			return err
		}
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, src := range loaded.Sources {
		if err := WriteSource(src.FilePath, src); err != nil {
			return err
		}
	}
	return nil
}

func applyDefaults(cfg *Config, home string) {
	defaults := DefaultConfig(home)
	if cfg.Data.Root == "" {
		cfg.Data.Root = defaults.Data.Root
	}
	if cfg.Data.Database == "" {
		cfg.Data.Database = filepath.Join(expandPath(cfg.Data.Root, home), "feedctl.db")
	}
	if cfg.Data.ContentDir == "" {
		cfg.Data.ContentDir = filepath.Join(expandPath(cfg.Data.Root, home), "content")
	}
	if cfg.Data.VersionsDir == "" {
		cfg.Data.VersionsDir = filepath.Join(expandPath(cfg.Data.Root, home), "versions")
	}
	if cfg.Sources.Dir == "" {
		cfg.Sources.Dir = defaults.Sources.Dir
	}
	if cfg.Sync.DefaultInterval == "" {
		cfg.Sync.DefaultInterval = defaults.Sync.DefaultInterval
	}
	if cfg.Sync.Concurrency == 0 {
		cfg.Sync.Concurrency = defaults.Sync.Concurrency
	}
	if cfg.TUI.Editor == "" {
		cfg.TUI.Editor = defaults.TUI.Editor
	}
	if cfg.TUI.Browser == "" {
		cfg.TUI.Browser = defaults.TUI.Browser
	}
	if cfg.Markdown.PathTemplate == "" {
		cfg.Markdown.PathTemplate = defaults.Markdown.PathTemplate
	}
}

func expandPath(path, home string) string {
	if path == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return os.ExpandEnv(path)
}

func validateNoForbiddenFields(path string, b []byte) error {
	var raw map[string]any
	if err := toml.Unmarshal(b, &raw); err != nil {
		return err
	}
	var errs ValidationErrors
	walkMap(raw, "", func(field string) {
		key := field
		if i := strings.LastIndex(field, "."); i >= 0 {
			key = field[i+1:]
		}
		if _, forbidden := forbiddenFields[key]; forbidden {
			errs = append(errs, ValidationError{Path: path, Field: field, Code: "forbidden-runtime-field", Message: "runtime fields must be stored in SQLite, not config"})
		}
	})
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func walkMap(m map[string]any, prefix string, visit func(string)) {
	for k, v := range m {
		field := k
		if prefix != "" {
			field = prefix + "." + k
		}
		visit(field)
		if child, ok := v.(map[string]any); ok {
			walkMap(child, field, visit)
		}
	}
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
