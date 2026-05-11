package content

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type RenderItem struct {
	ID           string
	SourceID     string
	SourceName   string
	SourceType   string
	Title        string
	URL          string
	CanonicalURL string
	PublishedAt  string
	FetchedAt    string
	ContentHash  string
	Version      int
	Tags         []string
	Body         string
}

type PathData struct {
	SourceID string
	Title    string
	ItemID   string
	Time     time.Time
}

func RenderMarkdown(item RenderItem, frontmatter bool) []byte {
	var b strings.Builder
	if frontmatter {
		b.WriteString("---\n")
		writeYAML(&b, "id", item.ID)
		writeYAML(&b, "source_id", item.SourceID)
		writeYAML(&b, "source_name", item.SourceName)
		writeYAML(&b, "source_type", item.SourceType)
		writeYAML(&b, "title", item.Title)
		writeYAML(&b, "url", item.URL)
		writeYAML(&b, "canonical_url", item.CanonicalURL)
		writeYAML(&b, "published_at", item.PublishedAt)
		writeYAML(&b, "fetched_at", item.FetchedAt)
		writeYAML(&b, "content_hash", item.ContentHash)
		b.WriteString("version: ")
		b.WriteString(strconv.Itoa(item.Version))
		b.WriteByte('\n')
		b.WriteString("tags: [")
		for i, tag := range item.Tags {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(yamlQuote(tag))
		}
		b.WriteString("]\n")
		b.WriteString("---\n\n")
	}
	if strings.TrimSpace(item.Title) != "" {
		b.WriteString("# ")
		b.WriteString(strings.TrimSpace(item.Title))
		b.WriteString("\n\n")
	}
	body := strings.TrimSpace(item.Body)
	if body == "" {
		body = "_No content._"
	}
	b.WriteString(body)
	b.WriteByte('\n')
	return []byte(b.String())
}

func StableHash(title, url, canonicalURL, body string) string {
	h := sha256.New()
	_, _ = io.WriteString(h, strings.TrimSpace(title))
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, strings.TrimSpace(url))
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, strings.TrimSpace(canonicalURL))
	_, _ = io.WriteString(h, "\n")
	_, _ = io.WriteString(h, normalizeBody(body))
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func ShortID(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])[:12]
}

func RenderPath(template string, data PathData) string {
	if template == "" {
		template = "{source_id}/{year}/{month}/{slug}.md"
	}
	when := data.Time
	if when.IsZero() {
		when = time.Now().UTC()
	}
	slug := Slug(data.Title)
	if slug == "" {
		slug = "item"
	}
	short := data.ItemID
	if len(short) > 12 {
		short = short[:12]
	}
	repl := map[string]string{
		"{source_id}":     data.SourceID,
		"{year}":          fmt.Sprintf("%04d", when.Year()),
		"{month}":         fmt.Sprintf("%02d", int(when.Month())),
		"{day}":           fmt.Sprintf("%02d", when.Day()),
		"{slug}":          slug,
		"{item_id}":       data.ItemID,
		"{item_id_short}": short,
	}
	out := template
	for k, v := range repl {
		out = strings.ReplaceAll(out, k, v)
	}
	return cleanRel(out)
}

func CollisionPath(rel, itemID string, attempt int) string {
	dir := filepath.Dir(rel)
	base := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
	ext := filepath.Ext(rel)
	short := itemID
	if len(short) > 12 {
		short = short[:12]
	}
	if attempt <= 1 {
		return cleanRel(filepath.Join(dir, base+"-"+short+ext))
	}
	return cleanRel(filepath.Join(dir, fmt.Sprintf("%s-%s-%d%s", base, short, attempt, ext)))
}

func ResolveCollision(initialRel, itemID string, assigned func(string) (bool, error)) (string, error) {
	rel := cleanRel(initialRel)
	for attempt := 0; attempt < 100; attempt++ {
		candidate := rel
		if attempt > 0 {
			candidate = CollisionPath(rel, itemID, attempt)
		}
		used, err := assigned(candidate)
		if err != nil {
			return "", err
		}
		if !used {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("could not resolve content path collision for %s", initialRel)
}

func SafeWrite(root, rel, tmpDir string, data []byte) (string, int64, error) {
	rel = cleanRel(rel)
	if rel == "." || strings.HasPrefix(rel, "../") || filepath.IsAbs(rel) {
		return "", 0, fmt.Errorf("unsafe relative path %q", rel)
	}
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp(tmpDir, "feedctl-*.tmp")
	if err != nil {
		return "", 0, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, err
	}
	if err := os.Rename(tmpName, abs); err != nil {
		return "", 0, err
	}
	return abs, int64(len(data)), nil
}

func SaveVersion(versionsRoot, itemID string, version int, currentPath string, tmpDir string) (string, int64, error) {
	b, err := os.ReadFile(currentPath)
	if err != nil {
		return "", 0, err
	}
	rel := filepath.Join(itemID, fmt.Sprintf("v%d.md", version))
	_, size, err := SafeWrite(versionsRoot, rel, tmpDir, b)
	if err != nil {
		return "", 0, err
	}
	return cleanRel(rel), size, nil
}

func Slug(value string) string {
	value = strings.ToLower(html.UnescapeString(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func normalizeBody(body string) string {
	lines := strings.Split(strings.TrimSpace(body), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}
	return strings.Join(lines, "\n")
}

func writeYAML(b *strings.Builder, key, value string) {
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(yamlQuote(value))
	b.WriteByte('\n')
}

func yamlQuote(value string) string {
	return strconv.Quote(value)
}

var repeatedSlash = regexp.MustCompile(`/+`)

func cleanRel(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	path = repeatedSlash.ReplaceAllString(path, "/")
	path = strings.TrimPrefix(path, "./")
	return path
}
