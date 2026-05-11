package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderMarkdownAndStableHash(t *testing.T) {
	h1 := StableHash("Title", "https://example.com", "", "Body\n")
	h2 := StableHash("Title", "https://example.com", "", "Body")
	if h1 != h2 {
		t.Fatalf("hash should ignore trailing whitespace: %s != %s", h1, h2)
	}
	md := string(RenderMarkdown(RenderItem{ID: "item_1", SourceID: "src", SourceName: "Source", SourceType: "rss", Title: "Title", URL: "https://example.com", FetchedAt: "now", ContentHash: h1, Version: 1, Tags: []string{"tech"}, Body: "Body"}, true))
	for _, want := range []string{"---", "id: \"item_1\"", "tags: [\"tech\"]", "# Title", "Body"} {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q:\n%s", want, md)
		}
	}
}

func TestPathsCollisionsSafeWriteAndVersion(t *testing.T) {
	when := time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC)
	rel := RenderPath("{source_id}/{year}/{month}/{slug}.md", PathData{SourceID: "src", Title: "Hello, World!", ItemID: "itemabcdef", Time: when})
	if rel != "src/2026/05/hello-world.md" {
		t.Fatalf("rel = %s", rel)
	}
	used := map[string]bool{rel: true}
	resolved, err := ResolveCollision(rel, "itemabcdef12345", func(candidate string) (bool, error) { return used[candidate], nil })
	if err != nil {
		t.Fatal(err)
	}
	if resolved == rel || !strings.Contains(resolved, "itemabcdef") {
		t.Fatalf("collision not resolved deterministically: %s", resolved)
	}
	root := t.TempDir()
	tmp := filepath.Join(root, "tmp")
	abs, size, err := SafeWrite(filepath.Join(root, "content"), rel, tmp, []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if size != 5 {
		t.Fatalf("size=%d", size)
	}
	if b, err := os.ReadFile(abs); err != nil || string(b) != "hello" {
		t.Fatalf("read %s: %s %v", abs, string(b), err)
	}
	versionRel, versionSize, err := SaveVersion(filepath.Join(root, "versions"), "item_1", 1, abs, tmp)
	if err != nil {
		t.Fatal(err)
	}
	if versionSize != 5 || versionRel != "item_1/v1.md" {
		t.Fatalf("version rel=%s size=%d", versionRel, versionSize)
	}
}
