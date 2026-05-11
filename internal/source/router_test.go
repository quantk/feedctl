package source

import (
	"context"
	"strings"
	"testing"

	"feedctl/internal/config"
)

type recordingAdapter struct {
	name    string
	fetched bool
	tested  bool
}

func (a *recordingAdapter) Fetch(ctx context.Context, src config.Source) (Feed, error) {
	_ = ctx
	a.fetched = true
	return Feed{Metadata: Metadata{Title: a.name, FeedURL: src.URL}}, nil
}

func (a *recordingAdapter) Test(ctx context.Context, src config.Source) (Metadata, error) {
	_ = ctx
	a.tested = true
	return Metadata{Title: a.name, FeedURL: src.URL}, nil
}

func TestAdapterRouterDispatchesBySourceType(t *testing.T) {
	rss := &recordingAdapter{name: "rss"}
	telegram := &recordingAdapter{name: "telegram"}
	router := NewAdapterRouter(map[string]Adapter{
		config.SourceTypeRSS:      rss,
		config.SourceTypeTelegram: telegram,
	})

	feed, err := router.Fetch(context.Background(), config.Source{Type: config.SourceTypeRSS, URL: "https://example.com/feed.xml"})
	if err != nil {
		t.Fatalf("fetch rss: %v", err)
	}
	if feed.Metadata.Title != "rss" || !rss.fetched || telegram.fetched {
		t.Fatalf("rss dispatch mismatch feed=%#v rss=%t telegram=%t", feed, rss.fetched, telegram.fetched)
	}

	meta, err := router.Test(context.Background(), config.Source{Type: config.SourceTypeTelegram, URL: "https://t.me/s/example"})
	if err != nil {
		t.Fatalf("test telegram: %v", err)
	}
	if meta.Title != "telegram" || !telegram.tested || rss.tested {
		t.Fatalf("telegram dispatch mismatch meta=%#v rss=%t telegram=%t", meta, rss.tested, telegram.tested)
	}
}

func TestAdapterRouterRejectsUnknownSourceType(t *testing.T) {
	router := NewAdapterRouter(map[string]Adapter{config.SourceTypeRSS: &recordingAdapter{name: "rss"}})

	_, err := router.Fetch(context.Background(), config.Source{Type: "html"})
	if err == nil {
		t.Fatal("expected unsupported source type error")
	}
	if !strings.Contains(err.Error(), "unsupported source type") || !strings.Contains(err.Error(), "html") {
		t.Fatalf("unexpected error: %v", err)
	}
}
