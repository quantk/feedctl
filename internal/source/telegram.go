package source

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"feedctl/internal/config"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown"
	"github.com/PuerkitoBio/goquery"
)

const DefaultTelegramMaxItems = 50

type TelegramAdapter struct {
	Client    *http.Client
	converter *htmltomarkdown.Converter
}

func NewTelegramAdapter() *TelegramAdapter {
	return &TelegramAdapter{
		Client:    &http.Client{Timeout: 15 * time.Second},
		converter: htmltomarkdown.NewConverter("", true, nil),
	}
}

func (a *TelegramAdapter) Fetch(ctx context.Context, src config.Source) (Feed, error) {
	if a == nil {
		a = NewTelegramAdapter()
	}
	client := a.Client
	if client == nil {
		client = http.DefaultClient
	}
	maxItems := src.MaxItems
	if maxItems <= 0 {
		maxItems = DefaultTelegramMaxItems
	}
	feedURL := strings.TrimSpace(src.URL)
	if feedURL == "" {
		return Feed{}, fmt.Errorf("telegram source url is required")
	}

	items := make([]Item, 0, maxItems)
	metadata := Metadata{FeedURL: feedURL, URL: feedURL}
	nextURL := feedURL
	seenPages := map[string]struct{}{}
	for nextURL != "" && len(items) < maxItems {
		if _, seen := seenPages[nextURL]; seen {
			break
		}
		seenPages[nextURL] = struct{}{}
		doc, err := a.fetchDocument(ctx, client, nextURL)
		if err != nil {
			return Feed{}, err
		}
		pageMeta, pageItems, beforeURL, err := a.parseDocument(src, doc, nextURL)
		if err != nil {
			return Feed{}, err
		}
		if metadata.Title == "" {
			metadata.Title = pageMeta.Title
		}
		if metadata.URL == "" {
			metadata.URL = pageMeta.URL
		}
		for _, item := range pageItems {
			if len(items) >= maxItems {
				break
			}
			items = append(items, item)
		}
		nextURL = beforeURL
	}
	metadata.ItemsFound = len(items)
	return Feed{Metadata: metadata, Items: items}, nil
}

func (a *TelegramAdapter) Test(ctx context.Context, src config.Source) (Metadata, error) {
	feed, err := a.Fetch(ctx, src)
	if err != nil {
		return Metadata{}, err
	}
	return feed.Metadata, nil
}

func (a *TelegramAdapter) fetchDocument(ctx context.Context, client *http.Client, pageURL string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("telegram request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("telegram fetch: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("telegram fetch: status %d", res.StatusCode)
	}
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("telegram parse html: %w", err)
	}
	return doc, nil
}

func (a *TelegramAdapter) parseDocument(src config.Source, doc *goquery.Document, pageURL string) (Metadata, []Item, string, error) {
	metadata := Metadata{Title: telegramPageTitle(doc), URL: pageURL, FeedURL: src.URL}
	items := parseTelegramItems(src, doc, a.converter)
	if len(items) == 0 {
		return metadata, nil, "", fmt.Errorf("telegram parse: no posts found")
	}
	return metadata, items, telegramBeforeURL(doc, pageURL), nil
}

func telegramPageTitle(doc *goquery.Document) string {
	title := strings.TrimSpace(doc.Find("title").First().Text())
	for _, suffix := range []string{" – Telegram", " - Telegram", " — Telegram"} {
		title = strings.TrimSuffix(title, suffix)
	}
	return strings.TrimSpace(title)
}

func parseTelegramItems(src config.Source, doc *goquery.Document, converter *htmltomarkdown.Converter) []Item {
	var items []Item
	doc.Find(".tgme_widget_message[data-post]").Each(func(_ int, message *goquery.Selection) {
		dataPost, ok := message.Attr("data-post")
		dataPost = strings.TrimSpace(dataPost)
		if !ok || dataPost == "" || !strings.Contains(dataPost, "/") {
			return
		}
		postURL := "https://t.me/" + dataPost
		textSel := message.Find(".tgme_widget_message_text").First()
		body := telegramMessageMarkdown(textSel, converter)
		title := telegramTitle(body, dataPost)
		if strings.TrimSpace(body) == "" {
			body = fmt.Sprintf("[Open Telegram post](%s)", postURL)
		}
		published := telegramMessageTime(message)
		items = append(items, Item{
			SourceID:     src.ID,
			SourceName:   src.Name,
			SourceType:   src.Type,
			Title:        title,
			URL:          postURL,
			CanonicalURL: postURL,
			PublishedAt:  published,
			Body:         body,
			Author:       normalizeWhitespace(message.Find(".tgme_widget_message_owner_name").First().Text()),
			GUID:         dataPost,
			Tags:         append([]string(nil), src.Tags...),
		})
	})
	return items
}

func telegramMessageMarkdown(textSel *goquery.Selection, converter *htmltomarkdown.Converter) string {
	if textSel == nil || textSel.Length() == 0 {
		return ""
	}
	html, err := textSel.Html()
	if err != nil || strings.TrimSpace(html) == "" {
		return normalizeWhitespace(textSel.Text())
	}
	if converter == nil {
		converter = htmltomarkdown.NewConverter("", true, nil)
	}
	md, err := converter.ConvertString(html)
	if err != nil {
		return normalizeWhitespace(textSel.Text())
	}
	return strings.TrimSpace(md)
}

func telegramTitle(text, dataPost string) string {
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line = normalizeWhitespace(strings.Trim(line, "#*_`> -"))
		if line == "" {
			continue
		}
		if len([]rune(line)) > 80 {
			runes := []rune(line)
			line = string(runes[:80]) + "…"
		}
		return line
	}
	parts := strings.Split(dataPost, "/")
	return "Telegram post " + parts[len(parts)-1]
}

func telegramMessageTime(message *goquery.Selection) *time.Time {
	value, ok := message.Find("time[datetime]").First().Attr("datetime")
	if !ok || strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func telegramBeforeURL(doc *goquery.Document, pageURL string) string {
	var before string
	doc.Find("a[href*='before=']").EachWithBreak(func(_ int, link *goquery.Selection) bool {
		href, ok := link.Attr("href")
		if !ok || !strings.Contains(href, "before=") {
			return true
		}
		before = href
		return false
	})
	if before == "" {
		return ""
	}
	return resolveReference(pageURL, before)
}

func resolveReference(baseURL, ref string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ref
	}
	rel, err := url.Parse(ref)
	if err != nil {
		return ref
	}
	return base.ResolveReference(rel).String()
}

func normalizeWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
