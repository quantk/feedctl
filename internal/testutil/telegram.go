package testutil

import (
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type TelegramPost struct {
	ID       int
	HTML     string
	Datetime string
	Media    string
}

func DefaultTelegramPost(id int, htmlText string) TelegramPost {
	return TelegramPost{ID: id, HTML: htmlText, Datetime: fmt.Sprintf("2026-05-11T15:%02d:00+00:00", id%60)}
}

func TelegramWebPage(title, channel string, posts []TelegramPost, before string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString(" – Telegram</title></head><body>")
	for _, post := range posts {
		text := post.HTML
		if text != "" {
			text = `<div class="tgme_widget_message_text js-message_text" dir="auto">` + text + `</div>`
		}
		b.WriteString(fmt.Sprintf(`<div class="tgme_widget_message_wrap js-widget_message_wrap"><div class="tgme_widget_message" data-post="%s/%d"><div class="tgme_widget_message_author"><span class="tgme_widget_message_owner_name">%s</span></div>%s%s<a class="tgme_widget_message_date" href="https://t.me/%s/%d"><time datetime="%s"></time></a><span class="tgme_widget_message_views">1K</span></div></div>`, channel, post.ID, html.EscapeString(title), text, post.Media, channel, post.ID, post.Datetime))
	}
	if before != "" {
		b.WriteString(`<a class="tme_messages_more" href="?before=` + html.EscapeString(before) + `">older</a>`)
	}
	b.WriteString("</body></html>")
	return b.String()
}

func TelegramServer(t *testing.T, channel string, pages map[string]string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		page, ok := pages[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	}))
	t.Cleanup(server.Close)
	return server
}
