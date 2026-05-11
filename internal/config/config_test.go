package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
)

func TestLoadDefaultsAndEnvPaths(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, "cfg")
	dataRoot := filepath.Join(home, "data")
	t.Setenv("FEEDCTL_CONFIG_DIR", cfgDir)
	t.Setenv("FEEDCTL_DATA_ROOT", dataRoot)
	loaded, err := LoadForHome(home)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Paths.ConfigFile != filepath.Join(cfgDir, "config.toml") {
		t.Fatalf("config path = %s", loaded.Paths.ConfigFile)
	}
	if loaded.Paths.SourcesDir != filepath.Join(cfgDir, "sources.d") {
		t.Fatalf("sources path = %s", loaded.Paths.SourcesDir)
	}
	if loaded.Paths.Database != filepath.Join(dataRoot, "feedctl.db") {
		t.Fatalf("database path = %s", loaded.Paths.Database)
	}
}

func TestLoadSourceValidation(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, "cfg")
	t.Setenv("FEEDCTL_CONFIG_DIR", cfgDir)
	if err := os.MkdirAll(filepath.Join(cfgDir, "sources.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(cfgDir, "sources.d", "bad.toml")
	if err := os.WriteFile(path, []byte("id = \"Bad ID\"\ntype = \"rss\"\nurl = \"https://example.com/feed.xml\"\nenabled = true\nlast_sync_at = \"never\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadForHome(home)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoadTelegramSourceWithMaxItems(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, "cfg")
	sourcesDir := filepath.Join(cfgDir, "sources.d")
	t.Setenv("FEEDCTL_CONFIG_DIR", cfgDir)
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sourcesDir, "telegram.toml")
	data := []byte("id = \"tg-example\"\ntype = \"telegram\"\nname = \"Example Channel\"\nurl = \"https://t.me/s/example\"\nenabled = true\ninterval = \"15m\"\ntags = [\"telegram\", \"ai\"]\nmax_items = 75\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadForHome(home)
	if err != nil {
		t.Fatalf("load telegram source: %v", err)
	}
	if len(loaded.Sources) != 1 {
		t.Fatalf("sources len=%d", len(loaded.Sources))
	}
	src := loaded.Sources[0]
	if src.Type != SourceTypeTelegram || src.MaxItems != 75 || src.URL != "https://t.me/s/example" {
		t.Fatalf("bad telegram source: %#v", src)
	}
}

func TestLoadRejectsUnsupportedSourceType(t *testing.T) {
	home := t.TempDir()
	cfgDir := filepath.Join(home, "cfg")
	sourcesDir := filepath.Join(cfgDir, "sources.d")
	t.Setenv("FEEDCTL_CONFIG_DIR", cfgDir)
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sourcesDir, "bad.toml")
	data := []byte("id = \"bad\"\ntype = \"html\"\nname = \"Bad\"\nurl = \"https://example.com\"\nenabled = true\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadForHome(home)
	if err == nil {
		t.Fatal("expected unsupported source type error")
	}
	var validationErrs ValidationErrors
	if !errors.As(err, &validationErrs) {
		t.Fatalf("expected validation errors, got %T %v", err, err)
	}
	if len(validationErrs) != 1 || validationErrs[0].Code != "unsupported-source-type" {
		t.Fatalf("expected unsupported-source-type, got %#v", validationErrs)
	}
}

func TestMarshalSourceUsesValidTOML(t *testing.T) {
	b, err := MarshalSource(Source{ID: "habr", Type: "rss", Name: "Habr", URL: "https://example.com/feed.xml", Enabled: true, Interval: "10m", Tags: []string{"tech", "ru"}})
	if err != nil {
		t.Fatal(err)
	}
	var src Source
	if err := tomlUnmarshalForTest(b, &src); err != nil {
		t.Fatalf("invalid toml: %v\n%s", err, string(b))
	}
	if src.ID != "habr" || src.Interval != "10m" || len(src.Tags) != 2 {
		t.Fatalf("bad source: %#v", src)
	}
}

func TestLoadUsesCurrentUserHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("FEEDCTL_CONFIG_DIR", filepath.Join(home, "config"))
	t.Setenv("FEEDCTL_DATA_ROOT", filepath.Join(home, "data"))

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Paths.HomeDir != home || loaded.Paths.DataRoot != filepath.Join(home, "data") {
		t.Fatalf("paths=%#v", loaded.Paths)
	}
}

func TestConfigDirectoryHelpersWriteSourceAndFormatExisting(t *testing.T) {
	home := t.TempDir()
	paths := DefaultPaths(home, DefaultConfig(home))

	if err := EnsureConfigDirs(paths); err != nil {
		t.Fatal(err)
	}
	if err := EnsureRuntimeDirs(paths); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{paths.SourcesDir, paths.DataRoot, filepath.Dir(paths.Database), paths.ContentDir, paths.VersionsDir, paths.TmpDir, paths.LogsDir} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("dir %s info=%v err=%v", dir, info, err)
		}
	}

	sourcePath := SourcePath(paths.SourcesDir, "example")
	if sourcePath != filepath.Join(paths.SourcesDir, "example.toml") {
		t.Fatalf("source path=%s", sourcePath)
	}
	if err := WriteSource(sourcePath, Source{ID: "example", Type: SourceTypeRSS, Name: "Example", URL: "https://example.com/feed.xml", Enabled: true, Tags: []string{"tech"}}); err != nil {
		t.Fatal(err)
	}

	sources, err := LoadSources(paths.SourcesDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sources) != 1 || sources[0].ID != "example" || sources[0].FilePath != sourcePath {
		t.Fatalf("sources=%#v", sources)
	}

	cfg := DefaultConfig(home)
	cfg.Sync.Concurrency = 2
	if err := os.WriteFile(paths.ConfigFile, []byte("[sync]\nconcurrency = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded := Loaded{Config: cfg, Sources: sources, Paths: paths}
	if err := FormatExisting(loaded); err != nil {
		t.Fatal(err)
	}
	formatted, err := LoadForHome(home)
	if err != nil {
		t.Fatal(err)
	}
	if formatted.Config.Sync.Concurrency != 2 || len(formatted.Sources) != 1 || formatted.Sources[0].ID != "example" {
		t.Fatalf("formatted=%#v", formatted)
	}
}

func TestConfigValueHelpers(t *testing.T) {
	if got, err := ParseDuration(""); err != nil || got != 5*time.Minute {
		t.Fatalf("ParseDuration empty=%v err=%v", got, err)
	}
	if got, err := ParseDuration("2h"); err != nil || got != 2*time.Hour {
		t.Fatalf("ParseDuration 2h=%v err=%v", got, err)
	}
	if _, err := ParseDuration("not-a-duration"); err == nil {
		t.Fatal("expected invalid duration error")
	}

	tests := []struct {
		name string
		in   string
		url  string
		want string
	}{
		{name: "name", in: "My Cool Feed!", want: "my-cool-feed"},
		{name: "host fallback", url: "https://www.example.com/feed.xml", want: "example-com"},
		{name: "empty fallback", want: "source"},
		{name: "non ascii fallback", in: "Искусственный интеллект 0", want: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateSourceID(tt.in, tt.url); got != tt.want {
				t.Fatalf("GenerateSourceID=%q want %q", got, tt.want)
			}
		})
	}

	validIDs := []string{"a", "a-b", "a_b", "a0", "0"}
	for _, id := range validIDs {
		if !ValidateSourceID(id) {
			t.Fatalf("ValidateSourceID(%q)=false", id)
		}
	}
	invalidIDs := []string{"", "Bad", "-bad", "bad space", "кириллица"}
	for _, id := range invalidIDs {
		if ValidateSourceID(id) {
			t.Fatalf("ValidateSourceID(%q)=true", id)
		}
	}
}

func TestApplyDefaultsAndValidationErrorStrings(t *testing.T) {
	home := t.TempDir()
	cfg := Config{}
	applyDefaults(&cfg, home)
	if cfg.Data.Root == "" || cfg.Data.Database == "" || cfg.Sources.Dir == "" || cfg.Sync.DefaultInterval != "5m" || cfg.Sync.Concurrency != 4 || cfg.TUI.Editor == "" || cfg.TUI.Browser == "" || cfg.Markdown.PathTemplate == "" {
		t.Fatalf("defaults=%#v", cfg)
	}

	cases := []struct {
		name string
		err  ValidationError
		want string
	}{
		{name: "path and field", err: ValidationError{Path: "file.toml", Field: "source.id", Message: "bad id"}, want: "file.toml source.id: bad id"},
		{name: "path only", err: ValidationError{Path: "file.toml", Message: "bad file"}, want: "file.toml: bad file"},
		{name: "message only", err: ValidationError{Message: "bad"}, want: "bad"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error()=%q want %q", got, tt.want)
			}
		})
	}

	errs := ValidationErrors{{Path: "a.toml", Message: "first"}, {Path: "b.toml", Field: "id", Message: "second"}}
	if got, want := errs.Error(), "a.toml: first; b.toml id: second"; got != want {
		t.Fatalf("ValidationErrors.Error()=%q want %q", got, want)
	}
	if got := (ValidationErrors{}).Error(); got != "" {
		t.Fatalf("empty ValidationErrors=%q want empty", got)
	}
}

func tomlUnmarshalForTest(b []byte, v any) error {
	return toml.Unmarshal(b, v)
}
