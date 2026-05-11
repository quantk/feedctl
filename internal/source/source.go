package source

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"feedctl/internal/config"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/mmcdole/gofeed"
)

type Metadata struct {
	Title      string `json:"title"`
	URL        string `json:"url"`
	FeedURL    string `json:"feed_url"`
	ItemsFound int    `json:"items_found"`
}

type Feed struct {
	Metadata Metadata `json:"metadata"`
	Items    []Item   `json:"items"`
}

type Item struct {
	SourceID     string
	SourceName   string
	SourceType   string
	Title        string
	URL          string
	CanonicalURL string
	PublishedAt  *time.Time
	UpdatedAt    *time.Time
	Body         string
	Author       string
	GUID         string
	Tags         []string
}

type Adapter interface {
	Fetch(ctx context.Context, src config.Source) (Feed, error)
	Test(ctx context.Context, src config.Source) (Metadata, error)
}

type RSSAdapter struct {
	parser    *gofeed.Parser
	converter *htmltomarkdown.Converter
}

func NewRSSAdapter() *RSSAdapter {
	return &RSSAdapter{
		parser:    gofeed.NewParser(),
		converter: htmltomarkdown.NewConverter("", true, nil),
	}
}

func (a *RSSAdapter) Fetch(ctx context.Context, src config.Source) (Feed, error) {
	feed, err := a.parser.ParseURLWithContext(src.URL, ctx)
	if err != nil {
		return Feed{}, fmt.Errorf("parse feed: %w", err)
	}
	out := Feed{
		Metadata: Metadata{
			Title:      strings.TrimSpace(feed.Title),
			URL:        strings.TrimSpace(feed.Link),
			FeedURL:    src.URL,
			ItemsFound: len(feed.Items),
		},
		Items: make([]Item, 0, len(feed.Items)),
	}
	for _, item := range feed.Items {
		out.Items = append(out.Items, a.normalizeItem(src, item))
	}
	return out, nil
}

func (a *RSSAdapter) Test(ctx context.Context, src config.Source) (Metadata, error) {
	feed, err := a.Fetch(ctx, src)
	if err != nil {
		return Metadata{}, err
	}
	return feed.Metadata, nil
}

func (a *RSSAdapter) normalizeItem(src config.Source, item *gofeed.Item) Item {
	body := strings.TrimSpace(item.Content)
	if body == "" {
		body = strings.TrimSpace(item.Description)
	}
	body = a.toMarkdown(body)
	var author string
	if item.Author != nil {
		author = item.Author.Name
	}
	canonical := ""
	if item.Custom != nil {
		for _, key := range []string{"canonical", "canonical_url", "origLink"} {
			if v := strings.TrimSpace(item.Custom[key]); v != "" {
				canonical = v
				break
			}
		}
	}
	if canonical == "" {
		canonical = strings.TrimSpace(item.Link)
	}
	return Item{
		SourceID:     src.ID,
		SourceName:   src.Name,
		SourceType:   src.Type,
		Title:        strings.TrimSpace(item.Title),
		URL:          strings.TrimSpace(item.Link),
		CanonicalURL: canonical,
		PublishedAt:  item.PublishedParsed,
		UpdatedAt:    item.UpdatedParsed,
		Body:         body,
		Author:       author,
		GUID:         strings.TrimSpace(item.GUID),
		Tags:         append([]string(nil), src.Tags...),
	}
}

func (a *RSSAdapter) toMarkdown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		md, err := a.converter.ConvertString(value)
		if err == nil {
			return strings.TrimSpace(md)
		}
	}
	return value
}

func Identity(srcID string, item Item) (identity string, kind string) {
	if item.GUID != "" {
		return item.GUID, "guid"
	}
	if item.CanonicalURL != "" {
		return item.CanonicalURL, "canonical_url"
	}
	if item.URL != "" {
		return item.URL, "url"
	}
	published := ""
	if item.PublishedAt != nil {
		published = item.PublishedAt.UTC().Format(time.RFC3339)
	}
	raw := srcID + "\x00" + strings.ToLower(strings.TrimSpace(item.Title)) + "\x00" + published
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:]), "fingerprint"
}
