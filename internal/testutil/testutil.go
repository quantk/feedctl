package testutil

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func IsolatedEnv(t *testing.T) (configDir, dataRoot string) {
	t.Helper()
	root := t.TempDir()
	configDir = filepath.Join(root, "config")
	dataRoot = filepath.Join(root, "data")
	t.Setenv("FEEDCTL_CONFIG_DIR", configDir)
	t.Setenv("FEEDCTL_DATA_ROOT", dataRoot)
	t.Setenv("EDITOR", "true")
	t.Setenv("BROWSER", "true")
	return configDir, dataRoot
}

func RSSFeed(title string, items ...RSSItem) string {
	out := `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel>`
	out += fmt.Sprintf("<title>%s</title><link>http://example.test/</link>", title)
	for _, item := range items {
		out += fmt.Sprintf("<item><guid>%s</guid><title>%s</title><link>%s</link><pubDate>%s</pubDate><description><![CDATA[%s]]></description></item>", item.GUID, item.Title, item.Link, item.PubDate, item.Body)
	}
	out += `</channel></rss>`
	return out
}

type RSSItem struct {
	GUID    string
	Title   string
	Link    string
	PubDate string
	Body    string
}

func DefaultItem(guid, title, body string) RSSItem {
	return RSSItem{GUID: guid, Title: title, Link: "http://example.test/" + guid, PubDate: "Mon, 02 Jan 2006 15:04:05 GMT", Body: body}
}

func FeedServer(t *testing.T, feed *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(*feed))
	}))
}
