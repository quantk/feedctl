package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"feedctl/internal/config"
)

func TestRSSAdapterFetchTestAndNormalizeItems(t *testing.T) {
	feedXML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel>
<title>Example Feed</title>
<link>https://example.com/</link>
<item>
  <guid>guid-1</guid>
  <title> First Item </title>
  <link>https://example.com/items/1</link>
  <pubDate>Mon, 02 Jan 2006 15:04:05 GMT</pubDate>
  <description><![CDATA[<p>Hello <strong>RSS</strong></p>]]></description>
</item>
<item>
  <title>Second Item</title>
  <link>https://example.com/items/2</link>
  <description>Plain body</description>
</item>
</channel></rss>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(feedXML))
	}))
	defer server.Close()

	adapter := NewRSSAdapter()
	src := config.Source{ID: "example", Type: config.SourceTypeRSS, Name: "Example", URL: server.URL, Tags: []string{"rss", "test"}}
	feed, err := adapter.Fetch(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if feed.Metadata.Title != "Example Feed" || feed.Metadata.URL != "https://example.com/" || feed.Metadata.FeedURL != server.URL || feed.Metadata.ItemsFound != 2 {
		t.Fatalf("metadata=%#v", feed.Metadata)
	}
	if len(feed.Items) != 2 {
		t.Fatalf("items len=%d", len(feed.Items))
	}
	first := feed.Items[0]
	if first.SourceID != "example" || first.SourceName != "Example" || first.SourceType != config.SourceTypeRSS || first.Title != "First Item" || first.GUID != "guid-1" {
		t.Fatalf("first item=%#v", first)
	}
	if first.URL != "https://example.com/items/1" || first.CanonicalURL != first.URL {
		t.Fatalf("first urls=%#v", first)
	}
	if first.PublishedAt == nil || first.PublishedAt.UTC().Format(time.RFC3339) != "2006-01-02T15:04:05Z" {
		t.Fatalf("published=%v", first.PublishedAt)
	}
	if strings.Contains(first.Body, "<strong>") || !strings.Contains(first.Body, "Hello") || !strings.Contains(first.Body, "RSS") {
		t.Fatalf("body was not converted to markdown: %q", first.Body)
	}
	if len(first.Tags) != 2 || first.Tags[0] != "rss" || first.Tags[1] != "test" {
		t.Fatalf("tags=%v", first.Tags)
	}

	meta, err := adapter.Test(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != feed.Metadata.Title || meta.ItemsFound != 2 {
		t.Fatalf("test metadata=%#v", meta)
	}
}

func TestIdentityPreferenceAndFingerprint(t *testing.T) {
	published := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		item Item
		want string
		kind string
	}{
		{name: "guid", item: Item{GUID: "guid-1", CanonicalURL: "https://canonical", URL: "https://url"}, want: "guid-1", kind: "guid"},
		{name: "canonical", item: Item{CanonicalURL: "https://canonical", URL: "https://url"}, want: "https://canonical", kind: "canonical_url"},
		{name: "url", item: Item{URL: "https://url"}, want: "https://url", kind: "url"},
		{name: "fingerprint", item: Item{Title: "Same", PublishedAt: &published}, kind: "fingerprint"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, kind := Identity("source", tt.item)
			if kind != tt.kind {
				t.Fatalf("kind=%q want %q", kind, tt.kind)
			}
			if tt.want != "" && got != tt.want {
				t.Fatalf("identity=%q want %q", got, tt.want)
			}
			if tt.kind == "fingerprint" && len(got) != 64 {
				t.Fatalf("fingerprint len=%d value=%q", len(got), got)
			}
		})
	}

	one, kind := Identity("source", Item{Title: "Same", PublishedAt: &published})
	two, _ := Identity("source", Item{Title: " same ", PublishedAt: &published})
	three, _ := Identity("source", Item{Title: "Different", PublishedAt: &published})
	if kind != "fingerprint" || one != two || one == three {
		t.Fatalf("fingerprint stability one=%s two=%s three=%s kind=%s", one, two, three, kind)
	}
}

func TestDefaultAdapterSupportsRSSAndTelegramTest(t *testing.T) {
	telegramPage := `<!doctype html><html><head><title>Channel – Telegram</title></head><body>
<div class="tgme_widget_message" data-post="channel/1">
  <a class="tgme_widget_message_date"><time datetime="2026-05-11T12:00:00+00:00"></time></a>
  <div class="tgme_widget_message_text">Hello Telegram</div>
</div>
</body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(telegramPage))
	}))
	defer server.Close()

	adapter := NewDefaultAdapter()
	meta, err := adapter.Test(context.Background(), config.Source{Type: config.SourceTypeTelegram, URL: server.URL, MaxItems: 1})
	if err != nil {
		t.Fatal(err)
	}
	if meta.Title != "Channel" || meta.ItemsFound != 1 {
		t.Fatalf("telegram meta=%#v", meta)
	}
}
