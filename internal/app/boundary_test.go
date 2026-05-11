package app_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCLIAndTUIDoNotDependOnConcreteStoreBoundary(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	for _, dir := range []string{"internal/cli", "internal/tui"} {
		entries, err := os.ReadDir(filepath.Join(repoRoot, dir))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			path := filepath.Join(repoRoot, dir, entry.Name())
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "\"feedctl/internal/store\"") {
				t.Fatalf("%s imports concrete store boundary", path)
			}
		}
	}
}

func TestAppConcreteDependenciesAreNotPublicFields(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	b, err := os.ReadFile(filepath.Join(repoRoot, "internal/app/app.go"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(b)
	for _, forbidden := range []string{"Loaded config.Loaded", "Store  *store.DB", "Store *store.DB"} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("App exposes concrete dependency field %q", forbidden)
		}
	}
}
