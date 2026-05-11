package config

import (
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
