package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func tomlUnmarshalForTest(b []byte, v any) error {
	return toml.Unmarshal(b, v)
}
