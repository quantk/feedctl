package metrics

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractHabrArticleID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
		ok   bool
	}{
		{
			name: "article url",
			url:  "https://habr.com/ru/articles/1033808/",
			want: "1033808",
			ok:   true,
		},
		{
			name: "company article url with rss query parameters",
			url:  "https://habr.com/ru/companies/kodik/articles/1032884/?utm_campaign=1032884&utm_source=habrahabr&utm_medium=rss",
			want: "1032884",
			ok:   true,
		},
		{
			name: "non habr url",
			url:  "https://example.com/ru/articles/1033808/",
			ok:   false,
		},
		{
			name: "habr url without article id",
			url:  "https://habr.com/ru/news/",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ExtractHabrArticleID(tt.url)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("ExtractHabrArticleID(%q) = %q, %v; want %q, %v", tt.url, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestHabrProviderFetchMapsStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kek/v2/articles/1033808/" {
			t.Fatalf("path=%q want %q", r.URL.Path, "/kek/v2/articles/1033808/")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"statistics": {
				"commentsCount": 0,
				"favoritesCount": 2,
				"readingCount": 23,
				"score": 0,
				"votesCount": 5
			}
		}`)
	}))
	defer server.Close()

	provider := HabrProvider{BaseURL: server.URL, Client: server.Client()}
	got, err := provider.Fetch(context.Background(), Candidate{URL: "https://habr.com/ru/articles/1033808/"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "habr" {
		t.Fatalf("provider=%q want habr", got.Provider)
	}
	assertIntPtr(t, "score", got.Score, 0)
	assertIntPtr(t, "comments", got.CommentsCount, 0)
	assertIntPtr(t, "votes", got.VotesCount, 5)
	assertIntPtr(t, "favorites", got.FavoritesCount, 2)
	assertIntPtr(t, "reading", got.ReadingCount, 23)
	if got.FetchedAt == "" {
		t.Fatal("FetchedAt is empty")
	}
}

func assertIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is nil, want %d", name, want)
	}
	if *got != want {
		t.Fatalf("%s=%d want %d", name, *got, want)
	}
}
