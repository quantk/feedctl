package source

import (
	"context"
	"strings"
	"testing"
	"time"

	"feedctl/internal/config"
	"feedctl/internal/testutil"
)

func TestTelegramAdapterFetchesMetadata(t *testing.T) {
	page := testutil.TelegramWebPage("LLM под капотом", "llm_under_hood", []testutil.TelegramPost{
		testutil.DefaultTelegramPost(831, `<b>First</b><br/>Body`),
		testutil.DefaultTelegramPost(832, `Second`),
	}, "")
	server := testutil.TelegramServer(t, "llm_under_hood", map[string]string{"/s/llm_under_hood": page})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-llm", Type: config.SourceTypeTelegram, Name: "LLM", URL: server.URL + "/s/llm_under_hood", Enabled: true})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	if feed.Metadata.Title != "LLM под капотом" || feed.Metadata.FeedURL != server.URL+"/s/llm_under_hood" || feed.Metadata.ItemsFound != 2 {
		t.Fatalf("bad metadata: %#v", feed.Metadata)
	}
}

func TestTelegramAdapterExtractsPostFields(t *testing.T) {
	page := testutil.TelegramWebPage("LLM под капотом", "llm_under_hood", []testutil.TelegramPost{
		{ID: 831, HTML: `<b>AI Ops</b><br/>Readable body`, Datetime: "2026-05-11T15:38:00+00:00"},
	}, "")
	server := testutil.TelegramServer(t, "llm_under_hood", map[string]string{"/s/llm_under_hood": page})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-llm", Type: config.SourceTypeTelegram, Name: "LLM", URL: server.URL + "/s/llm_under_hood", Enabled: true, Tags: []string{"telegram", "llm"}})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("items len=%d", len(feed.Items))
	}
	item := feed.Items[0]
	if item.SourceID != "tg-llm" || item.SourceType != config.SourceTypeTelegram || item.GUID != "llm_under_hood/831" {
		t.Fatalf("bad identity fields: %#v", item)
	}
	if item.URL != "https://t.me/llm_under_hood/831" || item.CanonicalURL != item.URL {
		t.Fatalf("bad urls: %#v", item)
	}
	wantPublished := time.Date(2026, 5, 11, 15, 38, 0, 0, time.UTC)
	if item.PublishedAt == nil || !item.PublishedAt.Equal(wantPublished) {
		t.Fatalf("published=%v want=%v", item.PublishedAt, wantPublished)
	}
	if item.Author != "LLM под капотом" || item.Title != "AI Ops" || len(item.Tags) != 2 {
		t.Fatalf("bad metadata fields: %#v", item)
	}
}

func TestTelegramAdapterConvertsRichTextToMarkdown(t *testing.T) {
	page := testutil.TelegramWebPage("Rich", "rich_channel", []testutil.TelegramPost{
		testutil.DefaultTelegramPost(10, `<b>Bold</b> and <i>italic</i><br/><blockquote>Quote</blockquote><br/><a href="https://example.com">link</a><br/>• one 😊`),
	}, "")
	server := testutil.TelegramServer(t, "rich_channel", map[string]string{"/s/rich_channel": page})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-rich", Type: config.SourceTypeTelegram, URL: server.URL + "/s/rich_channel", Enabled: true})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	body := feed.Items[0].Body
	for _, raw := range []string{"<b>", "<blockquote>", "tgme_widget_message"} {
		if strings.Contains(body, raw) {
			t.Fatalf("body contains raw html/chrome %q:\n%s", raw, body)
		}
	}
	for _, want := range []string{"Bold", "italic", "Quote", "[link](https://example.com)", "one", "😊"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func TestTelegramAdapterRepresentsMediaOnlyPost(t *testing.T) {
	media := `<a class="tgme_widget_message_photo_wrap" style="background-image:url('https://cdn.example/photo.jpg')"></a>`
	page := testutil.TelegramWebPage("Media", "media_channel", []testutil.TelegramPost{
		{ID: 44, Datetime: "2026-05-11T15:44:00+00:00", Media: media},
	}, "")
	server := testutil.TelegramServer(t, "media_channel", map[string]string{"/s/media_channel": page})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-media", Type: config.SourceTypeTelegram, URL: server.URL + "/s/media_channel", Enabled: true})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	item := feed.Items[0]
	if item.Title != "Telegram post 44" {
		t.Fatalf("title=%q", item.Title)
	}
	if !strings.Contains(item.Body, "https://t.me/media_channel/44") {
		t.Fatalf("media-only body should preserve post url: %q", item.Body)
	}
}

func TestTelegramAdapterPaginatesUntilMaxItems(t *testing.T) {
	first := testutil.TelegramWebPage("Paged", "paged_channel", []testutil.TelegramPost{
		testutil.DefaultTelegramPost(20, `Post 20`),
		testutil.DefaultTelegramPost(19, `Post 19`),
	}, "19")
	older := testutil.TelegramWebPage("Paged", "paged_channel", []testutil.TelegramPost{
		testutil.DefaultTelegramPost(18, `Post 18`),
		testutil.DefaultTelegramPost(17, `Post 17`),
	}, "17")
	server := testutil.TelegramServer(t, "paged_channel", map[string]string{
		"/s/paged_channel":           first,
		"/s/paged_channel?before=19": older,
	})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-paged", Type: config.SourceTypeTelegram, URL: server.URL + "/s/paged_channel", Enabled: true, MaxItems: 3})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	if len(feed.Items) != 3 {
		t.Fatalf("items len=%d", len(feed.Items))
	}
	got := []string{feed.Items[0].GUID, feed.Items[1].GUID, feed.Items[2].GUID}
	want := []string{"paged_channel/20", "paged_channel/19", "paged_channel/18"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("guids=%v want=%v", got, want)
		}
	}
}

func TestTelegramAdapterUsesDefaultMaxItems(t *testing.T) {
	posts := make([]testutil.TelegramPost, 0, DefaultTelegramMaxItems+5)
	for i := 1; i <= DefaultTelegramMaxItems+5; i++ {
		posts = append(posts, testutil.DefaultTelegramPost(i, `Post`))
	}
	page := testutil.TelegramWebPage("Default Limit", "default_limit", posts, "")
	server := testutil.TelegramServer(t, "default_limit", map[string]string{"/s/default_limit": page})

	feed, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-default", Type: config.SourceTypeTelegram, URL: server.URL + "/s/default_limit", Enabled: true})
	if err != nil {
		t.Fatalf("fetch telegram: %v", err)
	}
	if len(feed.Items) != DefaultTelegramMaxItems {
		t.Fatalf("items len=%d want=%d", len(feed.Items), DefaultTelegramMaxItems)
	}
}

func TestTelegramAdapterHTTPStatusError(t *testing.T) {
	server := testutil.TelegramServer(t, "missing_channel", map[string]string{})

	_, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-missing", Type: config.SourceTypeTelegram, URL: server.URL + "/s/missing_channel", Enabled: true})
	if err == nil {
		t.Fatal("expected status error")
	}
	if !strings.Contains(err.Error(), "telegram fetch: status 404") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTelegramAdapterFetchErrors(t *testing.T) {
	tests := []struct {
		name string
		page string
		want string
	}{
		{
			name: "empty public page",
			page: `<!doctype html><html><head><title>Empty – Telegram</title></head><body>No posts here</body></html>`,
			want: "telegram parse: no posts found",
		},
		{
			name: "malformed post identity",
			page: `<!doctype html><html><head><title>Bad – Telegram</title></head><body><div class="tgme_widget_message" data-post="bad"><div class="tgme_widget_message_text">Bad</div></div></body></html>`,
			want: "telegram parse: no posts found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.TelegramServer(t, "bad_channel", map[string]string{"/s/bad_channel": tt.page})
			_, err := NewTelegramAdapter().Fetch(context.Background(), config.Source{ID: "tg-bad", Type: config.SourceTypeTelegram, URL: server.URL + "/s/bad_channel", Enabled: true})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v want substring %q", err, tt.want)
			}
		})
	}
}
